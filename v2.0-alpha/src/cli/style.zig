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

pub const cursor_block = "\u{2588}"; // Block character

// Bright colors
pub const bright_green = "\x1b[92m";
pub const bright_cyan = "\x1b[96m";
pub const bright_white = "\x1b[97m";

// Tokyo Night Theme (TrueColor)
pub const tn_bg = "\x1b[48;2;26;27;38m";
pub const tn_fg = "\x1b[38;2;192;202;245m";     // #c0caf5
pub const tn_blue = "\x1b[38;2;122;162;247m";   // #7aa2f7
pub const tn_cyan = "\x1b[38;2;125;207;255m";   // #7dcfff
pub const tn_magenta = "\x1b[38;2;187;154;247m";// #bb9af7
pub const tn_green = "\x1b[38;2;158;206;106m";  // #9ece6a
pub const tn_red = "\x1b[38;2;247;118;142m";    // #f7768e
pub const tn_yellow = "\x1b[38;2;224;175;104m"; // #e0af68
pub const tn_comment = "\x1b[38;2;86;95;137m";  // #565f89 (dim)

/// Check if colors should be enabled at runtime
pub fn colorsEnabled() bool {
    // Respect NO_COLOR environment variable
    if (std.posix.getenv("NO_COLOR")) |_| {
        return false;
    }
    
    // Check if stdout is a TTY
    const stdout = std.fs.File{ .handle = std.posix.STDOUT_FILENO };
    return std.posix.isatty(stdout.handle);
}

/// Styled writer that respects NO_COLOR
pub const StyledWriter = struct {
    file: std.fs.File,
    colors: bool,

    pub fn init(file: std.fs.File) StyledWriter {
        return .{
            .file = file,
            .colors = colorsEnabled(),
        };
    }

    pub fn print(self: *StyledWriter, comptime fmt: []const u8, args: anytype) !void {
        var buf: [4096]u8 = undefined;
        var w = self.file.writer(&buf);
        try w.interface.print(fmt, args);
        try w.interface.flush();
    }

    pub fn styled(self: *StyledWriter, style_code: []const u8, text: []const u8) !void {
        if (self.colors) {
            var buf: [4096]u8 = undefined;
            var w = self.file.writer(&buf);
            try w.interface.print("{s}{s}{s}", .{ style_code, text, reset });
            try w.interface.flush();
        } else {
            try self.file.writeAll(text);
        }
    }
};

test "color constants exist" {
    try std.testing.expect(bold.len > 0);
    try std.testing.expect(reset.len > 0);
}
