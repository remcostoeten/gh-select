#!/usr/bin/env bash

set -e

# ═══════════════════════════════════════════════════════════════════════════════
# gh-select - Interactive GitHub Repository Selector
# ═══════════════════════════════════════════════════════════════════════════════

# Version info
VERSION="1.0.4"
AUTHOR="Remco Stoeten"
AUTHOR_HANDLE="@remcostoeten"
AUTHOR_URL="https://remcostoeten.nl"
REPO="https://github.com/remcostoeten/gh-select"

# ─────────────────────────────────────────────────────────────────────────────
# ANSI Color Codes
# ─────────────────────────────────────────────────────────────────────────────
if [[ -z "${NO_COLOR:-}" ]] && [[ -t 1 ]]; then
    BOLD="\033[1m"
    DIM="\033[2m"
    ITALIC="\033[3m"
    RESET="\033[0m"
    
    # Colors
    RED="\033[31m"
    GREEN="\033[32m"
    YELLOW="\033[33m"
    BLUE="\033[34m"
    MAGENTA="\033[35m"
    CYAN="\033[36m"
    WHITE="\033[37m"
    
    # Bright colors
    BRIGHT_GREEN="\033[92m"
    BRIGHT_CYAN="\033[96m"
    BRIGHT_WHITE="\033[97m"
else
    BOLD="" DIM="" ITALIC="" RESET=""
    RED="" GREEN="" YELLOW="" BLUE="" MAGENTA="" CYAN="" WHITE=""
    BRIGHT_GREEN="" BRIGHT_CYAN="" BRIGHT_WHITE=""
fi

# ─────────────────────────────────────────────────────────────────────────────
# Help & Version Display
# ─────────────────────────────────────────────────────────────────────────────
show_version() {
    echo -e ""
    echo -e "${BOLD}${CYAN}gh-select${RESET} ${DIM}v${VERSION}${RESET}"
    echo -e "${DIM}A GitHub CLI extension following the gh extension spec${RESET}"
    echo -e ""
    echo -e "Fuzzy-find and manage your GitHub repos from the terminal."
    echo -e ""
    echo -e "${DIM}─────────────────────────────────────────${RESET}"
    echo -e "${WHITE}By ${BOLD}${AUTHOR}${RESET}"
    echo -e "${DIM}   ${AUTHOR_HANDLE}${RESET}"
    echo -e "${BLUE}   ${AUTHOR_URL}${RESET}"
    echo -e ""
    echo -e "${DIM}Source:${RESET} ${BLUE}${REPO}${RESET}"
    echo -e "${DIM}─────────────────────────────────────────${RESET}"
    echo -e ""
}

show_help() {
    echo -e ""
    echo -e "${BOLD}${CYAN}gh-select${RESET} ${DIM}─ Interactive GitHub Repository Selector${RESET}"
    echo -e "${DIM}A GitHub CLI extension following the gh extension spec${RESET}"
    echo -e ""
    echo -e "${WHITE}By ${BOLD}${AUTHOR}${RESET}"
    echo -e "${DIM}   ${AUTHOR_HANDLE}${RESET}"
    echo -e "${BLUE}   ${AUTHOR_URL}${RESET}"
    echo -e ""
    echo -e "${DIM}Source:${RESET} ${BLUE}${REPO}${RESET}"
    echo -e ""
    echo -e "${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e ""
    echo -e "${BOLD}${WHITE}USAGE${RESET}"
    echo -e ""
    echo -e "    ${GREEN}gh select${RESET} ${DIM}[OPTIONS]${RESET}"
    echo -e ""
    echo -e "${BOLD}${WHITE}OPTIONS${RESET}"
    echo -e ""
    echo -e "    ${YELLOW}-n${RESET}, ${YELLOW}--no-cache${RESET}   Bypass cache, fetch fresh data"
    echo -e "    ${YELLOW}-r${RESET}, ${YELLOW}--refresh-only${RESET} Refresh cache and exit"
    echo -e "    ${YELLOW}-v${RESET}, ${YELLOW}--version${RESET}    Show version information"
    echo -e "    ${YELLOW}-h${RESET}, ${YELLOW}--help${RESET}       Show this help message"
    echo -e ""
    echo -e "${BOLD}${WHITE}FEATURES${RESET}"
    echo -e ""
    echo -e "    ${BRIGHT_GREEN}*${RESET} Interactive repository selection with fuzzy search"
    echo -e "    ${BRIGHT_GREEN}*${RESET} Clone repositories to any location"
    echo -e "    ${BRIGHT_GREEN}*${RESET} Copy repository names or URLs to clipboard"
    echo -e "    ${BRIGHT_GREEN}*${RESET} Open repositories in browser"
    echo -e "    ${BRIGHT_GREEN}*${RESET} Beautiful interface with live preview"
    echo -e ""
    echo -e "${BOLD}${WHITE}DEPENDENCIES${RESET}"
    echo -e ""
    echo -e "    ${CYAN}gh${RESET}     GitHub CLI           ${DIM}https://cli.github.com/${RESET}"
    echo -e "    ${CYAN}fzf${RESET}    Fuzzy finder         ${DIM}brew install fzf || sudo apt install fzf${RESET}"
    echo -e "    ${CYAN}jq${RESET}     JSON processor       ${DIM}brew install jq || sudo apt install jq${RESET}"
    echo -e ""
    echo -e "${BOLD}${WHITE}EXAMPLES${RESET}"
    echo -e ""
    echo -e "    ${DIM}\$${RESET} ${GREEN}gh select${RESET}              ${DIM}# Launch interactive selector${RESET}"
    echo -e "    ${DIM}\$${RESET} ${GREEN}gh select${RESET} ${YELLOW}--help${RESET}       ${DIM}# Show this help${RESET}"
    echo -e "    ${DIM}\$${RESET} ${GREEN}gh select${RESET} ${YELLOW}--version${RESET}    ${DIM}# Show version${RESET}"
    echo -e ""
    echo -e "${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e ""
    echo -e "${BOLD}${WHITE}QUICK ACTIONS${RESET}"
    echo -e ""
    echo -e "    ${CYAN}g${RESET}  Open source repository in browser"
    echo -e "    ${CYAN}s${RESET}  Star this repository"
    echo -e "    ${CYAN}r${RESET}  Run gh-select"
    echo -e "    ${CYAN}h${RESET}  Health check dependencies"
    echo -e "    ${CYAN}q${RESET}  Quit"
    echo -e ""
    echo -e -n "${DIM}Press a key or q to quit:${RESET} "
    
    # Read single character
    read -r -n 1 key
    echo -e ""
    
    case "$key" in
        g|G)
            echo -e "\n${CYAN}Opening${RESET} source repository..."
            if command -v xdg-open >/dev/null 2>&1; then
                xdg-open "${REPO}" 2>/dev/null &
            elif command -v open >/dev/null 2>&1; then
                open "${REPO}"
            else
                echo -e "${BLUE}${REPO}${RESET}"
            fi
            ;;
        s|S)
            echo -e "\n${CYAN}Starring${RESET} repository..."
            if gh repo set-default remcostoeten/gh-select 2>/dev/null; then
                :
            fi
            gh repo edit remcostoeten/gh-select --add-topic gh-extension 2>/dev/null || true
            if gh api -X PUT /user/starred/remcostoeten/gh-select 2>/dev/null; then
                echo -e "${GREEN}Starred!${RESET} Thanks for the support!"
            else
                echo -e "${YELLOW}Could not star.${RESET} You may need to star manually at ${BLUE}${REPO}${RESET}"
            fi
            ;;
        r|R)
            echo -e ""
            exec "$0"
            ;;
        h|H)
            echo -e "\n${BOLD}${WHITE}Dependency Health Check${RESET}\n"
            
            # Check gh
            if command -v gh >/dev/null 2>&1; then
                gh_version=$(gh --version 2>/dev/null | head -1)
                echo -e "    ${GREEN}[ok]${RESET}  gh     ${DIM}${gh_version}${RESET}"
            else
                echo -e "    ${RED}[x]${RESET}   gh     ${DIM}not installed${RESET}"
            fi
            
            # Check fzf
            if command -v fzf >/dev/null 2>&1; then
                fzf_version=$(fzf --version 2>/dev/null | head -1)
                echo -e "    ${GREEN}[ok]${RESET}  fzf    ${DIM}${fzf_version}${RESET}"
            else
                echo -e "    ${RED}[x]${RESET}   fzf    ${DIM}not installed${RESET}"
            fi
            
            # Check jq
            if command -v jq >/dev/null 2>&1; then
                jq_version=$(jq --version 2>/dev/null)
                echo -e "    ${GREEN}[ok]${RESET}  jq     ${DIM}${jq_version}${RESET}"
            else
                echo -e "    ${RED}[x]${RESET}   jq     ${DIM}not installed${RESET}"
            fi
            
            # Check gh auth
            echo -e ""
            if gh auth status >/dev/null 2>&1; then
                echo -e "    ${GREEN}[ok]${RESET}  GitHub authenticated"
            else
                echo -e "    ${YELLOW}[!]${RESET}  GitHub not authenticated ${DIM}(run: gh auth login)${RESET}"
            fi
            echo -e ""
            ;;
        q|Q|"")
            ;;
        *)
            ;;
    esac
}

# ─────────────────────────────────────────────────────────────────────────────
# Auto-Generated Argument System
# ─────────────────────────────────────────────────────────────────────────────
# Format: "name:short:function:exit_after"
# - name: long form (--name), also used for bare form if exit_after=1
# - short: short form (-x)
# - function: function to call
# - exit_after: 1 = exit after, 0 = continue

ARGS=(
    "version:v:show_version:1"
    "help:h:show_help:1"
    "no-cache:n:set_no_cache:0"
    "refresh-only:r:set_refresh_only:0"
)

# Runtime flags
FORCE_REFRESH=false
REFRESH_ONLY=false

# Flag setters
set_no_cache() {
    FORCE_REFRESH=true
}

set_refresh_only() {
    FORCE_REFRESH=true
    REFRESH_ONLY=true
}

# Spellcheck suggestion with interactive prompt
spellcheck() {
    local input="${1#--}"  # Strip -- prefix if present
    input="${input#-}"      # Strip - prefix if present
    local best_match=""
    local best_score=0
    local best_func=""
    local names=""
    local input_len=${#input}
    
    # Guard against empty input
    if [[ $input_len -eq 0 ]]; then
        echo "Unknown option: $1"
        echo "Use: gh select --help"
        return 1
    fi
    
    for def in "${ARGS[@]}"; do
        IFS=':' read -r name short func should_exit <<< "$def"
        names="$names --$name"
        
        # Count matching characters
        local score=0
        local i
        for ((i=0; i<input_len; i++)); do
            if [[ "$name" == *"${input:$i:1}"* ]]; then
                score=$((score + 1))
            fi
        done
        
        # Calculate match percentage
        local pct=$((score * 100 / input_len))
        if [[ $pct -gt $best_score ]]; then
            best_score=$pct
            best_match="$name"
            best_func="$func"
        fi
    done
    
    if [[ $best_score -gt 60 ]]; then
        echo -n "Did you mean: gh select --$best_match? [Y/n] "
        read -r response
        if [[ -z "$response" ]] || [[ "$response" =~ ^[Yy] ]]; then
            $best_func
            # Check if we should exit after
            for def in "${ARGS[@]}"; do
                IFS=':' read -r name short func should_exit <<< "$def"
                if [[ "$name" == "$best_match" ]] && [[ "$should_exit" == "1" ]]; then
                    exit 0
                fi
            done
        else
            exit 1
        fi
    else
        echo "Unknown option: $1"
        echo "Available:$names"
        return 1
    fi
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        local arg="$1"
        local matched=false
        
        for def in "${ARGS[@]}"; do
            IFS=':' read -r name short func should_exit <<< "$def"
            
            # Check --long form
            if [[ "$arg" == "--$name" ]]; then
                $func
                [[ "$should_exit" == "1" ]] && exit 0
                matched=true
                break
            fi
            
            # Check -short form
            if [[ "$arg" == "-$short" ]]; then
                $func
                [[ "$should_exit" == "1" ]] && exit 0
                matched=true
                break
            fi
            
            # Check bare form (only for exit commands like version/help)
            if [[ "$should_exit" == "1" ]] && [[ "$arg" == "$name" ]]; then
                $func
                exit 0
            fi
        done
        
        # Special case: "repo" is allowed
        if [[ "$arg" == "repo" ]]; then
            matched=true
        fi
        
        if [[ "$matched" == false ]]; then
            spellcheck "$arg"
            exit 1
        fi
        
        shift
    done
}

parse_args "$@"

# Cross-platform clipboard function
copy_to_clipboard() {
    local text="$1"
    
    # macOS
    if command -v pbcopy >/dev/null 2>&1; then
        echo "$text" | pbcopy
        return 0
    fi
    
    # WSL (Windows Subsystem for Linux)
    if command -v clip.exe >/dev/null 2>&1; then
        echo "$text" | clip.exe
        return 0
    fi
    
    # Linux with X11
    if command -v xclip >/dev/null 2>&1 && [ -n "$DISPLAY" ]; then
        echo "$text" | xclip -selection clipboard
        return 0
    fi
    
    # Linux with Wayland (check if Wayland is actually running)
    if command -v wl-copy >/dev/null 2>&1; then
        # Check if we can actually connect to Wayland
        if echo "test" | wl-copy 2>/dev/null; then
            echo "$text" | wl-copy
            return 0
        fi
    fi
    
    # Linux alternative clipboard tools
    if command -v xsel >/dev/null 2>&1 && [ -n "$DISPLAY" ]; then
        echo "$text" | xsel --clipboard --input
        return 0
    fi
    
    # If all else fails
    return 1
}



# Enhanced dependency checking with installation guidance
check_dependencies() {
    local missing_deps=()
    local error_messages=()
    
    # Check GitHub CLI
    if ! command -v gh >/dev/null 2>&1; then
        missing_deps+=("gh")
        error_messages+=("${RED}[x]${RESET} GitHub CLI (gh) is not installed")
        error_messages+=("    ${DIM}Install:${RESET} ${BLUE}https://cli.github.com/${RESET}")
        error_messages+=("    ${DIM}macOS:${RESET}   brew install gh")
        error_messages+=("    ${DIM}Ubuntu:${RESET}  sudo apt install gh")
        error_messages+=("    ${DIM}Fedora:${RESET}  sudo dnf install gh")
        error_messages+=("    ${DIM}Windows:${RESET} winget install GitHub.cli")
        error_messages+=("")
    fi
    
    # Check fzf
    if ! command -v fzf >/dev/null 2>&1; then
        missing_deps+=("fzf")
        error_messages+=("${RED}[x]${RESET} fzf (fuzzy finder) is not installed")
        error_messages+=("    ${DIM}Website:${RESET} ${BLUE}https://github.com/junegunn/fzf${RESET}")
        error_messages+=("    ${DIM}macOS:${RESET}   brew install fzf")
        error_messages+=("    ${DIM}Ubuntu:${RESET}  sudo apt install fzf")
        error_messages+=("    ${DIM}Fedora:${RESET}  sudo dnf install fzf")
        error_messages+=("")
    fi
    
    # Check jq
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
        error_messages+=("${RED}[x]${RESET} jq (JSON processor) is not installed")
        error_messages+=("    ${DIM}Website:${RESET} ${BLUE}https://jqlang.github.io/jq/${RESET}")
        error_messages+=("    ${DIM}macOS:${RESET}   brew install jq")
        error_messages+=("    ${DIM}Ubuntu:${RESET}  sudo apt install jq")
        error_messages+=("    ${DIM}Fedora:${RESET}  sudo dnf install jq")
        error_messages+=("    ${DIM}Windows:${RESET} winget install jqlang.jq")
        error_messages+=("")
    fi
    
    # If any dependencies are missing, show helpful error and exit
    if [ ${#missing_deps[@]} -gt 0 ]; then
        echo -e ""
        echo -e "${BOLD}${RED}Missing Dependencies${RESET}"
        echo -e "${DIM}────────────────────────────────────────${RESET}"
        echo -e ""
        for msg in "${error_messages[@]}"; do
            echo -e "$msg"
        done
        echo -e "${DIM}After installing, run:${RESET} ${GREEN}gh select${RESET}"
        echo -e "${DIM}For more help:${RESET} ${YELLOW}gh select --help${RESET}"
        echo -e ""
        exit 1
    fi
}

# Check all dependencies
check_dependencies

# Check auth
if ! gh auth status >/dev/null 2>&1; then
    echo -e "${RED}Not authenticated.${RESET} Please run: ${GREEN}gh auth login${RESET}"
    exit 1
fi

# ─────────────────────────────────────────────────────────────────────────────
# Cache Configuration
# ─────────────────────────────────────────────────────────────────────────────
CACHE_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/gh-select"
CACHE_FILE="$CACHE_DIR/repos.json"
CACHE_FORMATTED="$CACHE_DIR/repos.formatted"
CACHE_TTL="${GH_SELECT_CACHE_TTL:-1800}"  # 30 minutes default

mkdir -p "$CACHE_DIR"

cache_is_valid() {
    [[ "$FORCE_REFRESH" == "true" ]] && return 1
    [[ ! -f "$CACHE_FILE" ]] && return 1
    [[ ! -f "$CACHE_FORMATTED" ]] && return 1
    
    local cache_age now
    now=$(date +%s)
    if [[ "$(uname)" == "Darwin" ]]; then
        cache_age=$(stat -f %m "$CACHE_FILE" 2>/dev/null || echo 0)
    else
        cache_age=$(stat -c %Y "$CACHE_FILE" 2>/dev/null || echo 0)
    fi
    
    (( (now - cache_age) < CACHE_TTL ))
}

fetch_repos() {
    printf "${CYAN}Fetching repositories...${RESET}" >&2
    
    local repos
    if ! repos=$(gh repo list --json nameWithOwner,description,isPrivate --limit 1000 2>/dev/null); then
        printf "\r${RED}Error fetching repositories${RESET}\n" >&2
        return 1
    fi
    
    local count
    count=$(echo "$repos" | jq length 2>/dev/null || echo "0")
    
    printf "\r\033[K" >&2
    echo -e "${GREEN}Fetched $count repositories${RESET}" >&2
    echo "$repos"
}

format_repos() {
    jq -r '.[] | (if .isPrivate then "[private]" else "[public]" end) as $privacy | (if .description and .description != "" and .description != null then .description else "No description" end) as $desc | "\(.nameWithOwner)|\($privacy)|\($desc)"' | sort
}

refresh_cache() {
    local json_data
    json_data=$(fetch_repos) || return 1
    
    echo "$json_data" > "$CACHE_FILE"
    echo "$json_data" | format_repos > "$CACHE_FORMATTED"
}

# ─────────────────────────────────────────────────────────────────────────────
# Load Repositories (with caching)
# ─────────────────────────────────────────────────────────────────────────────

if cache_is_valid; then
    cache_age_sec=$(($(date +%s) - $(stat -c %Y "$CACHE_FILE" 2>/dev/null || echo 0)))
    echo -e "${DIM}Using cache (${cache_age_sec}s old)${RESET}"
    formatted_repos="$CACHE_FORMATTED"
else
    echo -e "${CYAN}Loading repositories...${RESET}"
    if refresh_cache; then
        repo_count=$(jq length "$CACHE_FILE" 2>/dev/null || echo "0")
        echo -e "${GREEN}Loaded $repo_count repositories${RESET}"
        formatted_repos="$CACHE_FORMATTED"
    else
        echo -e "${RED}Failed to fetch repositories${RESET}"
        exit 1
    fi
fi

# Exit if only refreshing was requested
if [[ "$REFRESH_ONLY" == "true" ]]; then
    exit 0
fi

# Validate we have data
if [[ ! -s "$formatted_repos" ]]; then
    echo -e "${YELLOW}No repositories found${RESET}"
    exit 1
fi

# fzf selection
selected_line=$(cat "$formatted_repos" | fzf \
    --height=80% \
    --layout=reverse \
    --border=rounded \
    --prompt="Select repository > " \
    --header="Navigate: ↑/↓  |  Filter: type  |  Select: Enter  |  Quit: Esc" \
    --delimiter=" | " \
    --with-nth=1,2,3 \
    --color=fg:#c0caf5,bg:#1a1b26,hl:#bb9af7 \
    --color=fg+:#c0caf5,bg+:#292e42,hl+:#7dcfff \
    --color=info:#7aa2f7,prompt:#7dcfff,pointer:#ff007c \
    --color=marker:#9ece6a,spinner:#9ece6a,header:#9ece6a \
    --preview='echo {} | cut -d"|" -f1 | xargs gh repo view 2>/dev/null || echo "Loading..."' \
    --preview-window=right:60%:wrap) || {
    echo -e "${DIM}No repository selected${RESET}"
    exit 0
}

# Extract repo name
selected_repo=$(echo "$selected_line" | cut -d'|' -f1 | xargs)

if [ -n "$selected_repo" ]; then
    echo -e ""
    echo -e "${GREEN}Selected:${RESET} ${BOLD}${selected_repo}${RESET}"
    echo -e ""
    echo -e "${BOLD}${WHITE}Actions${RESET}"
    echo -e "${DIM}─────────────────────────────────────────${RESET}"
    echo -e "  ${CYAN}1${RESET}  Clone repository"
    echo -e "  ${CYAN}2${RESET}  Copy repository name"  
    echo -e "  ${CYAN}3${RESET}  Copy repository URL"
    echo -e "  ${CYAN}4${RESET}  Open in browser"
    echo -e "  ${CYAN}5${RESET}  Show name and exit"
    echo -e ""
    
    # Use regular read to avoid arrow key escape sequences
    while true; do
        echo -e -n "${DIM}Choice${RESET} ${YELLOW}[1-5]${RESET}: "
        read choice
        case $choice in
            [1-5]) break ;;
            *) echo -e "${RED}Please enter a number between 1 and 5.${RESET}" ;;
        esac
    done
    
    case $choice in
        1)
            echo -e "\n${CYAN}Cloning${RESET} ${BOLD}${selected_repo}${RESET}...\n"
            gh repo clone "$selected_repo"
            ;;
        2)
            if copy_to_clipboard "$selected_repo"; then
                echo -e "${GREEN}Copied${RESET} repository name to clipboard"
            else
                echo -e "Repository name: ${BOLD}$selected_repo${RESET}"
                echo -e "${DIM}(Clipboard not available)${RESET}"
            fi
            ;;
        3)
            repo_url="https://github.com/$selected_repo"
            if copy_to_clipboard "$repo_url"; then
                echo -e "${GREEN}Copied${RESET} repository URL to clipboard"
            else
                echo -e "Repository URL: ${BLUE}$repo_url${RESET}"
                echo -e "${DIM}(Clipboard not available)${RESET}"
            fi
            ;;
        4)
            echo -e "\n${CYAN}Opening${RESET} in browser...\n"
            gh repo view "$selected_repo" --web
            ;;
        5|*)
            echo -e "${BOLD}$selected_repo${RESET}"
            ;;
    esac
else
    echo -e "${DIM}No repository selected${RESET}"
fi
