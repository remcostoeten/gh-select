#!/bin/bash

# Uninstallation script for gh-select extension
# Removes both user and global installations

set -e

# Parse command line arguments
GLOBAL_UNINSTALL=false
FORCE_UNINSTALL=false
QUIET=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --global)
            GLOBAL_UNINSTALL=true
            shift
            ;;
        --force)
            FORCE_UNINSTALL=true
            shift
            ;;
        --quiet|-q)
            QUIET=true
            shift
            ;;
        --help|-h)
            echo "gh-select Uninstallation Script"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --global       Remove global installation (requires sudo)"
            echo "  --force        Remove all installations without confirmation"
            echo "  --quiet, -q    Suppress output messages"
            echo "  --help, -h     Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                    # Remove user installation (GitHub CLI extension)"
            echo "  $0 --global          # Remove global installation"
            echo "  $0 --force           # Remove all installations found"
            echo "  sudo $0 --global     # Remove global installation (with sudo)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

log() {
    if [ "$QUIET" = false ]; then
        echo "$1"
    fi
}

log "🗑️  gh-select Uninstaller"
log ""

# Track what was found/removed
USER_REMOVED=false
GLOBAL_REMOVED=false
FOUND_INSTALLATIONS=false

# Check for user installation (GitHub CLI extension)
USER_EXTENSION_DIR="$HOME/.local/share/gh/extensions/gh-select"
if [ -d "$USER_EXTENSION_DIR" ]; then
    FOUND_INSTALLATIONS=true
    if [ "$GLOBAL_UNINSTALL" = false ]; then
        log "📍 Found user installation: $USER_EXTENSION_DIR"
        
        if [ "$FORCE_UNINSTALL" = false ]; then
            log "Remove GitHub CLI extension installation? [y/N]"
            read -r response
            case $response in
                [yY]|[yY][eE][sS])
                    ;;
                *)
                    log "❌ User installation kept"
                    ;;
            esac
        fi
        
        if [ "$FORCE_UNINSTALL" = true ] || [[ $response =~ ^[yY]([eE][sS])?$ ]]; then
            # Use GitHub CLI to remove the extension
            if command -v gh > /dev/null 2>&1; then
                if gh extension remove select > /dev/null 2>&1; then
                    log "✅ Removed GitHub CLI extension"
                    USER_REMOVED=true
                else
                    # Fallback to manual removal
                    rm -rf "$USER_EXTENSION_DIR"
                    log "✅ Manually removed user installation"
                    USER_REMOVED=true
                fi
            else
                # Manual removal if gh CLI not available
                rm -rf "$USER_EXTENSION_DIR"
                log "✅ Manually removed user installation"
                USER_REMOVED=true
            fi
        fi
    fi
fi

# Check for global installation
GLOBAL_PATHS=("/usr/local/bin/gh-select" "/usr/bin/gh-select")
for global_path in "${GLOBAL_PATHS[@]}"; do
    if [ -f "$global_path" ]; then
        FOUND_INSTALLATIONS=true
        if [ "$GLOBAL_UNINSTALL" = true ]; then
            log "📍 Found global installation: $global_path"
            
            # Check for root privileges
            if [ "$EUID" -ne 0 ]; then
                log "❌ Global uninstallation requires root privileges"
                log "Please run: sudo $0 --global"
                exit 1
            fi
            
            if [ "$FORCE_UNINSTALL" = false ]; then
                log "Remove global installation? [y/N]"
                read -r response
                case $response in
                    [yY]|[yY][eE][sS])
                        ;;
                    *)
                        log "❌ Global installation kept"
                        continue
                        ;;
                esac
            fi
            
            if [ "$FORCE_UNINSTALL" = true ] || [[ $response =~ ^[yY]([eE][sS])?$ ]]; then
                rm -f "$global_path"
                log "✅ Removed global installation: $global_path"
                GLOBAL_REMOVED=true
            fi
        elif [ "$GLOBAL_UNINSTALL" = false ]; then
            log "📍 Found global installation: $global_path"
            log "   Use --global flag to remove global installation"
        fi
    fi
done

# Summary
log ""
if [ "$FOUND_INSTALLATIONS" = false ]; then
    log "ℹ️  No gh-select installations found"
elif [ "$USER_REMOVED" = true ] || [ "$GLOBAL_REMOVED" = true ]; then
    log "🎉 Uninstallation complete!"
    log ""
    if [ "$USER_REMOVED" = true ]; then
        log "✅ Removed user installation (GitHub CLI extension)"
        log "   'gh select' command no longer available"
    fi
    if [ "$GLOBAL_REMOVED" = true ]; then
        log "✅ Removed global installation"
        log "   'gh-select' command no longer available"
    fi
    log ""
    log "💡 To reinstall:"
    log "   gh extension install remcostoeten/gh-select"
else
    log "ℹ️  No installations were removed"
fi

# Verify removal
log ""
log "🔍 Verification:"
if command -v gh > /dev/null 2>&1; then
    if gh extension list 2>/dev/null | grep -q "gh select"; then
        log "⚠️  GitHub CLI extension still present"
    else
        log "✅ GitHub CLI extension removed"
    fi
fi

if command -v gh-select > /dev/null 2>&1; then
    log "⚠️  Global command still available: $(which gh-select)"
else
    log "✅ Global command removed"
fi
