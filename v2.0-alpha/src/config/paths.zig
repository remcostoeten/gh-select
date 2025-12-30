//! XDG-compliant path utilities for config and cache
//!
//! Follows XDG Base Directory Specification.

const std = @import("std");

/// Get the cache directory path for gh-select
pub fn getCacheDir(allocator: std.mem.Allocator) ![]u8 {
    // Check XDG_CACHE_HOME first
    if (std.posix.getenv("XDG_CACHE_HOME")) |xdg_cache| {
        return try std.fmt.allocPrint(allocator, "{s}/gh-select", .{xdg_cache});
    }
    
    // Fall back to ~/.cache
    if (std.posix.getenv("HOME")) |home| {
        return try std.fmt.allocPrint(allocator, "{s}/.cache/gh-select", .{home});
    }
    
    return error.NoHomeDir;
}

/// Get the config directory path for gh-select
pub fn getConfigDir(allocator: std.mem.Allocator) ![]u8 {
    // Check XDG_CONFIG_HOME first
    if (std.posix.getenv("XDG_CONFIG_HOME")) |xdg_config| {
        return try std.fmt.allocPrint(allocator, "{s}/gh-select", .{xdg_config});
    }
    
    // Fall back to ~/.config
    if (std.posix.getenv("HOME")) |home| {
        return try std.fmt.allocPrint(allocator, "{s}/.config/gh-select", .{home});
    }
    
    return error.NoHomeDir;
}

/// Get the cache file path for repository data
pub fn getCacheFile(allocator: std.mem.Allocator) ![]u8 {
    const cache_dir = try getCacheDir(allocator);
    defer allocator.free(cache_dir);
    return try std.fmt.allocPrint(allocator, "{s}/repos.json", .{cache_dir});
}

/// Ensure the cache directory exists, creating parent dirs if needed
pub fn ensureCacheDir(allocator: std.mem.Allocator) !void {
    const cache_dir = try getCacheDir(allocator);
    defer allocator.free(cache_dir);
    
    // Use makePath to create parent directories recursively
    std.fs.makePath(std.fs.cwd(), cache_dir) catch |err| {
        if (err != error.PathAlreadyExists) {
            return err;
        }
    };
}

test "getCacheDir" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();
    
    const dir = try getCacheDir(allocator);
    defer allocator.free(dir);
    
    try std.testing.expect(std.mem.endsWith(u8, dir, "/gh-select"));
}

test "getConfigDir" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();
    
    const dir = try getConfigDir(allocator);
    defer allocator.free(dir);
    
    try std.testing.expect(std.mem.endsWith(u8, dir, "/gh-select"));
}
