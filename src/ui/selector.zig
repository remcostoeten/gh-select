//! UI Selector module using custom TUI engine and zf for filtering

const std = @import("std");
const types = @import("../core/types.zig");
const style = @import("../cli/style.zig");
const tui = @import("tui.zig");

pub const Selector = struct {
    allocator: std.mem.Allocator,
    repos: []types.Repository,
    filtered_matches: std.ArrayList(Match),
    query: std.ArrayList(u8),
    selected_index: usize,

    const Match = struct {
        index: usize,
        score: i64,
        
        fn lessThan(_: void, a: Match, b: Match) bool {
            return a.score > b.score;
        }
    };

    pub fn init(allocator: std.mem.Allocator, repos: []types.Repository) Selector {
        return .{
            .allocator = allocator,
            .repos = repos,
            .filtered_matches = std.ArrayList(Match).init(allocator),
            .query = std.ArrayList(u8).init(allocator),
            .selected_index = 0,
        };
    }

    pub fn deinit(self: *Selector) void {
        self.filtered_matches.deinit();
        self.query.deinit();
    }

    /// Calculate a fuzzy match score. Returns null if no match.
    /// Higher score is better.
    fn fuzzyScore(haystack: []const u8, needle: []const u8) ?i64 {
        if (needle.len == 0) return 0;
        
        // Helper to check containment without allocations for speed
        // But for fuzzy we want subsequence.
        
        var score: i64 = 0;
        var h_idx: usize = 0;
        var n_idx: usize = 0;
        var consecutive_matches: i64 = 0;
        var first_char_bonus: bool = true; // Bonus for matching start of string

        while (h_idx < haystack.len and n_idx < needle.len) {
            const h_char = std.ascii.toLower(haystack[h_idx]);
            const n_char = std.ascii.toLower(needle[n_idx]);

            if (h_char == n_char) {
                score += 10;
                
                // Bonus for consecutive matches
                if (consecutive_matches > 0) {
                    score += 5 + consecutive_matches; // Increasing bonus
                }
                consecutive_matches += 1;

                // Bonus for start of word boundaries
                if (h_idx == 0 or haystack[h_idx-1] == '/' or haystack[h_idx-1] == '-' or haystack[h_idx-1] == '_') {
                    score += 20;
                }
                
                // Huge bonus for exact substring match of the query at the beginning
                if (first_char_bonus and h_idx == 0) {
                    score += 50;
                }

                n_idx += 1;
            } else {
                consecutive_matches = 0;
                first_char_bonus = false;
                // Tiny penalty for gap, but we primarily care about presence and order
                score -= 1; 
            }
            h_idx += 1;
        }

        if (n_idx < needle.len) return null; // Not all characters matched

        // Penalty for length difference (shorter matches are better)
        const len_diff = @as(i64, @intCast(haystack.len)) - @as(i64, @intCast(n_idx));
        score -= len_diff * 2;

        return score;
    }

    /// Update filter based on current query using fuzzy matching
    pub fn updateFilter(self: *Selector) !void {
        self.filtered_matches.clearRetainingCapacity();
        
        const q = self.query.items;

        if (q.len == 0) {
            // No query: show all, preserve original order (or maybe reverse chrono if we had it?)
            // For now, original order.
            try self.filtered_matches.ensureTotalCapacity(self.repos.len);
            for (0..self.repos.len) |i| {
                self.filtered_matches.appendAssumeCapacity(.{ .index = i, .score = 0 });
            }
        } else {
            // Fuzzy match
            for (self.repos, 0..) |repo, i| {
                if (fuzzyScore(repo.nameWithOwner, q)) |score| {
                    try self.filtered_matches.append(.{ .index = i, .score = score });
                }
            }
            
            // Sort by score descending
            std.sort.block(Match, self.filtered_matches.items, {}, Match.lessThan);
        }
        
        // Reset selection bounds
        if (self.filtered_matches.items.len == 0) {
            self.selected_index = 0;
        } else if (self.selected_index >= self.filtered_matches.items.len) {
            self.selected_index = 0; // Reset to top on filter change usually feels better
        }
    }

    /// Run the interactive selector
    pub fn run(self: *Selector) !?types.Repository {
        try self.updateFilter();
        
        const stdout = std.io.getStdOut().writer();
        var tui_engine = try tui.Tui.init(stdout);
        try tui_engine.enableRawMode();
        defer tui_engine.disableRawMode() catch {};

        while (true) {
            try tui_engine.clear();
            
            // 1. Header with search box style
            try tui_engine.setCursor(0, 0);
            try stdout.print("{s} Search {s} {s} {s}", .{ style.bold, style.reset, self.query.items, style.cursor_block });
            
            // 2. List
            const list_start_row = 2;
            const term_height = 20; // TODO: Dynamic height would be better
            const visible_rows = term_height - list_start_row - 2;
            
            const total_matches = self.filtered_matches.items.len;
            const max_items = @min(total_matches, visible_rows);
            
            // Simple scrolling: keep selected in middle if possible
            var start_idx: usize = 0;
            if (self.selected_index > visible_rows / 2) {
                start_idx = self.selected_index - visible_rows / 2;
            }
            if (start_idx + max_items > total_matches) {
                if (total_matches > max_items) {
                    start_idx = total_matches - max_items;
                } else {
                    start_idx = 0;
                }
            }

            for (0..max_items) |i| {
                const match_idx = start_idx + i;
                if (match_idx >= total_matches) break;

                const match = self.filtered_matches.items[match_idx];
                const repo = self.repos[match.index];
                const is_selected = (match_idx == self.selected_index);
                
                try tui_engine.setCursor(list_start_row + i, 0);
                
                // Icon based on privacy
                const status_icon = if (repo.isPrivate) "🔒" else "  "; // 2 spaces for alignment
                
                if (is_selected) {
                    try stdout.print("{s}> {s} {s}{s}", .{ 
                        style.cyan, 
                        status_icon,
                        repo.nameWithOwner, 
                        style.reset 
                    });
                    
                    // Show description on new line or same line? 
                    // Let's put it on the same line if it fits, or trunc.
                    if (repo.description) |desc| {
                        // Truncate desc to roughly 50 chars for clean TUI
                        var desc_len = desc.len;
                        if (desc_len > 60) desc_len = 60;
                        try stdout.print(" {s}- {s}{s}", .{ style.dim, desc[0..desc_len], style.reset });
                    }
                } else {
                     try stdout.print("  {s} {s}{s}", .{ 
                        status_icon,
                        repo.nameWithOwner,
                        style.reset 
                    });
                     if (repo.description) |desc| {
                        var desc_len = desc.len;
                        if (desc_len > 60) desc_len = 60;
                        try stdout.print(" {s}{s}{s}", .{ style.dim, desc[0..desc_len], style.reset });
                    }
                }
            }
            
            if (total_matches == 0) {
                 try tui_engine.setCursor(list_start_row, 0);
                 try stdout.print("  {s}No matches found.{s}", .{ style.dim, style.reset });
            }
            
            // 3. Status Footer
            try tui_engine.setCursor(term_height, 0);
            try stdout.print("{s}Navigate: ↑/↓ | Select: Enter | Quit: Esc/Ctrl+C{s}", .{ style.dim, style.reset });
            try stdout.print(" {s}[{d}/{d}]{s}", .{ style.blue, self.selected_index + 1, total_matches, style.reset });

            // Handle input
            const key = try tui_engine.readKey();
            switch (key) {
                .ctrl => |c| if (c == 3) return null,
                .escape => return null,
                .enter => {
                    if (self.filtered_matches.items.len > 0) {
                        return self.repos[self.filtered_matches.items[self.selected_index].index];
                    }
                    // If no matches but hitting enter, maybe do nothing?
                },
                .up => {
                    if (self.selected_index > 0) self.selected_index -= 1;
                    // Wrap around? 
                    // else self.selected_index = total_matches - 1;
                },
                .down => {
                    if (self.selected_index + 1 < total_matches) {
                         self.selected_index += 1;
                    }
                },
                .char => |c| {
                    try self.query.append(c);
                    try self.updateFilter();
                },
                .backspace => {
                    if (self.query.items.len > 0) {
                        _ = self.query.pop();
                        try self.updateFilter();
                    }
                },
                else => {},
            }
        }
    }
};
