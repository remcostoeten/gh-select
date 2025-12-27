//! gh-select — Interactive GitHub Repository Selector
//!
//! A GitHub CLI extension for fuzzy-finding and managing repositories.
//! Zig rewrite of v1.x bash implementation for performance and portability.

const std = @import("std");
const cli = @import("cli/args.zig");
const style = @import("cli/style.zig");
const config = @import("config/paths.zig");
const api = @import("github/api.zig");
const cache = @import("cache/cache.zig");

pub const version = "2.0.0-alpha";
pub const author = "Remco Stoeten";
pub const repo = "https://github.com/remcostoeten/gh-select";

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    var styled_stdout = style.StyledWriter.init(std.io.getStdOut());
    var styled_stderr = style.StyledWriter.init(std.io.getStdErr());

    // Parse arguments
    const args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    if (args.len > 1) {
        const arg = args[1];
        const cmd = cli.parseArg(arg);

        switch (cmd) {
            .version => {
                try showVersion(&styled_stdout);
                return;
            },
            .help => {
                try showHelp(&styled_stdout);
                return;
            },
            .no_cache => {
                try styled_stderr.print("{s}Force refresh enabled{s}\n", .{ style.dim, style.reset });
                // In a real run, we would set a flag here and continue to runInteractive
                // For now, let's just proceed to interactive
            },
            .refresh_only => {
                try refreshCache(allocator, &styled_stdout, &styled_stderr);
                return;
            },
            .unknown => {
                try styled_stderr.print("Unknown option: {s}\n", .{arg});
                if (cli.suggestCommand(arg)) |suggestion| {
                    try styled_stderr.print("Did you mean: {s}--{s}{s}? [Y/n] ", .{ style.bold, suggestion, style.reset });
                    // Simple input check could go here for interactive confirmation
                }
                return;
            },
            else => {},
        }
    }

    // Main interactive flow
    try runInteractive(allocator, &styled_stdout, &styled_stderr);
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
    // 1. Init API
    const gh_api = api.Api.init(allocator);
    
    // 2. Check Auth
    gh_api.checkAuth() catch |err| {
        try stderr.print("{s}Error:{s} GitHub CLI not authenticated or installed.\n", .{ style.red, style.reset });
        return err;
    };
    
    try stdout.print("{s}Fetching repositories...{s}", .{ style.cyan, style.reset });
    
    // 3. Fetch Repos
    const parsed_repos = gh_api.fetchRepos() catch |err| {
        try stderr.print("\n{s}Error:{s} Failed to fetch repositories.\n", .{ style.red, style.reset });
        return err;
    };
    defer parsed_repos.deinit();
    
    try stdout.print("\n{s}Fetched {d} repositories{s}\n", .{ style.green, parsed_repos.value.len, style.reset });
    
    // 4. Save to Cache
    const repo_cache = cache.Cache.init(allocator);
    repo_cache.save(parsed_repos.value) catch |err| {
        try stderr.print("{s}Error:{s} Failed to save cache.\n", .{ style.red, style.reset });
        return err;
    };
    
    try stdout.print("Cache updated.\n", .{});
}

fn runInteractive(allocator: std.mem.Allocator, stdout: anytype, stderr: anytype) !void {
    const gh_api = api.Api.init(allocator);
    const repo_cache = cache.Cache.init(allocator);

    // 1. Try to load from cache
    var repos_parsed = repo_cache.load() catch |err| switch (err) {
        error.CacheExpired, error.CacheReadFailed => blk: {
            // 2. Fetch fresh if cache miss
            try stdout.print("{s}Fetching repositories...{s}", .{ style.cyan, style.reset });
            const fresh = try gh_api.fetchRepos(); // propagates error if fails
            try repo_cache.save(fresh.value);
            break :blk fresh;
        },
        else => return err,
    };
    defer repos_parsed.deinit();

    // 3. Run UI Selector
    const selector_mod = @import("ui/selector.zig");
    var selector = selector_mod.Selector.init(allocator, repos_parsed.value);
    defer selector.deinit();

    if (try selector.run()) |selected| {
        // 4. Handle Selection (default: open in browser for now, or print)
        // For v1 parity we need to support actions, but interactive mode usually defaults to something or prompts for action?
        // v1 behavior: ENTER -> print URL or path? 
        // Actually v1 usually prints the selected repo to stdout so the shell can cd? 
        // checking v1 scripts: `gh_select` script usually performs action or prints.
        // The bash script does: `selected=$(...); if [ -n "$selected" ]; then ... handle_selection ... fi`
        
        // For this port, let's just print the name to stdout for now, as that allows `cd $(gh-select)` style usage.
        try stderr.print("\nSelected: {s}\n", .{selected.nameWithOwner});
        
        // Use actions module
        const actions = @import("ui/actions.zig");
        // For now hardcode 'show_name' or maybe we can add a secondary menu later.
        // Let's executed 'show_name' action
        try actions.executeAction(allocator, .show_name, selected);
    } else {
        try stderr.print("\nCancelled.\n", .{});
    }
}
