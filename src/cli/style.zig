//! ANSI styling utilities for terminal output
//!
//! Respects NO_COLOR environment variable and TTY detection.

const std = @import("std");

// ANSI codes - always defined, runtime check at output time
pub const bold = "\x1b[1m";
pub const dim = "\x1b[2m";
pub const italic = "\x1b[3m";
pub const reset = "\x1b[0m";

// Colors
pub const red = "\x1b[31m";
pub const green = "\x1b[32m";
pub const yellow = "\x1b[33m";
pub const blue = "\x1b[34m";
pub const magenta = "\x1b[35m";
pub const cyan = "\x1b[36m";
pub const white = "\x1b[37m";

// Bright colors
pub const bright_green = "\x1b[92m";
pub const bright_cyan = "\x1b[96m";
pub const bright_white = "\x1b[97m";

/// Check if colors should be enabled at runtime
pub fn colorsEnabled() bool {
    // Respect NO_COLOR environment variable
    if (std.posix.getenv("NO_COLOR")) |_| {
        return false;
    }
    
    // Check if stdout is a TTY
    const stdout = std.io.getStdOut();
    return std.posix.isatty(stdout.handle);
}

/// Styled writer that respects NO_COLOR
pub const StyledWriter = struct {
    inner: std.fs.File.Writer,
    colors: bool,

    pub fn init(file: std.fs.File) StyledWriter {
        return .{
            .inner = file.writer(),
            .colors = colorsEnabled(),
        };
    }

    pub fn print(self: *StyledWriter, comptime fmt: []const u8, args: anytype) !void {
        try self.inner.print(fmt, args);
    }

    pub fn styled(self: *StyledWriter, style_code: []const u8, text: []const u8) !void {
        if (self.colors) {
            try self.inner.print("{s}{s}{s}", .{ style_code, text, reset });
        } else {
            try self.inner.writeAll(text);
        }
    }
};

test "color constants exist" {
    try std.testing.expect(bold.len > 0);
    try std.testing.expect(reset.len > 0);
}
