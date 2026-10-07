Output "/tmp/claude-1000/-home-remcostoeten/ad56559f-c03a-4ec3-a0f0-f7014c50bda4/scratchpad/demo/blog.mp4"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 18
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set Padding 16
Set Framerate 30
Set TypingSpeed 35ms
Set Shell bash
Hide
Type "export GH_SELECT_CLONE_DIR=$(mktemp -d) XDG_CONFIG_HOME=$(mktemp -d) XDG_DATA_HOME=$(mktemp -d) GH_CONFIG_DIR=$HOME/.config/gh PS1='$ '; clear"
Enter
Show
Type "gh extension install remcostoeten/gh-select"
Enter
Sleep 2500ms
Type "gh select"
Enter
Sleep 1500ms
Tab
Sleep 300ms
Tab
Type "charmbracelet bubbletea"
Sleep 2s
Enter
Sleep 1s
Type "4"
Sleep 1500ms
Type "/examples"
Sleep 500ms
Enter
Sleep 500ms
Space
Sleep 500ms
Escape
Type "/tutorials"
Sleep 500ms
Enter
Sleep 500ms
Space
Sleep 1s
Type "c"
Sleep 5s
Type "clear; cd $GH_SELECT_CLONE_DIR/bubbletea && ls -d */"
Enter
Sleep 3s
Hide
