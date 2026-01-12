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
    scroll_offset: usize,

    const Match = struct {
        index: usize,
        score: i64,
        
        fn lessThan(_: void, a: Match, b: Match) bool {
            return a.score > b.score;
        }
    };

    pub fn getSelectedRepo(self: *Selector) ?types.Repository {
        if (self.filtered_matches.items.len == 0) return null;
        var safe_idx = self.selected_index;
        if (safe_idx >= self.filtered_matches.items.len) safe_idx = 0;
        return self.repos[self.filtered_matches.items[safe_idx].index];
    }

    pub fn init(allocator: std.mem.Allocator, repos: []types.Repository) Selector {
        return .{
            .allocator = allocator,
            .repos = repos,
            .filtered_matches = std.ArrayList(Match){},
            .query = std.ArrayList(u8){},
            .selected_index = 0,
            .scroll_offset = 0,
        };
    }

    pub fn deinit(self: *Selector) void {
        self.filtered_matches.deinit(self.allocator);
        self.query.deinit(self.allocator);
    }

    /// Calculate a fuzzy match score. Returns null if no match.
    /// Higher score is better.
    fn fuzzyScore(haystack: []const u8, needle: []const u8) ?i64 {
        if (needle.len == 0) return 0;
        
        var score: i64 = 0;
        var h_idx: usize = 0;
        var n_idx: usize = 0;
        var consecutive_matches: i64 = 0;
        var first_char_bonus: bool = true;

        while (h_idx < haystack.len and n_idx < needle.len) {
            const h_char = std.ascii.toLower(haystack[h_idx]);
            const n_char = std.ascii.toLower(needle[n_idx]);

            if (h_char == n_char) {
                score += 10;
                
                if (consecutive_matches > 0) {
                    score += 5 + consecutive_matches; 
                }
                consecutive_matches += 1;

                if (h_idx == 0 or haystack[h_idx-1] == '/' or haystack[h_idx-1] == '-' or haystack[h_idx-1] == '_') {
                    score += 20;
                }
                
                if (first_char_bonus and h_idx == 0) {
                    score += 50;
                }

                n_idx += 1;
            } else {
                consecutive_matches = 0;
                first_char_bonus = false;
                score -= 1; 
            }
            h_idx += 1;
        }

        if (n_idx < needle.len) return null; 

        const len_diff = @as(i64, @intCast(haystack.len)) - @as(i64, @intCast(n_idx));
        score -= len_diff * 2;

        return score;
    }

    /// Update filter based on current query using fuzzy matching
    pub fn updateFilter(self: *Selector) !void {
        self.filtered_matches.clearRetainingCapacity();
        
        const q = self.query.items;

        if (q.len == 0) {
            try self.filtered_matches.ensureTotalCapacity(self.allocator, self.repos.len);
            for (0..self.repos.len) |i| {
                self.filtered_matches.appendAssumeCapacity(.{ .index = i, .score = 0 });
            }
        } else {
            for (self.repos, 0..) |repo, i| {
                if (fuzzyScore(repo.nameWithOwner, q)) |score| {
                    try self.filtered_matches.append(self.allocator, .{ .index = i, .score = score });
                }
            }
            std.sort.block(Match, self.filtered_matches.items, {}, Match.lessThan);
        }
        
        if (self.filtered_matches.items.len == 0) {
            self.selected_index = 0;
            self.scroll_offset = 0;
        } else if (self.selected_index >= self.filtered_matches.items.len) {
            self.selected_index = 0;
            self.scroll_offset = 0;
        } else {
            // Reset scroll on filter change to avoid confusion
            self.scroll_offset = 0;
        }
    }

    /// Run the interactive selector
    pub fn run(self: *Selector) !?types.Repository {
        try self.updateFilter();
        
        const stdout_file = std.fs.File{ .handle = std.posix.STDOUT_FILENO };
        var write_buf: [4096]u8 = undefined;
        var stdout = stdout_file.writer(&write_buf);
        
        var tui_engine = try tui.Tui.init(stdout_file);
        try tui_engine.enableRawMode();
        defer {
            // Ensure terminal is restored, log error if restore fails
            tui_engine.disableRawMode() catch |err| {
                std.io.getStdErr().writer().print("Warning: Failed to restore terminal mode: {any}\n", .{err}) catch {};
            };
        }

        while (true) {
            // Buffer the cleared frame
            try stdout.interface.print("\x1b[2J\x1b[H", .{}); 
            
            const term_size = try tui_engine.getTermSize();
            const term_height = term_size.rows;
            const term_width = term_size.cols;

            // 1. Header
            try tui_engine.setCursor(0, 0);
            try stdout.interface.print("{s}{s} gh-select {s}{s} {s}{s} {s}", .{ 
                style.tn_bg, style.tn_blue, style.reset, 
                style.tn_fg, self.query.items, style.cursor_block, style.reset 
            });
            
            // Layout Calculations
            const list_start_row = 2;
            const status_height = 3;
            const min_required = list_start_row + status_height + 1;
            
            // Guard against terminal too small / underflow
            var visible_rows: usize = if (term_height > min_required) 
                term_height - list_start_row - status_height 
            else 
                1; // Minimum 1 row to display something
            if (visible_rows > term_height) visible_rows = 10;

            const half_width = term_width / 2;
            const left_width = half_width - 1; 
            const right_start_col = half_width + 2; 
            const right_width = term_width - right_start_col;

            const total_matches = self.filtered_matches.items.len;
            const max_items = @min(total_matches, visible_rows);
            
            // Scroll logic
            if (self.selected_index < self.scroll_offset) {
                self.scroll_offset = self.selected_index;
            } else if (self.selected_index >= self.scroll_offset + visible_rows) {
                self.scroll_offset = self.selected_index - visible_rows + 1;
            }
            
            // Clamp scroll_offset
            if (total_matches <= visible_rows) {
                self.scroll_offset = 0;
            } else {
                const max_scroll = total_matches - visible_rows;
                if (self.scroll_offset > max_scroll) {
                    self.scroll_offset = max_scroll;
                }
            }

            const start_idx = self.scroll_offset;

            // --- LEFT PANE (LIST) ---
            for (0..visible_rows) |i| {
                const row = list_start_row + i;
                try tui_engine.setCursor(row, 0);
                
                if (i < max_items) {
                    const match_idx = start_idx + i;
                    const match = self.filtered_matches.items[match_idx];
                    const repo = self.repos[match.index];
                    const is_selected = (match_idx == self.selected_index);
                    
                    const status_icon = if (repo.isPrivate) "🔒" else "  ";
                    
                    // Truncate name
                    var name_limit = left_width -| 5; 
                    if (name_limit < 5) name_limit = 5;
                    var name_v = repo.nameWithOwner;
                    if (name_v.len > name_limit) name_v = name_v[0..name_limit];

                    if (is_selected) {
                        try stdout.interface.print("{s}{s} {s} {s}", .{ 
                            style.tn_bg, style.tn_magenta, 
                            status_icon, name_v 
                        });
                        try stdout.interface.print("{s}", .{style.reset});
                    } else {
                        try stdout.interface.print("  {s} {s}{s}", .{ status_icon, name_v, style.reset });
                    }
                }
            }

            // --- SEPARATOR ---
            for (0..visible_rows) |i| {
                try tui_engine.setCursor(list_start_row + i, half_width);
                try stdout.interface.print("{s}│{s}", .{style.tn_comment, style.reset});
            }

            // --- RIGHT PANE (PREVIEW) ---
            if (total_matches > 0) {
                var safe_idx = self.selected_index;
                if (safe_idx >= self.filtered_matches.items.len) safe_idx = 0;
                
                const selected_repo = self.repos[self.filtered_matches.items[safe_idx].index];
                
                var r: usize = list_start_row;
                // Name
                try tui_engine.setCursor(r, right_start_col);
                try stdout.interface.print("{s}name: {s}{s} {s}{s}", .{style.tn_comment, style.reset, style.tn_blue, selected_repo.nameWithOwner, style.reset});
                r += 1;
                
                // Visibility
                try tui_engine.setCursor(r, right_start_col);
                const vis = if (selected_repo.isPrivate) "Private 🔒" else "Public 🌍";
                try stdout.interface.print("{s}visibility: {s}{s}{s}{s}", .{style.tn_comment, style.reset, style.tn_magenta, vis, style.reset});
                r += 1;
                
                // Homepage
                if (selected_repo.homepageUrl) |url| {
                     try tui_engine.setCursor(r, right_start_col);
                     try stdout.interface.print("{s}url: {s}{s}{s}{s}", .{style.tn_comment, style.reset, style.tn_cyan, url, style.reset});
                     r += 1;
                }

                r += 1; // Spacer
                
                // Description
                if (selected_repo.description) |desc| {
                     try tui_engine.setCursor(r, right_start_col);
                     try stdout.interface.print("{s}description:{s}", .{style.tn_comment, style.reset});
                     r += 1;
                     
                     var remaining = desc;
                     // Simple wrapping
                     while (remaining.len > 0 and r < list_start_row + visible_rows) {
                         try tui_engine.setCursor(r, right_start_col);
                         const take = @min(remaining.len, right_width);
                         try stdout.interface.print(" {s}", .{remaining[0..take]});
                         
                         if (take < remaining.len) {
                             remaining = remaining[take..];
                         } else {
                             remaining = "";
                         }
                         r += 1;
                     }
                }
                
                r += 1;
                if (r < list_start_row + visible_rows) {
                     try tui_engine.setCursor(r, right_start_col);
                     try stdout.interface.print("{s}Press 'w' to view on GitHub{s}", .{style.dim, style.reset});
                }

            } else {
                try tui_engine.setCursor(list_start_row, right_start_col);
                try stdout.interface.print("{s}No repositories found.{s}", .{style.tn_comment, style.reset});
            }
            
            // Footer
            const footer_row = term_height - 3;
            try tui_engine.setCursor(footer_row, 0);
            try stdout.interface.print("{s}─{s}", .{ style.tn_comment, style.reset });
            
            try tui_engine.setCursor(footer_row + 1, 0);
<<<<<<<< HEAD:v2.0-alpha/src/ui/selector.zig
            // Fix counter display: show 0/0 when no matches, otherwise show 1-indexed position
            const display_idx: usize = if (total_matches == 0) 0 else self.selected_index + 1;
            try stdout.print("{s}{d}/{d} {s}Selections: Enter | Quit: Esc/q{s}", .{ 
                style.tn_cyan, display_idx, total_matches,
========
            try stdout.interface.print("{s}{d}/{d} {s}Selections: Enter | Quit: Esc/q{s}", .{ 
                style.tn_cyan, self.selected_index + 1, total_matches,
>>>>>>>> 8cb7599 (build: zig version mvp):v2/src/ui/selector.zig
                style.tn_comment, style.reset 
            });
            try tui_engine.setCursor(footer_row + 2, 0);
            try stdout.interface.print("{s}Quick Actions: [w]eb [r]emote [o]pen{s}", .{ style.tn_comment, style.reset });

            // FLUSH
            try stdout.interface.flush();

            // Input
            const key = try tui_engine.readKey();
            switch (key) {
                .ctrl => |c| if (c == 3) return null,
                .escape => return null,
                .enter => {
                    if (self.filtered_matches.items.len > 0) {
                         var safe_idx = self.selected_index;
                         if (safe_idx >= self.filtered_matches.items.len) safe_idx = 0;
                         return self.repos[self.filtered_matches.items[safe_idx].index];
                    }
                },
                .up => {
                    if (self.selected_index > 0) self.selected_index -= 1;
                },
                .down => {
                    if (self.selected_index + 1 < total_matches) {
                         self.selected_index += 1;
                    }
                },
                .char => |c| {
                    if (self.filtered_matches.items.len > 0) {
                        var safe_idx = self.selected_index;
                        if (safe_idx >= self.filtered_matches.items.len) safe_idx = 0;
                        const current_repo = self.repos[self.filtered_matches.items[safe_idx].index];
                        
                        // Quick actions
                        if (c == 'w') {
                            const actions = @import("actions.zig");
                            try actions.executeAction(self.allocator, .open_browser, current_repo);
                            return null; 
                        } else if (c == 'r') {
                             const actions = @import("actions.zig");
                             try actions.executeAction(self.allocator, .copy_url, current_repo);
                        } else if (c == 'o') {
                             return current_repo; 
                        } else {
                            try self.query.append(self.allocator, c);
                            try self.updateFilter();
                        }
                    } else {
<<<<<<<< HEAD:v2.0-alpha/src/ui/selector.zig
                        // No matches - still add character to query for searching
                        try self.query.append(c);
                        try self.updateFilter();
========
                        if (c != 'w' and c != 'r' and c != 'o') {
                            try self.query.append(self.allocator, c);
                            try self.updateFilter();
                        } else {
                             try self.query.append(self.allocator, c);
                             try self.updateFilter();
                        }
>>>>>>>> 8cb7599 (build: zig version mvp):v2/src/ui/selector.zig
                    }
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

// Unit tests for fuzzy scoring
test "fuzzyScore empty needle returns 0" {
    const score = Selector.fuzzyScore("anything", "");
    try std.testing.expectEqual(@as(?i64, 0), score);
}

test "fuzzyScore exact match scores high" {
    const score = Selector.fuzzyScore("hello", "hello");
    try std.testing.expect(score != null);
    try std.testing.expect(score.? > 0);
}

test "fuzzyScore case insensitive" {
    const score1 = Selector.fuzzyScore("Hello", "hello");
    const score2 = Selector.fuzzyScore("hello", "HELLO");
    try std.testing.expect(score1 != null);
    try std.testing.expect(score2 != null);
}

test "fuzzyScore no match returns null" {
    const score = Selector.fuzzyScore("abc", "xyz");
    try std.testing.expectEqual(@as(?i64, null), score);
}

test "fuzzyScore subsequence match" {
    // "ac" is a subsequence of "abc"
    const score = Selector.fuzzyScore("abc", "ac");
    try std.testing.expect(score != null);
}

test "fuzzyScore prefers word boundary matches" {
    // Matching at word boundary "g" in "gh-select" should score higher
    const score_boundary = Selector.fuzzyScore("gh-select", "gs");
    const score_middle = Selector.fuzzyScore("aghselectb", "gs");
    try std.testing.expect(score_boundary != null);
    try std.testing.expect(score_middle != null);
    try std.testing.expect(score_boundary.? > score_middle.?);
}

test "fuzzyScore consecutive matches score higher" {
    // "ab" consecutive in "abc" should score higher than spread in "aXbXc"
    const score_consecutive = Selector.fuzzyScore("abc", "ab");
    const score_spread = Selector.fuzzyScore("aXb", "ab");
    try std.testing.expect(score_consecutive != null);
    try std.testing.expect(score_spread != null);
    try std.testing.expect(score_consecutive.? > score_spread.?);
}

test "fuzzyScore shorter haystack preferred" {
    // Exact match in shorter string should beat match in longer string
    const score_short = Selector.fuzzyScore("cat", "cat");
    const score_long = Selector.fuzzyScore("category", "cat");
    try std.testing.expect(score_short != null);
    try std.testing.expect(score_long != null);
    try std.testing.expect(score_short.? > score_long.?);
}
