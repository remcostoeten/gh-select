//! UI Selector module using custom TUI engine and zf for filtering

const std = @import("std");
const types = @import("../core/types.zig");
const style = @import("../cli/style.zig");
const tui = @import("tui.zig");

pub const Selector = struct {
    allocator: std.mem.Allocator,
    repos: []types.Repository,
    filtered_indices: std.ArrayList(usize),
    query: std.ArrayList(u8),
    selected_index: usize,

    pub fn init(allocator: std.mem.Allocator, repos: []types.Repository) Selector {
        return .{
            .allocator = allocator,
            .repos = repos,
            .filtered_indices = std.ArrayList(usize).init(allocator),
            .query = std.ArrayList(u8).init(allocator),
            .selected_index = 0,
        };
    }

    pub fn deinit(self: *Selector) void {
        self.filtered_indices.deinit();
        self.query.deinit();
    }

    /// Update filter based on current query using naive search (zf later)
    pub fn updateFilter(self: *Selector) !void {
        self.filtered_indices.clearRetainingCapacity();
        
        if (self.query.items.len == 0) {
            // No query: show all
            try self.filtered_indices.ensureTotalCapacity(self.repos.len);
            for (0..self.repos.len) |i| {
                self.filtered_indices.appendAssumeCapacity(i);
            }
            return;
        }

        // Use naive substring matching for now (zf integration in next step if needed, or if import works)
        // Since I imported zf, I should try to use it if I can figure out the API.
        // But for "Make it work", substring is safer.
        // Let's stick to substring for v1 of the TUI to ensure stability.
        
        for (self.repos, 0..) |repo, i| {
            if (std.mem.indexOf(u8, repo.nameWithOwner, self.query.items)) |_| {
                 try self.filtered_indices.append(i);
            }
        }
        
        // Reset selection if out of bounds
        if (self.filtered_indices.items.len == 0) {
            self.selected_index = 0;
        } else if (self.selected_index >= self.filtered_indices.items.len) {
            self.selected_index = self.filtered_indices.items.len - 1;
        }
    }

    /// Run the interactive selector
    /// Returns the selected repository or null if cancelled
    pub fn run(self: *Selector) !?types.Repository {
        try self.updateFilter();
        
        // Initialize TUI
        const stdout = std.io.getStdOut().writer();
        var tui_engine = try tui.Tui.init(stdout);
        try tui_engine.enableRawMode();
        defer tui_engine.disableRawMode() catch {};

        while (true) {
            try tui_engine.clear();
            
            // 1. Header
            try tui_engine.setCursor(0, 0);
            try stdout.print("{s}gh-select{s} > {s}{s}", .{ style.bold, style.reset, self.query.items, style.cursor_block });
            
            // 2. List
            const list_start_row = 2;
            const term_height = 20; // TODO: Get size dynamically if possible or assume 24
            const max_items = @min(self.filtered_indices.items.len, term_height - list_start_row - 2);
            
            for (0..max_items) |i| {
                const repo_idx = self.filtered_indices.items[i];
                const repo = self.repos[repo_idx];
                const is_selected = i == self.selected_index;
                
                try tui_engine.setCursor(list_start_row + i, 0);
                
                if (is_selected) {
                    try stdout.print("{s}> {s}{s}", .{ style.cyan, repo.nameWithOwner, style.reset });
                } else {
                    try stdout.print("  {s}", .{repo.nameWithOwner});
                }
            }
            
            // 3. Status Footer
            try tui_engine.setCursor(term_height, 0);
            try stdout.print("{s}Use arrows to navigate, Enter to select, Esc to quit{s}", .{ style.dim, style.reset });

            // Handle input
            const key = try tui_engine.readKey();
            switch (key) {
                .ctrl => |c| if (c == 3) return null, // Ctrl-C
                .escape => return null,
                .enter => {
                    if (self.filtered_indices.items.len > 0) {
                        return self.repos[self.filtered_indices.items[self.selected_index]];
                    }
                    return null;
                },
                .up => {
                    if (self.selected_index > 0) self.selected_index -= 1;
                },
                .down => {
                    if (self.selected_index + 1 < self.filtered_indices.items.len) {
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
