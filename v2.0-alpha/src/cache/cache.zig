//! Cache management for gh-select
//!
//! Handles reading/writing the repository list to a JSON file with TTL expiration.

const std = @import("std");
const types = @import("../core/types.zig");
const paths = @import("../config/paths.zig");
const errors = @import("../core/errors.zig");

pub const Cache = struct {
    allocator: std.mem.Allocator,
    ttl_seconds: i64,

    pub fn init(allocator: std.mem.Allocator) Cache {
        // Default TTL: 30 minutes (1800 seconds)
        // Can be overridden by GH_SELECT_CACHE_TTL env var (in hours)
        var ttl: i64 = 1800;
        
        if (std.posix.getenv("GH_SELECT_CACHE_TTL")) |ttl_str| {
            if (std.fmt.parseInt(i64, ttl_str, 10)) |hours| {
                ttl = hours * 3600; // Convert hours to seconds
            } else |_| {
                // Invalid value, use default
            }
        }
        
        return .{
            .allocator = allocator,
            .ttl_seconds = ttl,
        };
    }

    /// Save repositories to cache
    pub fn save(self: Cache, repos: []const types.Repository) !void {
        try paths.ensureCacheDir(self.allocator);
        
        const cache_file_path = try paths.getCacheFile(self.allocator);
        defer self.allocator.free(cache_file_path);

        const file = std.fs.createFileAbsolute(cache_file_path, .{ .truncate = true }) catch {
            return errors.GhSelectError.CacheWriteFailed;
        };
        defer file.close();

        const writer = file.writer();
        try std.json.stringify(repos, .{ .whitespace = .indent_2 }, writer);
    }

    /// Load repositories from cache if valid
    /// Returns error.CacheExpired if expired or missing
    pub fn load(self: Cache) !std.json.Parsed([]types.Repository) {
        const cache_file_path = try paths.getCacheFile(self.allocator);
        defer self.allocator.free(cache_file_path);

        const file = std.fs.openFileAbsolute(cache_file_path, .{}) catch |err| {
            // FileNotFound means cache doesn't exist yet - treat as expired
            if (err == error.FileNotFound) {
                return errors.GhSelectError.CacheExpired;
            }
            // Other errors are actual read failures
            return errors.GhSelectError.CacheReadFailed;
        };
        defer file.close();

        // Check modification time
        const stat = try file.stat();
        const mtime = @divTrunc(stat.mtime, 1000000000); // ns to seconds
        const now = std.time.timestamp();

        if (now - mtime > self.ttl_seconds) {
            return errors.GhSelectError.CacheExpired;
        }

        const json_body = try file.readToEndAlloc(self.allocator, 10 * 1024 * 1024); // 10MB limit
        defer self.allocator.free(json_body);

        const parsed = std.json.parseFromSlice([]types.Repository, self.allocator, json_body, .{
            .ignore_unknown_fields = true,
            .allocate = .alloc_always,
        }) catch return errors.GhSelectError.GhJsonParseError;

        return parsed;
    }
};

// Unit tests
test "Cache init default TTL" {
    const cache = Cache.init(std.testing.allocator);
    try std.testing.expectEqual(@as(i64, 1800), cache.ttl_seconds);
}

test "Cache TTL is 30 minutes" {
    const cache = Cache.init(std.testing.allocator);
    // 30 minutes = 30 * 60 = 1800 seconds
    try std.testing.expectEqual(@as(i64, 30 * 60), cache.ttl_seconds);
}
