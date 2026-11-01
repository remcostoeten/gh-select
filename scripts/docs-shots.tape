Output "/tmp/ghs-shots/ui.gif"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 16
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set Padding 16
Set Framerate 20
Set TypingSpeed 30ms
Set Shell bash

Hide
Type "export PATH=/tmp/ghs:$PATH GH_CONFIG_DIR=$HOME/.config/gh XDG_CONFIG_HOME=$(mktemp -d) GH_SELECT_CLONE_DIR=$(mktemp -d) GH_SELECT_NO_UPDATE_CHECK=1 PS1='$ '; . /tmp/ghs/fn.sh; cd $(mktemp -d); clear"
Enter
Show
Sleep 400ms

Type "gh select"
Enter
Sleep 2500ms
Type "is:public gh-select"
Sleep 1200ms
Screenshot docs/public/screens/list.png
Sleep 1s

Enter
Sleep 900ms
Screenshot docs/public/screens/actions.png
Sleep 1s

Type "5"
Sleep 4s
Screenshot docs/public/screens/tree.png
Sleep 1s

Type "/"
Type "README"
Enter
Sleep 600ms
Right
Sleep 1200ms
Screenshot docs/public/screens/tree-preview.png
Sleep 1s

Escape
Sleep 400ms
Space
Sleep 700ms
Screenshot docs/public/screens/tree-marked.png
Sleep 1s

Type "c"
Sleep 8s
Screenshot docs/public/screens/partial-clone.png
Sleep 1s

Type "clear; gh select"
Enter
Sleep 2000ms
Type "gh-select"
Sleep 800ms
Enter
Sleep 700ms
Type "7"
Sleep 3000ms
Screenshot docs/public/screens/releases.png
Sleep 1s

Enter
Sleep 1200ms
Screenshot docs/public/screens/release-assets.png
Sleep 1s

Escape
Sleep 300ms
Escape
Sleep 500ms
Type "6"
Sleep 2500ms
Screenshot docs/public/screens/readme.png
Sleep 1s

Escape
Sleep 400ms
Escape
Sleep 400ms
Ctrl+X
Sleep 400ms
Ctrl+D
Sleep 800ms
Screenshot docs/public/screens/delete.png
Sleep 1s

Escape
Sleep 400ms
Ctrl+X
Sleep 300ms
Ctrl+C
Sleep 300ms
Ctrl+C
Sleep 500ms

Type "clear; gh select"
Enter
Sleep 2000ms
Type "?"
Sleep 1500ms
Screenshot docs/public/screens/help.png
Sleep 1s
Escape
Sleep 300ms
Tab
Sleep 800ms
Screenshot docs/public/screens/starred.png
Sleep 1s
Tab
Sleep 500ms
Type "charmbracelet bubbletea"
Sleep 3000ms
Screenshot docs/public/screens/github-search.png
Sleep 1s
Ctrl+C
Ctrl+C
Ctrl+C
Sleep 300ms

Type "clear; gh select --theme nord"
Enter
Sleep 2000ms
Type "is:public gh-select"
Sleep 1000ms
Screenshot docs/public/screens/theme-nord.png
Sleep 1s
Ctrl+C
Ctrl+C
Sleep 300ms

Type "clear; gh select --theme tokyonight --border rounded"
Enter
Sleep 2000ms
Type "is:public gh-select"
Sleep 1000ms
Screenshot docs/public/screens/theme-tokyonight-rounded.png
Sleep 1s
Ctrl+C
Ctrl+C
Sleep 300ms
