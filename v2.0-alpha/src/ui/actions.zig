//! Action handling for selected repositories

const std = @import("std");
const types = @import("../core/types.zig");

/// Version-safe print helper for Zig 0.15.2
fn print(file: std.fs.File, comptime fmt: []const u8, args: anytype) !void {
    const s = try std.fmt.allocPrint(std.heap.page_allocator, fmt, args);
    defer std.heap.page_allocator.free(s);
    try file.writeAll(s);
}

pub fn executeAction(allocator: std.mem.Allocator, action: types.Action, repo: types.Repository) !void {
    const stderr_file = std.fs.File{ .handle = std.posix.STDERR_FILENO };
    const stdout_file = std.fs.File{ .handle = std.posix.STDOUT_FILENO };

    switch (action) {
        .clone => {
            const argv = [_][]const u8{ "gh", "repo", "clone", repo.nameWithOwner };
            var child = std.process.Child.init(&argv, allocator);
            const result = try child.spawnAndWait();
            
            switch (result) {
                .Exited => |code| {
                    if (code != 0) {
                        try print(stderr_file, "\nError: Clone failed with exit code {d}\n", .{code});
                        return error.CloneFailed;
                    }
                },
                else => {
                    try print(stderr_file, "\nError: Clone process terminated abnormally\n", .{});
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
                        try print(stderr_file, "\nError: Failed to open browser (exit code {d})\n", .{code});
                        return error.OpenBrowserFailed;
                    }
                },
                else => {
                    try print(stderr_file, "\nError: Browser process terminated abnormally\n", .{});
                    return error.OpenBrowserFailed;
                },
            }
        },
        .show_name => {
            try print(stdout_file, "{s}\n", .{repo.nameWithOwner});
        },
    }
}

fn copyToClipboard(allocator: std.mem.Allocator, text: []const u8) !void {
    const stdout_file = std.fs.File{ .handle = std.posix.STDOUT_FILENO };
    
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
            try print(stdout_file, "\nCopied to clipboard!\n", .{});
            return;
        } else |_| {
            continue;
        }
    }
    
    try print(stdout_file, "\nError: No clipboard tool found (pbcopy, wl-copy, xclip, clip.exe).\n", .{});
}
