//! Core data types for gh-select

const std = @import("std");

pub const Repository = struct {
    nameWithOwner: []const u8,
    description: ?[]const u8 = null,
    homepageUrl: ?[]const u8 = null,
    isPrivate: bool,
    
    // Helper to get just the owner or just the name might be useful later
    pub fn owner(self: Repository) []const u8 {
        var it = std.mem.splitScalar(u8, self.nameWithOwner, '/');
        return it.first();
    }
    
    pub fn name(self: Repository) []const u8 {
        var it = std.mem.splitScalar(u8, self.nameWithOwner, '/');
        _ = it.next();
        return it.rest();
    }
};

pub const Action = enum {
    clone,
    copy_name,
    copy_url,
    open_browser,
    show_name,
};
