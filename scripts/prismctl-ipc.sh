#!/usr/bin/env bash
# Helper script for sending IPC commands to prismctl

set -euo pipefail

SOCKET_DIR="/run/user/$(id -u)/shine"

usage() {
    cat <<EOF
Usage: $0 <socket-name> <action> [prism-name]

Commands:
  $0 <socket> start <prism-name>  # Start/resume prism (idempotent)
  $0 <socket> kill <prism-name>   # Kill prism (auto-resumes next)
  $0 <socket> status               # Query current status
  $0 <socket> stop                 # Stop prismctl

Socket name format: prism-<instance>.sock

Examples:
  $0 prism-clock.sock start shine-spotify
  $0 prism-bar.sock kill shine-clock
  $0 prism-chat.sock status
  $0 prism-spotify.sock stop

List available sockets:
  ls -la $SOCKET_DIR/
EOF
    exit 1
}

if [ $# -lt 2 ]; then
    usage
fi

SOCKET_NAME="$1"
ACTION="$2"
PRISM="${3:-}"

SOCKET_PATH="$SOCKET_DIR/$SOCKET_NAME"

if [ ! -S "$SOCKET_PATH" ]; then
    echo "Error: Socket not found: $SOCKET_PATH" >&2
    echo "Available sockets:" >&2
    ls -la "$SOCKET_DIR/" 2>/dev/null || echo "  (none)" >&2
    exit 1
fi

case "$ACTION" in
    start)
        if [ -z "$PRISM" ]; then
            echo "Error: prism name required for start action" >&2
            exit 1
        fi
        CMD="{\"action\":\"start\",\"prism\":\"$PRISM\"}"
        ;;
    kill)
        if [ -z "$PRISM" ]; then
            echo "Error: prism name required for kill action" >&2
            exit 1
        fi
        CMD="{\"action\":\"kill\",\"prism\":\"$PRISM\"}"
        ;;
    status)
        CMD="{\"action\":\"status\"}"
        ;;
    stop)
        CMD="{\"action\":\"stop\"}"
        ;;
    *)
        echo "Error: unknown action: $ACTION" >&2
        usage
        ;;
esac

echo "Sending command to $SOCKET_PATH: $CMD"
echo "$CMD" | socat - "UNIX-CONNECT:$SOCKET_PATH"
