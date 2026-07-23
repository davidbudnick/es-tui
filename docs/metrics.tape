# es-tui metrics still — hold long enough for history/sparklines
Output "docs/metrics.gif"

Require ./bin/es-tui

Set Shell "zsh"
Set FontSize 18
Set FontFamily "SFMono Nerd Font"
Set Theme "Dracula"
Set Width 1920
Set Height 1080
Set TypingSpeed 0.03s
Set Padding 20
Set Framerate 24

# Force truecolor: parent shells often set NO_COLOR=1 / TERM=dumb.
Hide
Type "export NO_COLOR= CLICOLOR_FORCE=1 COLORTERM=truecolor TERM=xterm-256color DEMO_HOME=$(mktemp -d) && mkdir -p $DEMO_HOME/.config/es-tui && cp docs/demo-config.json $DEMO_HOME/.config/es-tui/config.json"
Enter
Sleep 0.4s
Type "HOME=$DEMO_HOME ./bin/es-tui"
Enter
Sleep 1.8s
Show
Sleep 1.2s
Enter
Sleep 2s
Type "m"
# Live metrics refreshes every 2s — collect several samples for sparklines.
Sleep 10s
