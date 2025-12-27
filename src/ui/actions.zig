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
             // TODO: clipboard
        },
        .copy_url => {
            // TODO: clipboard
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
