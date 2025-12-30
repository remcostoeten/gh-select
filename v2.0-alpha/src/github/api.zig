//! GitHub API integration using the gh CLI tool

const std = @import("std");
const types = @import("../core/types.zig");
const errors = @import("../core/errors.zig");

pub const Api = struct {
    allocator: std.mem.Allocator,

    pub fn init(allocator: std.mem.Allocator) Api {
        return .{
            .allocator = allocator,
        };
    }

    /// Check if gh CLI is installed and authenticated
    pub fn checkAuth(self: Api) !void {
        // Check if gh is installed
        const argv_version = [_][]const u8{ "gh", "--version" };
        var child_version = std.process.Child.init(&argv_version, self.allocator);
        child_version.stdout_behavior = .Ignore;
        child_version.stderr_behavior = .Ignore;
        
        switch (try child_version.spawnAndWait()) {
            .Exited => |code| if (code != 0) return errors.GhSelectError.GhCliNotInstalled,
            else => return errors.GhSelectError.GhCliNotInstalled,
        }

        // Check auth status
        const argv_auth = [_][]const u8{ "gh", "auth", "status" };
        var child_auth = std.process.Child.init(&argv_auth, self.allocator);
        child_auth.stdout_behavior = .Ignore;
        child_auth.stderr_behavior = .Ignore;
        
        switch (try child_auth.spawnAndWait()) {
            .Exited => |code| if (code != 0) return errors.GhSelectError.GhNotAuthenticated,
            else => return errors.GhSelectError.GhApiFailed, // Signal or Stopped
        }
    }

    /// Fetch repositories using gh repo list
    /// Returns a parsed JSON object that must be deinitialized by the caller
    pub fn fetchRepos(self: Api) !std.json.Parsed([]types.Repository) {
        // gh repo list --json nameWithOwner,description,isPrivate --limit 1000
        const argv = [_][]const u8{
            "gh", "repo", "list",
            "--json", "nameWithOwner,description,homepageUrl,isPrivate",
            "--limit", "1000",
        };

        var child = std.process.Child.init(&argv, self.allocator);
        child.stdout_behavior = .Pipe;
        child.stderr_behavior = .Ignore; // Ignore stderr to prevent deadlock if buffer fills

        try child.spawn();

        const stdout = child.stdout.?.reader();
        const json_body = try stdout.readAllAlloc(self.allocator, 10 * 1024 * 1024); // 10MB limit
        defer self.allocator.free(json_body);

        switch (try child.wait()) {
            .Exited => |code| {
                if (code != 0) return errors.GhSelectError.GhApiFailed;
            },
            else => return errors.GhSelectError.GhApiFailed,
        }

        // Parse JSON
        // We use parseFromSlice with duplicate_fields to ensure the result owns its data
        // independent of the json_body buffer we are about to free.
        const parsed = std.json.parseFromSlice([]types.Repository, self.allocator, json_body, .{
            .ignore_unknown_fields = true,
            .allocate = .alloc_always, 
        }) catch return errors.GhSelectError.GhJsonParseError;
        
        return parsed;
    }
};
