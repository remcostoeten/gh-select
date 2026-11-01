Output "/tmp/ghs-shots/cli.gif"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 16
Set Width 1200
Set Height 1250
Set Theme "Catppuccin Mocha"
Set Padding 16
Set Framerate 20
Set TypingSpeed 30ms
Set Shell bash

Hide
Type "export PATH=/tmp/ghs:$PATH GH_CONFIG_DIR=$HOME/.config/gh XDG_CONFIG_HOME=$(mktemp -d) GH_SELECT_NO_UPDATE_CHECK=1 PS1='$ '; . /tmp/ghs/fn.sh; cd $(mktemp -d); clear"
Enter
Show
Sleep 400ms

Type "gh select doctor"
Enter
Sleep 2s
Screenshot docs/public/screens/doctor.png
Sleep 1s

Hide
Type "clear"
Enter
Show
Type "gh select limits"
Enter
Sleep 3s
Screenshot docs/public/screens/limits.png
Sleep 1s

Hide
Type "clear"
Enter
Show
Type "gh select help"
Enter
Sleep 2s
Screenshot docs/public/screens/help-cli.png
Sleep 1s

Hide
Type "clear"
Enter
Show
Type "gh select refresh"
Enter
Sleep 8s
Screenshot docs/public/screens/refresh.png
Sleep 1s

Hide
Type "clear"
Enter
Show
Type "gh select limts"
Enter
Sleep 1500ms
Screenshot docs/public/screens/typo.png
Sleep 1s
