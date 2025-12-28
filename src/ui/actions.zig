//! Action handling for selected repositories

const std = @import("std");
const types = @import("../core/types.zig");

pub fn executeAction(allocator: std.mem.Allocator, action: types.Action, repo: types.Repository) !void {
    switch (action) {
        .clone => {
            const argv = [_][]const u8{ "gh", "repo", "clone", repo.nameWithOwner };
            var child = std.process.Child.init(&argv, allocator);
            _ = try child.spawnAndWait();
        },
        .copy_name => {
            try copyToClipboard(allocator, repo.nameWithOwner);
        },
        .copy_url => {
            const url = try std.fmt.allocPrint(allocator, "https://github.com/{s}", .{repo.nameWithOwner});
            defer allocator.free(url);
            try copyToClipboard(allocator, url);
        },
        .open_browser => {
            const argv = [_][]const u8{ "gh", "repo", "view", "--web", repo.nameWithOwner };
            var child = std.process.Child.init(&argv, allocator);
            _ = try child.spawnAndWait();
        },
        .show_name => {
            const stdout = std.io.getStdOut().writer();
            try stdout.print("{s}\n", .{repo.nameWithOwner});
        },
    }
}

fn copyToClipboard(allocator: std.mem.Allocator, text: []const u8) !void {
    const stdout = std.io.getStdOut().writer();
    
    // Try list of clipboard commands
    const commands = [_][]const []const u8{
        &[_][]const u8{ "pbcopy" },                  // macOS
        &[_][]const u8{ "wl-copy" },                 // Wayland
        &[_][]const u8{ "xclip", "-selection", "c" },// X11
        &[_][]const u8{ "clip.exe" },                // WSL
    };
    
    for (commands) |argv| {
        var child = std.process.Child.init(argv, allocator);
        child.stdin_behavior = .Pipe;
        child.stdout_behavior = .Ignore;
        child.stderr_behavior = .Ignore;
        
        if (child.spawn()) |_| {
            // Write to stdin
            if (child.stdin) |*stdin| {
                try stdin.writeAll(text);
                stdin.close();
            }
            _ = child.wait() catch continue;
            try stdout.print("\nCopied to clipboard!\n", .{});
            return;
        } else |_| {
            continue;
        }
    }
    
    try stdout.print("\nError: No clipboard tool found (pbcopy, wl-copy, xclip, clip.exe).\n", .{});
}
