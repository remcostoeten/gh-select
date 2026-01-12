//! Action handling for selected repositories

const std = @import("std");
const types = @import("../core/types.zig");

fn printToStdout(comptime fmt: []const u8, args: anytype) !void {
    const stdout_file = std.fs.File{ .handle = std.posix.STDOUT_FILENO };
    var buf: [1024]u8 = undefined;
    var writer = stdout_file.writer(&buf);
    try writer.interface.print(fmt, args);
    try writer.interface.flush();
}

pub fn executeAction(allocator: std.mem.Allocator, action: types.Action, repo: types.Repository) !void {
    const stderr = std.io.getStdErr().writer();
    
    switch (action) {
        .clone => {
            const argv = [_][]const u8{ "gh", "repo", "clone", repo.nameWithOwner };
            var child = std.process.Child.init(&argv, allocator);
            const result = try child.spawnAndWait();
            
            switch (result) {
                .Exited => |code| {
                    if (code != 0) {
                        try stderr.print("\nError: Clone failed with exit code {d}\n", .{code});
                        return error.CloneFailed;
                    }
                },
                else => {
                    try stderr.print("\nError: Clone process terminated abnormally\n", .{});
                    return error.CloneFailed;
                },
            }
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
            const result = try child.spawnAndWait();
            
            switch (result) {
                .Exited => |code| {
                    if (code != 0) {
                        try stderr.print("\nError: Failed to open browser (exit code {d})\n", .{code});
                        return error.OpenBrowserFailed;
                    }
                },
                else => {
                    try stderr.print("\nError: Browser process terminated abnormally\n", .{});
                    return error.OpenBrowserFailed;
                },
            }
        },
        .show_name => {
            try printToStdout("{s}\n", .{repo.nameWithOwner});
        },
    }
}

fn copyToClipboard(allocator: std.mem.Allocator, text: []const u8) !void {
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
            try printToStdout("\nCopied to clipboard!\n", .{});
            return;
        } else |_| {
            continue;
        }
    }
    
    try printToStdout("\nError: No clipboard tool found (pbcopy, wl-copy, xclip, clip.exe).\n", .{});
}
