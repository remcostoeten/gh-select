const std = @import("std");

pub fn main() void {
    // Manual stdout
    const stdout_file = std.fs.File{ .handle = std.posix.STDOUT_FILENO };
    const writer = stdout_file.writer();
    _ = writer;

    // Manual stdin
    const stdin_file = std.fs.File{ .handle = std.posix.STDIN_FILENO };
    const reader = stdin_file.reader();
    _ = reader;
    
    // Check if bufferedWriter exists in std.io
    // const bw = std.io.bufferedWriter;
}
