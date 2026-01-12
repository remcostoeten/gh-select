#!/usr/bin/env bash

# Helper script to manage multi-version gh-select project
# Usage: ./manage.sh <version> [command]
# <version>: v1, v2, or v3
# [command]: run (default), build

set -e

VERSION="$1"
COMMAND="$2"
ARGS="${@:3}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <v1|v2|v3> [run|build] [args...]"
    echo "Example: $0 v2 run"
    echo "Example: $0 v1 run --help"
    exit 1
fi

# Valid versions
if [[ ! "$VERSION" =~ ^v[1-3]$ ]]; then
    echo "Error: Invalid version '$VERSION'. Must be v1, v2, or v3."
    exit 1
fi

# Default command is run
if [ -z "$COMMAND" ]; then
    COMMAND="run"
fi

echo ">> Managing $VERSION with command: $COMMAND"

case "$VERSION" in
    v1)
        if [ "$COMMAND" = "build" ]; then
            echo "v1 is a shell script. No build step required."
            exit 0
        elif [ "$COMMAND" = "run" ]; then
            # Ensure script is executable
            chmod +x ./v1/gh-select
            exec ./v1/gh-select $ARGS
        else
            echo "Unknown command for v1: $COMMAND. Use 'run' or 'build'."
            exit 1
        fi
        ;;
        
    v2)
        # Check if zig is installed, prefer snap version
        ZIG_CMD=""
        if [ -x "/snap/bin/zig" ]; then
            ZIG_CMD="/snap/bin/zig"
        elif command -v zig &> /dev/null; then
            ZIG_CMD="zig"
        else
            echo "Error: 'zig' is not installed or not in PATH."
            exit 1
        fi

        cd v2.0-alpha
        echo "Using Zig binary: $ZIG_CMD"
        echo "Zig version: $($ZIG_CMD version)"
        
        if [ "$COMMAND" = "build" ]; then
            echo "Building v2 (Zig)..."
            $ZIG_CMD build
            echo "Build complete. Output in v2/zig-out/bin/"
        elif [ "$COMMAND" = "run" ]; then
            echo "Running v2 (Zig)..."
            # zig build run handles arguments after --
            if [ -n "$ARGS" ]; then
                $ZIG_CMD build run -- $ARGS
            else
                $ZIG_CMD build run
            fi
        else
            echo "Unknown command for v2: $COMMAND. Use 'run' or 'build'."
            exit 1
        fi
        ;;
        
    v3)
        # Check if go is installed
        if ! command -v go &> /dev/null; then
             echo "Error: 'go' is not installed or not in PATH."
             exit 1
        fi

        cd v3
        if [ "$COMMAND" = "build" ]; then
            echo "Building v3 (Go)..."
            go build -o gh-select ./cmd/gh-select
            echo "Built v3/gh-select"
        elif [ "$COMMAND" = "run" ]; then
            echo "Running v3 (Go)..."
            go run ./cmd/gh-select $ARGS
        else
            echo "Unknown command for v3: $COMMAND. Use 'run' or 'build'."
            exit 1
        fi
        ;;
esac
