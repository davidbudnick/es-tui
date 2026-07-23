# es-tui demo — opens on connections (main page), then tours dense screens
Output "docs/main.gif"

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
Set PlaybackSpeed 1.0
# Start the loop on the connections (main) page, not mid-tour.
Set LoopOffset 0%

# Isolated config + force truecolor (agents often set NO_COLOR=1 / TERM=dumb).
Hide
Type "export NO_COLOR= CLICOLOR_FORCE=1 COLORTERM=truecolor TERM=xterm-256color DEMO_HOME=$(mktemp -d) && mkdir -p $DEMO_HOME/.config/es-tui && cp docs/demo-config.json $DEMO_HOME/.config/es-tui/config.json"
Enter
Sleep 0.4s
Type "HOME=$DEMO_HOME ./bin/es-tui"
Enter
Sleep 2s
Show

# === MAIN PAGE: connections with multicolor ELASTIC logo ===
Sleep 5s

# Connect Elasticsearch → dense index list
Enter
Sleep 3s

# Full index list (many indices)
Sleep 2s

# logs-app documents (dense rows + JSON preview)
Type "f"
Sleep 0.2s
Type "logs-app"
Sleep 0.3s
Enter
Sleep 1s
Enter
Sleep 3.5s

# Document detail
Enter
Sleep 3s
Escape
Sleep 0.7s

# Back to indices
Escape
Sleep 0.6s
Type "f"
Sleep 0.2s
Backspace 12
Type "*"
Sleep 0.2s
Enter
Sleep 2s

# Nodes
Type "n"
Sleep 2.2s
Escape
Sleep 0.5s

# Live metrics
Type "m"
Sleep 6s
Escape
Sleep 0.5s

# Products + search
Type "f"
Sleep 0.2s
Type "products"
Sleep 0.25s
Enter
Sleep 1s
Enter
Sleep 2.5s
Type "/"
Sleep 0.25s
Type "tags:merch"
Sleep 0.3s
Enter
Sleep 2.5s

# Back to main page → OpenSearch
Escape
Sleep 0.5s
Escape
Sleep 0.6s
Escape
Sleep 1s
Down
Sleep 0.4s
Enter
Sleep 3s
