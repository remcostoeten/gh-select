//! Post-selection action menu UI
//! 
//! Displays after a repository is selected, allowing the user to choose
//! an action (Clone, Copy, Open, etc.)

const std = @import("std");
const types = @import("../core/types.zig");
const style = @import("../cli/style.zig");
const tui = @import("tui.zig");

pub const ActionMenu = struct {
    allocator: std.mem.Allocator,
    selected_index: usize,

    pub fn init(allocator: std.mem.Allocator) ActionMenu {
        return .{
            .allocator = allocator,
            .selected_index = 0,
        };
    }

    pub fn run(self: *ActionMenu, repo: types.Repository) !?types.Action {
        const stdout = std.io.getStdOut().writer();
        var tui_engine = try tui.Tui.init(stdout);
        try tui_engine.enableRawMode();
        defer tui_engine.disableRawMode() catch {};

        const actions = [_]types.Action{
            .clone,
            .copy_name,
            .copy_url,
            .open_browser,
            .show_name,
        };

        const titles = [_][]const u8{
            "Clone",
            "Copy Name",
            "Copy URL",
            "Open Web",
            "Show Name",
        };

        const descriptions = [_][]const u8{
            "Clone repository locally",
            "Copy owner/repo to clipboard",
            "Copy https://github.com/... URL",
            "Open repository in browser",
            "Output name to stdout and exit",
        };

        while (true) {
            try tui_engine.clear();
            const term_size = try tui_engine.getTermSize();

            // 1. Header showing selected repo
            try tui_engine.setCursor(0, 0);
            try stdout.print("{s} Selected: {s}{s}{s}\n", .{ 
                style.tn_comment, style.tn_blue, repo.nameWithOwner, style.reset 
            });
            try stdout.print("{s} What would you like to do?{s}\n", .{ style.bold, style.reset });

            // 2. Options
            const list_start_row = 3;
            
            for (actions, 0..) |action, i| {
                _ = action;
                const is_selected = (i == self.selected_index);
                try tui_engine.setCursor(list_start_row + i, 0);

                // Align: "  1. ActionName    - Description"
                // Pad ActionName to 12 chars
                
                // Manual padding to 12 chars
                const pad_len = 12 -| titles[i].len;
                
                if (is_selected) {
                    try stdout.print("{s}> {d}. {s}", .{ style.tn_magenta, i + 1, titles[i] });
                    try tui_engine.writeSpaces(pad_len);
                    try stdout.print(" {s}-{s} {s}{s}", .{
                        style.dim, style.reset,
                        descriptions[i],
                        style.reset
                    });
                } else {
                    try stdout.print("  {d}. {s}", .{ i + 1, titles[i] });
                    try tui_engine.writeSpaces(pad_len);
                    try stdout.print(" {s}-{s} {s}{s}", .{
                        style.tn_comment, style.reset,
                        descriptions[i], style.reset
                    });
                }
            }
            
            // 3. Footer
            const footer_row = term_size.rows - 2;
            try tui_engine.setCursor(footer_row, 0);
            try stdout.print("{s}Use numbers 1-5 or arrows. Enter to select. Esc to quit.{s}", .{ style.tn_comment, style.reset });

            // Input
            const key = try tui_engine.readKey();
            switch (key) {
                .ctrl => |c| if (c == 3) return null,
                .escape => return null,
                .enter => return actions[self.selected_index],
                .up => {
                    if (self.selected_index > 0) self.selected_index -= 1;
                },
                .down => {
                    if (self.selected_index < actions.len - 1) self.selected_index += 1;
                },
                .char => |c| {
                    switch (c) {
                        '1' => return actions[0],
                        '2' => return actions[1],
                        '3' => return actions[2],
                        '4' => return actions[3],
                        '5' => return actions[4],
                        'q' => return null,
                        else => {},
                    }
                },
                else => {},
            }
        }
    }
};
