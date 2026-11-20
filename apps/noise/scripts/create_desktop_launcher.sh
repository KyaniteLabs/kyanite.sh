#!/bin/bash
# Script to create a desktop launcher for noise.sh on macOS

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"
DESKTOP_DIR="$HOME/Desktop"

echo "Creating desktop launcher for noise.sh..."
echo "Project directory: $PROJECT_DIR"
echo "Desktop directory: $DESKTOP_DIR"
echo

# Point the desktop launcher at the in-tree launcher script. Earlier
# versions required a "Launch noise.sh.command" file that was never in the
# tree, which made this script always exit 1.
LAUNCHER_FILE="$PROJECT_DIR/launch.sh"

if [ ! -f "$LAUNCHER_FILE" ]; then
    echo "Error: Launcher script not found at $LAUNCHER_FILE"
    exit 1
fi

# Make it executable
chmod +x "$LAUNCHER_FILE"

# Create a symlink on the desktop; the .command suffix makes macOS open it
# in Terminal on double-click.
if [ -d "$DESKTOP_DIR" ]; then
    SYMLINK="$DESKTOP_DIR/Launch noise.command"
    if [ -L "$SYMLINK" ] || [ -f "$SYMLINK" ]; then
        echo "Removing existing launcher from desktop..."
        rm "$SYMLINK"
    fi
    
    echo "Creating desktop launcher..."
    ln -s "$LAUNCHER_FILE" "$SYMLINK"
    echo "✅ Desktop launcher created at: $SYMLINK"
    echo
    echo "You can now double-click 'Launch noise.command' on your Desktop to launch the app!"
else
    echo "Warning: Desktop directory not found at $DESKTOP_DIR"
    echo "You can manually copy the launcher to your Desktop:"
    echo "  cp '$LAUNCHER_FILE' ~/Desktop/"
fi

