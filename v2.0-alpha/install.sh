#!/usr/bin/env bash
#
# gh-select Interactive Installer
# ================================
# Author: Remco Stoeten
# Repo: https://github.com/remcostoeten/gh-select
#

set -euo pipefail

# ─────────────────────────────────────────────────────────────────────────────
# Configuration
# ─────────────────────────────────────────────────────────────────────────────
VERSION="2.0.0-alpha.1"
REPO_URL="https://github.com/remcostoeten/gh-select"
REPO_RAW="https://raw.githubusercontent.com/remcostoeten/gh-select"
INSTALL_DIR="${HOME}/.local/bin"
EXTENSION_DIR="${HOME}/.local/share/gh/extensions/gh-select"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
DIM='\033[2m'
BOLD='\033[1m'
RESET='\033[0m'
REVERSE='\033[7m'

# Menu items
MENU_ITEMS=(
    "Install gh-select"
    "Uninstall gh-select"
    "Build locally from source"
    "Check for source updates"
    "View source on GitHub"
    "Help & About"
)

MENU_ICONS=(
    "📦"
    "🗑️ "
    "🔨"
    "🔄"
    "🌐"
    "❓"
)

# State
SELECTED_INDEX=0
SEARCH_QUERY=""
FILTERED_INDICES=()

# ─────────────────────────────────────────────────────────────────────────────
# Utilities
# ─────────────────────────────────────────────────────────────────────────────

log_info() { echo -e "${CYAN}ℹ${RESET} $1"; }
log_success() { echo -e "${GREEN}✔${RESET} $1"; }
log_warn() { echo -e "${YELLOW}⚠${RESET} $1"; }
log_error() { echo -e "${RED}✖${RESET} $1"; }

press_any_key() {
    echo ""
    echo -e "${DIM}Press any key to continue...${RESET}"
    read -rsn1
}

confirm() {
    local prompt="${1:-Continue?}"
    local default="${2:-y}"
    
    if [[ "$default" == "y" ]]; then
        echo -en "${YELLOW}?${RESET} ${prompt} ${DIM}[Y/n]${RESET} "
    else
        echo -en "${YELLOW}?${RESET} ${prompt} ${DIM}[y/N]${RESET} "
    fi
    
    read -rsn1 answer
    echo "$answer"
    
    [[ -z "$answer" ]] && answer="$default"
    [[ "${answer,,}" == "y" ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Terminal Control
# ─────────────────────────────────────────────────────────────────────────────

hide_cursor() { printf '\033[?25l'; }
show_cursor() { printf '\033[?25h'; }
clear_screen() { printf '\033[2J\033[H'; }
move_cursor() { printf '\033[%d;%dH' "$1" "$2"; }

cleanup() {
    show_cursor
    stty echo 2>/dev/null || true
    echo ""
}
trap cleanup EXIT

# Read a single keypress
read_key() {
    local key
    IFS= read -rsn1 key
    
    # Handle escape sequences (arrow keys)
    if [[ "$key" == $'\x1b' ]]; then
        read -rsn2 -t 0.1 key2 || true
        key+="$key2"
    fi
    
    echo "$key"
}

# ─────────────────────────────────────────────────────────────────────────────
# Menu Filtering (Fuzzy Search)
# ─────────────────────────────────────────────────────────────────────────────

update_filter() {
    FILTERED_INDICES=()
    
    if [[ -z "$SEARCH_QUERY" ]]; then
        for i in "${!MENU_ITEMS[@]}"; do
            FILTERED_INDICES+=("$i")
        done
    else
        local query_lower="${SEARCH_QUERY,,}"
        for i in "${!MENU_ITEMS[@]}"; do
            local item_lower="${MENU_ITEMS[$i],,}"
            if [[ "$item_lower" == *"$query_lower"* ]]; then
                FILTERED_INDICES+=("$i")
            fi
        done
    fi
    
    # Reset selection if out of bounds
    if (( SELECTED_INDEX >= ${#FILTERED_INDICES[@]} )); then
        SELECTED_INDEX=0
    fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Menu Rendering
# ─────────────────────────────────────────────────────────────────────────────

render_header() {
    echo -e "${BOLD}${CYAN}"
    echo "   ╔══════════════════════════════════════════╗"
    echo "   ║       gh-select Installer v${VERSION}      ║"
    echo "   ╚══════════════════════════════════════════╝"
    echo -e "${RESET}"
    echo -e "   ${DIM}Use ↑/↓ or 1-6 to navigate, Enter to select${RESET}"
    echo -e "   ${DIM}Type to search, Backspace to clear, q to quit${RESET}"
    echo ""
}

render_search() {
    if [[ -n "$SEARCH_QUERY" ]]; then
        echo -e "   ${YELLOW}Search:${RESET} ${SEARCH_QUERY}█"
    else
        echo -e "   ${DIM}Search: (type to filter)${RESET}"
    fi
    echo ""
}

render_menu() {
    local visible_count=${#FILTERED_INDICES[@]}
    
    if (( visible_count == 0 )); then
        echo -e "   ${DIM}No matches found${RESET}"
        echo ""
        return
    fi
    
    for i in "${!FILTERED_INDICES[@]}"; do
        local real_index="${FILTERED_INDICES[$i]}"
        local item="${MENU_ITEMS[$real_index]}"
        local icon="${MENU_ICONS[$real_index]}"
        local num=$((real_index + 1))
        
        if (( i == SELECTED_INDEX )); then
            echo -e "   ${REVERSE}${BOLD} ${icon} ${num}. ${item} ${RESET}"
        else
            echo -e "   ${DIM}${icon}${RESET} ${num}. ${item}"
        fi
    done
    echo ""
}

render_footer() {
    echo -e "   ${DIM}─────────────────────────────────────────────${RESET}"
    echo -e "   ${DIM}Enter: Select | Backspace: Clear | q: Quit${RESET}"
}

draw_menu() {
    clear_screen
    render_header
    render_search
    render_menu
    render_footer
}

# ─────────────────────────────────────────────────────────────────────────────
# Menu Actions
# ─────────────────────────────────────────────────────────────────────────────

get_installed_version() {
    if command -v gh-select &>/dev/null; then
        gh-select --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+[^[:space:]]*' | head -1 || echo "unknown"
    elif [[ -x "${INSTALL_DIR}/gh-select" ]]; then
        "${INSTALL_DIR}/gh-select" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+[^[:space:]]*' | head -1 || echo "unknown"
    else
        echo ""
    fi
}

action_install() {
    clear_screen
    echo -e "${BOLD}${CYAN}📦 Install gh-select${RESET}\n"
    
    local installed_version
    installed_version=$(get_installed_version)
    
    if [[ -n "$installed_version" ]]; then
        log_warn "Installation detected: gh-select v${installed_version}"
        echo ""
        
        if confirm "Remove existing installation and install v${VERSION}?" "y"; then
            echo ""
            action_uninstall_silent
        else
            log_info "Installation cancelled."
            press_any_key
            return
        fi
    fi
    
    echo ""
    log_info "Installing gh-select v${VERSION}..."
    
    # Check for Zig
    if ! command -v zig &>/dev/null; then
        log_error "Zig is not installed. Please install Zig first:"
        echo -e "   ${DIM}https://ziglang.org/download/${RESET}"
        press_any_key
        return 1
    fi
    
    # Build from source
    log_info "Building from source..."
    if zig build -Doptimize=ReleaseSafe; then
        log_success "Build successful!"
    else
        log_error "Build failed."
        press_any_key
        return 1
    fi
    
    # Install binary
    mkdir -p "$INSTALL_DIR"
    cp "./zig-out/bin/gh-select" "$INSTALL_DIR/"
    chmod +x "${INSTALL_DIR}/gh-select"
    
    log_success "Installed to ${INSTALL_DIR}/gh-select"
    
    # Check PATH
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        log_warn "${INSTALL_DIR} is not in your PATH."
        echo -e "   ${DIM}Add this to your shell config:${RESET}"
        echo -e "   ${CYAN}export PATH=\"\$PATH:${INSTALL_DIR}\"${RESET}"
    fi
    
    log_success "Installation complete!"
    press_any_key
}

action_uninstall_silent() {
    log_info "Uninstalling gh-select..."
    
    # Remove from local bin
    if [[ -f "${INSTALL_DIR}/gh-select" ]]; then
        rm -f "${INSTALL_DIR}/gh-select"
        log_success "Removed ${INSTALL_DIR}/gh-select"
    fi
    
    # Remove gh extension if exists
    if [[ -d "$EXTENSION_DIR" ]]; then
        rm -rf "$EXTENSION_DIR"
        log_success "Removed gh extension"
    fi
    
    # Try gh extension uninstall
    if command -v gh &>/dev/null; then
        gh extension remove select 2>/dev/null || true
    fi
}

action_uninstall() {
    clear_screen
    echo -e "${BOLD}${RED}Uninstall gh-select${RESET}\n"
    
    local installed_version
    installed_version=$(get_installed_version)
    
    if [[ -z "$installed_version" ]]; then
        log_info "gh-select is not installed."
        press_any_key
        return
    fi
    
    log_info "Found gh-select v${installed_version}"
    echo ""
    
    if ! confirm "Are you sure you want to uninstall?" "n"; then
        log_info "Uninstall cancelled."
        press_any_key
        return
    fi
    
    echo ""
    action_uninstall_silent
    
    echo ""
    log_success "Uninstall complete. Goodbye! 👋"
    press_any_key
}

action_build_local() {
    clear_screen
    echo -e "${BOLD}${YELLOW}Build Locally${RESET}\n"
    
    # Check for Zig
    if ! command -v zig &>/dev/null; then
        log_error "Zig is not installed. Please install Zig first:"
        echo -e "   ${DIM}https://ziglang.org/download/${RESET}"
        press_any_key
        return 1
    fi
    
    log_info "Building gh-select from source..."
    echo ""
    
    if zig build; then
        echo ""
        log_success "Local build succeeded!"
        log_info "Binary: ./zig-out/bin/gh-select"
        echo ""
        
        if confirm "Run gh-select @v${VERSION}-local-build now?" "y"; then
            echo ""
            ./zig-out/bin/gh-select || true
        else
            echo ""
            log_info "You can run it later with:"
            echo -e "   ${CYAN}./zig-out/bin/gh-select${RESET}"
        fi
    else
        log_error "Build failed. Check the errors above."
    fi
    
    press_any_key
}

action_check_updates() {
    clear_screen
    echo -e "${BOLD}${BLUE}Check for Source Updates${RESET}\n"
    
    log_info "Checking for updates from ${REPO_URL}..."
    echo ""
    
    # Detect the default branch (main or master)
    local default_branch
    default_branch=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@') || true
    if [[ -z "$default_branch" ]]; then
        # Fallback: try 'main' first, then 'master'
        if git show-ref --verify --quiet refs/remotes/origin/main 2>/dev/null; then
            default_branch="main"
        else
            default_branch="master"
        fi
    fi
    
    # Fetch latest
    if ! git fetch origin "$default_branch" --quiet 2>/dev/null; then
        log_error "Failed to fetch from remote. Are you in the git repository?"
        press_any_key
        return 1
    fi
    
    # Check for differences
    local local_commit remote_commit
    local_commit=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    remote_commit=$(git rev-parse "origin/$default_branch" 2>/dev/null || echo "unknown")
    
    if [[ "$local_commit" == "$remote_commit" ]]; then
        log_success "You are up to date!"
        echo -e "   ${DIM}Current commit: ${local_commit:0:8}${RESET}"
    else
        log_warn "Updates available!"
        echo ""
        echo -e "   ${DIM}Local:  ${local_commit:0:8}${RESET}"
        echo -e "   ${DIM}Remote: ${remote_commit:0:8}${RESET}"
        echo ""
        
        # Show diff summary
        log_info "Changes:"
        git --no-pager diff --stat "HEAD..origin/$default_branch" 2>/dev/null | head -10 || true
        echo ""
        
        if confirm "Pull latest changes?" "y"; then
            echo ""
            if git pull origin "$default_branch"; then
                log_success "Updated successfully!"
            else
                log_error "Pull failed. You may have local changes."
            fi
        fi
    fi
    
    press_any_key
}

action_view_source() {
    clear_screen
    echo -e "${BOLD}${MAGENTA}View Source${RESET}\n"
    
    log_info "Opening ${REPO_URL} ..."
    
    # Try to open in browser
    if command -v xdg-open &>/dev/null; then
        xdg-open "$REPO_URL" 2>/dev/null &
        log_success "Opened in browser."
    elif command -v open &>/dev/null; then
        open "$REPO_URL" 2>/dev/null &
        log_success "Opened in browser."
    else
        log_warn "Could not detect browser. Visit:"
        echo -e "   ${CYAN}${REPO_URL}${RESET}"
    fi
    
    press_any_key
}

action_help() {
    clear_screen
    echo -e "${BOLD}${WHITE}Help & About${RESET}\n"
    
    echo -e "${CYAN}gh-select${RESET} v${VERSION}"
    echo -e "${DIM}Interactive GitHub Repository Selector${RESET}"
    echo ""
    echo -e "${BOLD}Author${RESET}"
    echo -e "   Remco Stoeten"
    echo -e "   ${DIM}@remcostoeten${RESET}"
    echo -e "   ${CYAN}https://remcostoeten.nl${RESET}"
    echo ""
    echo -e "${BOLD}Repository${RESET}"
    echo -e "   ${CYAN}${REPO_URL}${RESET}"
    echo ""
    echo -e "${BOLD}Description${RESET}"
    echo "   A blazing-fast GitHub repository selector written in Zig."
    echo "   Fuzzy-find and manage your repos directly from the terminal."
    echo ""
    echo -e "${BOLD}Requirements${RESET}"
    echo "   • Zig >= 0.13.0 (for building)"
    echo "   • gh CLI (for GitHub API access)"
    echo ""
    echo -e "${BOLD}License${RESET}"
    echo "   MIT"
    echo ""
    
    press_any_key
}

execute_action() {
    local real_index="${FILTERED_INDICES[$SELECTED_INDEX]}"
    
    case "$real_index" in
        0) action_install ;;
        1) action_uninstall ;;
        2) action_build_local ;;
        3) action_check_updates ;;
        4) action_view_source ;;
        5) action_help ;;
    esac
}

# ─────────────────────────────────────────────────────────────────────────────
# Main Loop
# ─────────────────────────────────────────────────────────────────────────────

main() {
    hide_cursor
    update_filter
    
    while true; do
        draw_menu
        
        local key
        key=$(read_key)
        
        case "$key" in
            # Arrow keys
            $'\x1b[A') # Up
                (( SELECTED_INDEX > 0 )) && (( SELECTED_INDEX-- )) || true
                ;;
            $'\x1b[B') # Down
                (( SELECTED_INDEX < ${#FILTERED_INDICES[@]} - 1 )) && (( SELECTED_INDEX++ )) || true
                ;;
            
            # Number keys (1-6)
            [1-6])
                local target_index=$((key - 1))
                # Find if this index is in filtered list
                for i in "${!FILTERED_INDICES[@]}"; do
                    if (( FILTERED_INDICES[i] == target_index )); then
                        SELECTED_INDEX=$i
                        execute_action
                        break
                    fi
                done
                ;;
            
            # Enter
            ''|$'\n')
                if (( ${#FILTERED_INDICES[@]} > 0 )); then
                    execute_action
                fi
                ;;
            
            # Backspace
            $'\x7f'|$'\b')
                if [[ -n "$SEARCH_QUERY" ]]; then
                    SEARCH_QUERY="${SEARCH_QUERY%?}"
                    update_filter
                fi
                ;;
            
            # Escape - clear search or quit
            $'\x1b')
                if [[ -n "$SEARCH_QUERY" ]]; then
                    SEARCH_QUERY=""
                    update_filter
                else
                    break
                fi
                ;;
            
            # q/Q - quit
            q|Q)
                if [[ -z "$SEARCH_QUERY" ]]; then
                    break
                else
                    SEARCH_QUERY+="$key"
                    update_filter
                fi
                ;;
            
            # Spacebar - also select
            ' ')
                if (( ${#FILTERED_INDICES[@]} > 0 )); then
                    execute_action
                fi
                ;;
            
            # Any other printable character - search
            [[:print:]])
                SEARCH_QUERY+="$key"
                update_filter
                ;;
        esac
    done
    
    clear_screen
    echo -e "${GREEN}Goodbye!${RESET} 👋"
    echo ""
}

# ─────────────────────────────────────────────────────────────────────────────
# Entry Point
# ─────────────────────────────────────────────────────────────────────────────

main "$@"
