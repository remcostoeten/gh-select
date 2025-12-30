#!/usr/bin/env bash

# Pre-Migration Benchmark for gh-select
# Captures comprehensive metrics to compare after Zig migration
# Run: ./scripts/benchmark_pre_migration.sh

set -e

# Colors
BOLD="\033[1m"
DIM="\033[2m"
GREEN="\033[32m"
CYAN="\033[36m"
YELLOW="\033[33m"
RESET="\033[0m"

# Determine executable
if [ -f "./gh-select" ]; then
    GH_SELECT="./gh-select"
elif [ -f "../gh-select" ]; then
    GH_SELECT="../gh-select"
else
    GH_SELECT="gh-select"
fi

CACHE_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/gh-select"
RESULTS_FILE="./scripts/benchmark_baseline_v1.json"

echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════════${RESET}"
echo -e "${BOLD}gh-select Pre-Migration Benchmark${RESET}"
echo -e "${DIM}Baseline metrics for Zig migration comparison${RESET}"
echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════════${RESET}"
echo ""

# System info
echo -e "${BOLD}System Info${RESET}"
echo -e "${DIM}─────────────────────────────────────────${RESET}"
OS=$(uname -s)
ARCH=$(uname -m)
BASH_VER=${BASH_VERSION}
echo -e "  OS:       $OS $ARCH"
echo -e "  Bash:     $BASH_VER"
echo -e "  Date:     $(date -Iseconds)"
echo ""

# Repository count
echo -e "${BOLD}Repository Stats${RESET}"
echo -e "${DIM}─────────────────────────────────────────${RESET}"
REPO_COUNT=$(gh repo list --json nameWithOwner --limit 5000 2>/dev/null | jq 'length')
PRIVATE_COUNT=$(gh repo list --json isPrivate --limit 5000 2>/dev/null | jq '[.[] | select(.isPrivate == true)] | length')
PUBLIC_COUNT=$((REPO_COUNT - PRIVATE_COUNT))
echo -e "  Total:    ${BOLD}$REPO_COUNT${RESET} repositories"
echo -e "  Private:  $PRIVATE_COUNT"
echo -e "  Public:   $PUBLIC_COUNT"
echo ""

# Benchmark functions
time_command() {
    local start end
    start=$(date +%s.%N)
    eval "$1" >/dev/null 2>&1 || true
    end=$(date +%s.%N)
    echo "$end - $start" | bc -l
}

run_benchmark() {
    local name="$1"
    local cmd="$2"
    local runs="${3:-3}"
    local times=()
    
    for ((i=1; i<=runs; i++)); do
        times+=("$(time_command "$cmd")")
    done
    
    # Calculate average
    local sum=0
    for t in "${times[@]}"; do
        sum=$(echo "$sum + $t" | bc -l)
    done
    local avg=$(echo "scale=3; $sum / $runs" | bc -l)
    
    # Calculate min
    local min=${times[0]}
    for t in "${times[@]}"; do
        if (( $(echo "$t < $min" | bc -l) )); then
            min=$t
        fi
    done
    
    printf "  %-25s avg: ${GREEN}%.3fs${RESET}  min: %.3fs  (${DIM}%d runs${RESET})\n" "$name" "$avg" "$min" "$runs"
    echo "$avg"
}

# Memory usage (RSS in KB)
get_memory() {
    local cmd="$1"
    if command -v /usr/bin/time >/dev/null 2>&1; then
        /usr/bin/time -v bash -c "$cmd" 2>&1 | grep "Maximum resident" | awk '{print $NF}'
    else
        echo "N/A"
    fi
}

echo -e "${BOLD}Performance Tests${RESET}"
echo -e "${DIM}─────────────────────────────────────────${RESET}"

# Test 1: Help (no I/O, pure startup)
HELP_TIME=$(run_benchmark "Help (--help)" "echo q | $GH_SELECT --help")

# Test 2: Version (minimal)
VERSION_TIME=$(run_benchmark "Version (--version)" "$GH_SELECT --version")

# Test 3: Cold start (clear cache, fetch repos)
rm -rf "$CACHE_DIR"
echo -e "  ${DIM}Clearing cache for cold start...${RESET}"
COLD_TIME=$(run_benchmark "Cold start (API fetch)" "$GH_SELECT --refresh-only" 1)

# Test 4: Warm start (cached)
WARM_TIME=$(run_benchmark "Warm start (cached)" "$GH_SELECT --refresh-only")

# Test 5: Cache refresh
REFRESH_TIME=$(run_benchmark "Cache refresh" "$GH_SELECT --no-cache --refresh-only" 1)

echo ""

# Memory test
echo -e "${BOLD}Memory Usage${RESET}"
echo -e "${DIM}─────────────────────────────────────────${RESET}"
if command -v /usr/bin/time >/dev/null 2>&1; then
    MEM_HELP=$(/usr/bin/time -v bash -c "echo q | $GH_SELECT --help" 2>&1 | grep "Maximum resident" | awk '{print $NF}' || echo "N/A")
    MEM_REFRESH=$(/usr/bin/time -v bash -c "$GH_SELECT --refresh-only" 2>&1 | grep "Maximum resident" | awk '{print $NF}' || echo "N/A")
    echo -e "  Help:     ${MEM_HELP} KB"
    echo -e "  Refresh:  ${MEM_REFRESH} KB"
else
    MEM_HELP="N/A"
    MEM_REFRESH="N/A"
    echo -e "  ${YELLOW}Install GNU time for memory metrics: sudo apt install time${RESET}"
fi
echo ""

# Binary size
echo -e "${BOLD}Binary Size${RESET}"
echo -e "${DIM}─────────────────────────────────────────${RESET}"
BINARY_SIZE=$(stat -c%s "$GH_SELECT" 2>/dev/null || stat -f%z "$GH_SELECT" 2>/dev/null || echo "N/A")
BINARY_LINES=$(wc -l < "$GH_SELECT" 2>/dev/null || echo "N/A")
echo -e "  Size:     ${BINARY_SIZE} bytes ($(echo "scale=2; $BINARY_SIZE / 1024" | bc) KB)"
echo -e "  Lines:    ${BINARY_LINES}"
echo ""

# Dependencies
echo -e "${BOLD}Dependencies${RESET}"
echo -e "${DIM}─────────────────────────────────────────${RESET}"
GH_VER=$(gh --version 2>/dev/null | head -1 || echo "N/A")
FZF_VER=$(fzf --version 2>/dev/null | head -1 || echo "N/A")
JQ_VER=$(jq --version 2>/dev/null || echo "N/A")
echo -e "  gh:   $GH_VER"
echo -e "  fzf:  $FZF_VER"
echo -e "  jq:   $JQ_VER"
echo ""

# Save results as JSON
cat > "$RESULTS_FILE" << EOF
{
  "version": "v1.0.4",
  "language": "bash",
  "date": "$(date -Iseconds)",
  "system": {
    "os": "$OS",
    "arch": "$ARCH",
    "bash": "$BASH_VER"
  },
  "repos": {
    "total": $REPO_COUNT,
    "private": $PRIVATE_COUNT,
    "public": $PUBLIC_COUNT
  },
  "timing_seconds": {
    "help": $HELP_TIME,
    "version": $VERSION_TIME,
    "cold_start": $COLD_TIME,
    "warm_start": $WARM_TIME,
    "cache_refresh": $REFRESH_TIME
  },
  "memory_kb": {
    "help": "${MEM_HELP}",
    "refresh": "${MEM_REFRESH}"
  },
  "binary": {
    "size_bytes": $BINARY_SIZE,
    "lines": $BINARY_LINES
  },
  "dependencies": {
    "gh": "$GH_VER",
    "fzf": "$FZF_VER",
    "jq": "$JQ_VER"
  }
}
EOF

echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════════${RESET}"
echo -e "${GREEN}Results saved to:${RESET} $RESULTS_FILE"
echo -e "${DIM}Run this again after Zig migration to compare${RESET}"
echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════════${RESET}"
