//! gh-select — Interactive GitHub Repository Selector
//!
//! A GitHub CLI extension for fuzzy-finding and managing repositories.
//! Zig rewrite of v1.x bash implementation for performance and portability.

const std = @import("std");
const cli = @import("cli/args.zig");
const style = @import("cli/style.zig");
const config = @import("config/paths.zig");

pub const version = "2.0.0-alpha";
pub const author = "Remco Stoeten";
pub const repo = "https://github.com/remcostoeten/gh-select";

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const stdout = std.io.getStdOut().writer();
    const stderr = std.io.getStdErr().writer();

    // Parse arguments
    const args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    if (args.len > 1) {
        const arg = args[1];
        
        if (std.mem.eql(u8, arg, "--version") or std.mem.eql(u8, arg, "-v") or std.mem.eql(u8, arg, "version")) {
            try showVersion(stdout);
            return;
        }
        
        if (std.mem.eql(u8, arg, "--help") or std.mem.eql(u8, arg, "-h") or std.mem.eql(u8, arg, "help")) {
            try showHelp(stdout);
            return;
        }
        
        if (std.mem.eql(u8, arg, "--no-cache") or std.mem.eql(u8, arg, "-n")) {
            // Set force refresh flag (handled in main logic)
            try stderr.print("{s}Force refresh enabled{s}\n", .{ style.dim, style.reset });
        }
        
        if (std.mem.eql(u8, arg, "--refresh-only") or std.mem.eql(u8, arg, "-r")) {
            try refreshCache(allocator, stdout, stderr);
            return;
        }
    }

    // Main interactive flow
    try runInteractive(allocator, stdout, stderr);
}

fn showVersion(writer: anytype) !void {
    try writer.print("\n", .{});
    try writer.print("{s}{s}gh-select{s} {s}v{s}{s}\n", .{ 
        style.bold, style.cyan, style.reset, style.dim, version, style.reset 
    });
    try writer.print("{s}A GitHub CLI extension following the gh extension spec{s}\n", .{ 
        style.dim, style.reset 
    });
    try writer.print("\n", .{});
    try writer.print("Fuzzy-find and manage your GitHub repos from the terminal.\n", .{});
    try writer.print("\n", .{});
    try writer.print("{s}─────────────────────────────────────────{s}\n", .{ style.dim, style.reset });
    try writer.print("By {s}{s}{s}\n", .{ style.bold, author, style.reset });
    try writer.print("{s}   @remcostoeten{s}\n", .{ style.dim, style.reset });
    try writer.print("{s}   https://remcostoeten.nl{s}\n", .{ style.blue, style.reset });
    try writer.print("\n", .{});
    try writer.print("{s}Source:{s} {s}{s}{s}\n", .{ style.dim, style.reset, style.blue, repo, style.reset });
    try writer.print("{s}─────────────────────────────────────────{s}\n", .{ style.dim, style.reset });
    try writer.print("\n", .{});
}

fn showHelp(writer: anytype) !void {
    try writer.print("\n", .{});
    try writer.print("{s}{s}gh-select{s} {s}─ Interactive GitHub Repository Selector{s}\n", .{
        style.bold, style.cyan, style.reset, style.dim, style.reset
    });
    try writer.print("{s}A GitHub CLI extension following the gh extension spec{s}\n", .{
        style.dim, style.reset
    });
    try writer.print("\n", .{});
    try writer.print("{s}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━{s}\n", .{
        style.dim, style.reset
    });
    try writer.print("\n", .{});
    try writer.print("{s}USAGE{s}\n", .{ style.bold, style.reset });
    try writer.print("\n", .{});
    try writer.print("    {s}gh select{s} {s}[OPTIONS]{s}\n", .{
        style.green, style.reset, style.dim, style.reset
    });
    try writer.print("\n", .{});
    try writer.print("{s}OPTIONS{s}\n", .{ style.bold, style.reset });
    try writer.print("\n", .{});
    try writer.print("    {s}-n{s}, {s}--no-cache{s}       Bypass cache, fetch fresh data\n", .{
        style.yellow, style.reset, style.yellow, style.reset
    });
    try writer.print("    {s}-r{s}, {s}--refresh-only{s}   Refresh cache and exit\n", .{
        style.yellow, style.reset, style.yellow, style.reset
    });
    try writer.print("    {s}-v{s}, {s}--version{s}        Show version information\n", .{
        style.yellow, style.reset, style.yellow, style.reset
    });
    try writer.print("    {s}-h{s}, {s}--help{s}           Show this help message\n", .{
        style.yellow, style.reset, style.yellow, style.reset
    });
    try writer.print("\n", .{});
    try writer.print("{s}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━{s}\n", .{
        style.dim, style.reset
    });
    try writer.print("\n", .{});
}

fn refreshCache(allocator: std.mem.Allocator, stdout: anytype, stderr: anytype) !void {
    _ = allocator;
    _ = stderr;
    try stdout.print("{s}Loading repositories...{s}\n", .{ style.cyan, style.reset });
    
    // TODO: Implement GitHub API call
    // For now, call gh CLI as subprocess
    try stdout.print("{s}[TODO] GitHub API integration{s}\n", .{ style.yellow, style.reset });
}

fn runInteractive(allocator: std.mem.Allocator, stdout: anytype, stderr: anytype) !void {
    _ = allocator;
    _ = stderr;
    try stdout.print("{s}[TODO] Interactive UI with fuzzy search{s}\n", .{ style.yellow, style.reset });
}

test "version string" {
    try std.testing.expectEqualStrings("2.0.0-alpha", version);
}

test "author string" {
    try std.testing.expectEqualStrings("Remco Stoeten", author);
}
