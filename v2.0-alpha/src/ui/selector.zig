//! Interactive full-width repository selector UI

const std = @import("std");
const types = @import("../core/types.zig");
const style = @import("../cli/style.zig");
const tui = @import("tui.zig");

pub const Selector = struct {
    allocator: std.mem.Allocator,
    all_repos: []const types.Repository,
    filtered_repos: std.ArrayList(types.Repository),
    query: std.ArrayList(u8),
    selected_index: usize,
    scroll_offset: usize,

    pub fn init(allocator: std.mem.Allocator, repos: []const types.Repository) Selector {
        return .{
            .allocator = allocator,
            .all_repos = repos,
            .filtered_repos = std.ArrayList(types.Repository){},
            .query = std.ArrayList(u8){},
            .selected_index = 0,
            .scroll_offset = 0,
        };
    }

    pub fn deinit(self: *Selector) void {
        self.filtered_repos.deinit(self.allocator);
        self.query.deinit(self.allocator);
    }

    fn updateFilter(self: *Selector) !void {
        self.filtered_repos.clearRetainingCapacity();
        for (self.all_repos) |repo| {
            if (self.query.items.len == 0) {
                try self.filtered_repos.append(self.allocator, repo);
                continue;
            }

            // Simple case-insensitive fuzzy search
            if (std.ascii.indexOfIgnoreCase(repo.nameWithOwner, self.query.items)) |_| {
                try self.filtered_repos.append(self.allocator, repo);
            }
        }
        
        // Reset selection if out of bounds
        if (self.selected_index >= self.filtered_repos.items.len) {
            self.selected_index = if (self.filtered_repos.items.len > 0) self.filtered_repos.items.len - 1 else 0;
        }
    }

    /// Version-safe print helper for Zig 0.15.2
    fn print(file: std.fs.File, comptime fmt: []const u8, args: anytype) !void {
        const s = try std.fmt.allocPrint(std.heap.page_allocator, fmt, args);
        defer std.heap.page_allocator.free(s);
        try file.writeAll(s);
    }

    pub fn run(self: *Selector) !?types.Repository {
        try self.updateFilter();
        
        const stdout_file = std.fs.File{ .handle = std.posix.STDOUT_FILENO };
        
        var tui_engine = try tui.Tui.init(stdout_file);
        try tui_engine.enableRawMode();
        defer {
            tui_engine.disableRawMode() catch {};
        }

        while (true) {
            try print(stdout_file, "\x1b[2J\x1b[H", .{}); 
            
            const term_size = try tui_engine.getTermSize();
            const term_height = term_size.rows;
            const term_width = term_size.cols;

            // 1. Header
            try tui_engine.setCursor(0, 0);
            try print(stdout_file, "{s}{s} gh-select {s}{s} {s}{s} {s}", .{ 
                style.tn_bg, style.tn_blue, style.reset, 
                style.tn_fg, self.query.items, style.cursor_block, style.reset 
            });

            // 2. Repository List
            const list_start_row = 2;
            const footer_height = 1;
            const max_visible_items = if (term_height > list_start_row + footer_height) term_height - list_start_row - footer_height else 0;
            const total_matches = self.filtered_repos.items.len;

            // Adjust scroll offset
            if (self.selected_index < self.scroll_offset) {
                self.scroll_offset = self.selected_index;
            } else if (self.selected_index >= self.scroll_offset + max_visible_items) {
                self.scroll_offset = self.selected_index - max_visible_items + 1;
            }

            for (0..max_visible_items) |i| {
                const row = list_start_row + i;
                const repo_idx = self.scroll_offset + i;
                
                try tui_engine.setCursor(row, 0);
                
                if (repo_idx < total_matches) {
                    const repo = self.filtered_repos.items[repo_idx];
                    const is_selected = (repo_idx == self.selected_index);
                    
                    const prefix = if (is_selected) "> " else "  ";
                    const privacy_str = if (repo.isPrivate) "[P]" else "[ ]";
                    const name_color = if (is_selected) style.tn_magenta else style.tn_blue;
                    
                    if (is_selected) {
                        try print(stdout_file, "{s}", .{style.tn_bg});
                    }

                    try print(stdout_file, "{s}{s} {s}{s}{s} ", .{
                        prefix,
                        privacy_str,
                        name_color,
                        repo.nameWithOwner,
                        style.reset
                    });

                    // Inline description
                    if (repo.description) |desc| {
                        const current_pos = prefix.len + privacy_str.len + 1 + repo.nameWithOwner.len + 1;
                        if (current_pos < term_width - 10) {
                            const desc_limit = term_width - current_pos - 3;
                            var desc_v = desc;
                            if (desc_v.len > desc_limit) desc_v = desc_v[0..desc_limit];
                            
                            try print(stdout_file, "{s}- {s}{s}", .{
                                style.tn_comment,
                                desc_v,
                                style.reset
                            });
                        }
                    }
                    
                    if (is_selected) {
                        try print(stdout_file, "{s}", .{style.reset});
                    }
                }
            }

            // 3. Footer
            const footer_row = term_height - 1;
            try tui_engine.setCursor(footer_row, 0);
            const display_idx: usize = if (total_matches == 0) 0 else self.selected_index + 1;
            try print(stdout_file, "{s} {d}/{d} matches • ↓/j down • ↑/k up • Enter select • Esc quit{s}", .{ 
                style.tn_cyan, display_idx, total_matches, style.reset 
            });

            // Input
            const key = try tui_engine.readKey();
            switch (key) {
                .up => {
                    if (self.selected_index > 0) self.selected_index -= 1;
                },
                .down => {
                    if (self.selected_index < total_matches - 1) self.selected_index += 1;
                },
                .enter => {
                    if (total_matches > 0) return self.filtered_repos.items[self.selected_index];
                },
                .escape => return null,
                .backspace => {
                    if (self.query.items.len > 0) {
                        _ = self.query.pop();
                        try self.updateFilter();
                    }
                },
                .char => |c| {
                    if (c == 'k') {
                        if (self.selected_index > 0) self.selected_index -= 1;
                    } else if (c == 'j') {
                        if (self.selected_index < total_matches - 1) self.selected_index += 1;
                    } else if (c == 'q') {
                        return null;
                    } else {
                        try self.query.append(self.allocator, c);
                        try self.updateFilter();
                    }
                },
                else => {},
            }
        }
    }
};
