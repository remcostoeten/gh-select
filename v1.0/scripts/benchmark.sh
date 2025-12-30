#!/usr/bin/env bash

# Benchmark script for gh-select performance
set -e

# Spinner function
spin() {
    local pid=$1
    local delay=0.1
    local spinstr='|/-\'
    while ps -p "$pid" > /dev/null 2>&1; do
        local temp=${spinstr#?}
        printf " [%c]  " "$spinstr"
        local spinstr=$temp${spinstr%"$temp"}
        sleep $delay
        printf "\b\b\b\b\b\b"
    done
    printf "    \b\b\b\b"
}

run_with_spinner() {
    local cmd="$1"
    eval "$cmd" &
    local pid=$!
    spin $pid
    wait $pid
    return $?
}

echo "gh-select Performance Benchmark"
echo "=================================="

# Determine the executable path
# If running from scripts dir, look in ../
# If running from root, look in ./
if [ -f "./gh-select" ]; then
    GH_SELECT="./gh-select"
elif [ -f "../gh-select" ]; then
    GH_SELECT="../gh-select"
else
    # Fallback to simple command if installed globally or in path, otherwise fail
    GH_SELECT="gh-select"
fi

# Warm up
echo "Warming up..."
run_with_spinner "gh auth status >/dev/null 2>&1"

# Test 1: Cold start (clear cache)
echo ""
echo "Test 1: Cold start (clear cache)"
rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/gh-select"
start_time=$(date +%s.%N)
run_with_spinner "echo q | $GH_SELECT --help >/dev/null 2>&1"
end_time=$(date +%s.%N)
cold_time=$(echo "$end_time - $start_time" | bc -l)
printf "Cold start time: %.3f seconds\n" "$cold_time"

# Test 2: Warm start (with cache)
echo ""
echo "Test 2: Warm start (with cache)"
start_time=$(date +%s.%N)
# Use 'timeout' inside the command string so it's executed properly
run_with_spinner "echo ch | timeout 10s $GH_SELECT --help >/dev/null 2>&1 || true"
end_time=$(date +%s.%N)
warm_time=$(echo "$end_time - $start_time" | bc -l)
printf "Warm start time: %.3f seconds\n" "$warm_time"

# Test 3: Cache refresh time
echo ""
echo "Test 3: Cache refresh time"
start_time=$(date +%s.%N)
run_with_spinner "$GH_SELECT --refresh-only >/dev/null 2>&1"
end_time=$(date +%s.%N)
refresh_time=$(echo "$end_time - $start_time" | bc -l)
printf "Cache refresh time: %.3f seconds\n" "$refresh_time"

# Summary
echo ""
echo "Performance Summary"
echo "======================"
printf "Cold start:   %.3f seconds\n" "$cold_time"
printf "Warm start:   %.3f seconds\n" "$warm_time" 
printf "Cache refresh: %.3f seconds\n" "$refresh_time"

if (( $(echo "$warm_time < 1.0" | bc -l) )); then
    echo "Performance is good (< 1 second warm start)"
elif (( $(echo "$warm_time < 2.0" | bc -l) )); then
    echo "Performance is acceptable (< 2 seconds warm start)"
else
    echo "Performance needs improvement (> 2 seconds warm start)"
fi

echo ""
echo "Tips for better performance:"
echo "   - Use cache (default TTL: 30 minutes)"
echo "   - Run 'gh select --refresh' only when needed"
echo "   - Ensure fast internet connection for GitHub API"
