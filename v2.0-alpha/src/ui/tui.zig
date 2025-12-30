//! Minimal TUI implementation for gh-select
//! 
//! Handles raw mode, input reading, and basic ANSI terminal control.

const std = @import("std");
const posix = std.posix;

pub const Key = union(enum) {
    char: u8,
    ctrl: u8,
    enter,
    backspace,
    escape,
    up,
    down,
    left,
    right,
    unknown,
};

pub const Tui = struct {
    original_termios: posix.termios,
    in_raw_mode: bool,
    writer: std.fs.File.Writer,

    pub fn init(writer: std.fs.File.Writer) !Tui {
        const stdin = std.io.getStdIn();
        const original = try posix.tcgetattr(stdin.handle);
        return .{
            .original_termios = original,
            .in_raw_mode = false,
            .writer = writer,
        };
    }

    pub fn enableRawMode(self: *Tui) !void {
        if (self.in_raw_mode) return;
        
        const stdin = std.io.getStdIn();
        var raw = self.original_termios;
        
        // input modes: no break, no CR to NL, no parity check, no strip char,
        // no start/stop output control.
        raw.iflag.BRKINT = false;
        raw.iflag.ICRNL = false;
        raw.iflag.INPCK = false;
        raw.iflag.ISTRIP = false;
        raw.iflag.IXON = false;
        
        // output modes - disable post processing
        raw.oflag.OPOST = false;
        
        // control modes - set 8 bit chars
        raw.cflag.CSIZE = .CS8;
        
        // local modes - chopping off canonical mode
        raw.lflag.ECHO = false;
        raw.lflag.ICANON = false;
        raw.lflag.IEXTEN = false;
        raw.lflag.ISIG = false;
        
        // control chars - min chars to read and timeout
        raw.cc[@intFromEnum(posix.V.MIN)] = 0;
        raw.cc[@intFromEnum(posix.V.TIME)] = 1;
        
        try posix.tcsetattr(stdin.handle, .FLUSH, raw);
        self.in_raw_mode = true;
        
        // Switch to alt screen and hide cursor
        try self.writer.print("\x1b[?1049h\x1b[?25l", .{});
    }

    pub fn disableRawMode(self: *Tui) !void {
        if (!self.in_raw_mode) return;
        
        // Restore cursor and exit alt screen
        try self.writer.print("\x1b[?25h\x1b[?1049l", .{});
        
        const stdin = std.io.getStdIn();
        try posix.tcsetattr(stdin.handle, .FLUSH, self.original_termios);
        self.in_raw_mode = false;
    }

    /// Read a key from stdin
    pub fn readKey(self: *Tui) !Key {
        _ = self;
        var buf: [1]u8 = undefined;
        const stdin = std.io.getStdIn().reader();
        
        // Non-blocking read loop (since VMIN=0)
        while (true) {
            const n = try stdin.read(&buf);
            if (n == 0) continue;
            
            const c = buf[0];
            
            if (c == 27) { // Escape sequence
                var seq: [2]u8 = undefined;
                if (try stdin.read(&seq) < 2) return .escape;
                
                if (seq[0] == '[') {
                    switch (seq[1]) {
                        'A' => return .up,
                        'B' => return .down,
                        'C' => return .right,
                        'D' => return .left,
                        else => return .unknown,
                    }
                }
                return .escape;
            } else if (c == 13) {
                return .enter;
            } else if (c == 127) {
                return .backspace;
            } else if (c == 3) {
                return .{ .ctrl = 3 }; // Ctrl-C
            } else if (std.ascii.isControl(c)) {
                 return .{ .ctrl = c };
            } else {
                return .{ .char = c };
            }
        }
    }

    // Drawing primitives
    pub fn clear(self: *Tui) !void {
        try self.writer.print("\x1b[2J\x1b[H", .{});
    }

    /// Get terminal size using TIOCGWINSZ ioctl
    /// Falls back to 24x80 if ioctl fails
    pub fn getTermSize(self: *Tui) !struct { rows: usize, cols: usize } {
        _ = self;
        
        // winsize struct layout for TIOCGWINSZ
        const Winsize = extern struct {
            ws_row: u16,
            ws_col: u16,
            ws_xpixel: u16,
            ws_ypixel: u16,
        };
        
        var ws: Winsize = undefined;
        const TIOCGWINSZ: u32 = 0x5413; // Linux value
        
        const stdout = std.io.getStdOut();
        const result = std.posix.system.ioctl(stdout.handle, TIOCGWINSZ, @intFromPtr(&ws));
        
        if (result == 0 and ws.ws_row > 0 and ws.ws_col > 0) {
            return .{ .rows = ws.ws_row, .cols = ws.ws_col };
        }
        
        // Fallback to default if ioctl fails
        return .{ .rows = 24, .cols = 80 };
    }

    pub fn setCursor(self: *Tui, row: usize, col: usize) !void {
       try self.writer.print("\x1b[{d};{d}H", .{ row + 1, col + 1 });
    }

    pub fn writeSpaces(self: *Tui, count: usize) !void {
        var i: usize = 0;
        while (i < count) : (i += 1) {
            try self.writer.print(" ", .{});
        }
    }
};
