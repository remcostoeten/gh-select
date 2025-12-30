//! Argument parsing for gh-select CLI
//!
//! Supports --long, -short, and bare forms for commands.

const std = @import("std");

pub const Command = enum {
    help,
    version,
    no_cache,
    refresh_only,
    interactive,
    unknown,
};

pub fn parseArg(arg: []const u8) Command {
    // Version
    if (std.mem.eql(u8, arg, "--version") or 
        std.mem.eql(u8, arg, "-v") or 
        std.mem.eql(u8, arg, "version")) {
        return .version;
    }
    
    // Help
    if (std.mem.eql(u8, arg, "--help") or 
        std.mem.eql(u8, arg, "-h") or 
        std.mem.eql(u8, arg, "help")) {
        return .help;
    }
    
    // No cache
    if (std.mem.eql(u8, arg, "--no-cache") or 
        std.mem.eql(u8, arg, "-n")) {
        return .no_cache;
    }
    
    // Refresh only
    if (std.mem.eql(u8, arg, "--refresh-only") or 
        std.mem.eql(u8, arg, "-r")) {
        return .refresh_only;
    }
    
    return .unknown;
}

/// Calculate similarity score for typo suggestions
pub fn similarity(a: []const u8, b: []const u8) f32 {
    if (a.len == 0 or b.len == 0) return 0.0;
    
    var matches: u32 = 0;
    for (a) |char| {
        if (std.mem.indexOfScalar(u8, b, char) != null) {
            matches += 1;
        }
    }
    
    return @as(f32, @floatFromInt(matches)) / @as(f32, @floatFromInt(a.len));
}

/// Suggest closest matching command for typos
pub fn suggestCommand(input: []const u8) ?[]const u8 {
    const commands = [_][]const u8{ "version", "help", "no-cache", "refresh-only" };
    
    // Strip leading dashes for comparison
    var clean_input = input;
    while (clean_input.len > 0 and clean_input[0] == '-') {
        clean_input = clean_input[1..];
    }
    
    var best_match: ?[]const u8 = null;
    var best_score: f32 = 0.6; // Minimum threshold
    
    for (commands) |cmd| {
        const score = similarity(clean_input, cmd);
        if (score > best_score) {
            best_score = score;
            best_match = cmd;
        }
    }
    
    return best_match;
}

test "parseArg version" {
    try std.testing.expectEqual(Command.version, parseArg("--version"));
    try std.testing.expectEqual(Command.version, parseArg("-v"));
    try std.testing.expectEqual(Command.version, parseArg("version"));
}

test "parseArg help" {
    try std.testing.expectEqual(Command.help, parseArg("--help"));
    try std.testing.expectEqual(Command.help, parseArg("-h"));
    try std.testing.expectEqual(Command.help, parseArg("help"));
}

test "parseArg flags" {
    try std.testing.expectEqual(Command.no_cache, parseArg("--no-cache"));
    try std.testing.expectEqual(Command.no_cache, parseArg("-n"));
    try std.testing.expectEqual(Command.refresh_only, parseArg("--refresh-only"));
    try std.testing.expectEqual(Command.refresh_only, parseArg("-r"));
}

test "suggestCommand typos" {
    try std.testing.expectEqualStrings("version", suggestCommand("--versoin").?);
    try std.testing.expectEqualStrings("help", suggestCommand("--hlep").?);
}
