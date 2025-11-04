#!/bin/bash
#
# Coverage Trend Tracker
# Records coverage percentage over time for trend analysis
#
# Usage:
#   ./scripts/coverage-trend.sh              # Record current coverage
#   ./scripts/coverage-trend.sh --show       # Show trend history
#   ./scripts/coverage-trend.sh --graph      # Show ASCII graph (requires bc)
#

set -e

TREND_FILE=".coverage-trend.txt"
COVERAGE_FILE="coverage.out"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get current coverage
get_coverage() {
    if [ ! -f "$COVERAGE_FILE" ]; then
        echo "Generating coverage report..." >&2
        go test -coverprofile="$COVERAGE_FILE" -covermode=atomic ./... > /dev/null 2>&1 || true
    fi

    if [ ! -f "$COVERAGE_FILE" ]; then
        echo "❌ Failed to generate coverage report" >&2
        exit 1
    fi

    COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}' | sed 's/%//')

    # Validate coverage is a valid number
    if [ -z "$COVERAGE" ]; then
        echo "❌ Failed to parse coverage from report (empty result)" >&2
        exit 1
    fi

    # Check if coverage is a valid number (integer or decimal)
    if ! [[ "$COVERAGE" =~ ^[0-9]+\.?[0-9]*$ ]]; then
        echo "❌ Invalid coverage value: '$COVERAGE' (expected number)" >&2
        exit 1
    fi

    echo "$COVERAGE"
}

# Record coverage with timestamp
record_coverage() {
    local coverage=$1
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    local branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    local commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

    echo "$timestamp | $branch | $commit | $coverage%" >> "$TREND_FILE"
    echo -e "${GREEN}✅ Recorded: $coverage% (branch: $branch, commit: $commit)${NC}"
}

# Show trend history
show_trend() {
    if [ ! -f "$TREND_FILE" ]; then
        echo "No coverage history found."
        echo "Run './scripts/coverage-trend.sh' to start recording."
        exit 0
    fi

    echo ""
    echo "╔════════════════════════════════════════════════════════════════════╗"
    echo "║               COVERAGE TREND HISTORY                               ║"
    echo "╚════════════════════════════════════════════════════════════════════╝"
    echo ""
    printf "%-20s %-25s %-10s %s\n" "DATE" "BRANCH" "COMMIT" "COVERAGE"
    echo "────────────────────────────────────────────────────────────────────"

    while IFS='|' read -r timestamp branch commit coverage; do
        # Trim whitespace
        timestamp=$(echo "$timestamp" | xargs)
        branch=$(echo "$branch" | xargs)
        commit=$(echo "$commit" | xargs)
        coverage=$(echo "$coverage" | xargs)

        # Color code based on coverage
        coverage_num=$(echo "$coverage" | sed 's/%//')
        if (( $(echo "$coverage_num >= 85" | bc -l) )); then
            color=$GREEN
        elif (( $(echo "$coverage_num >= 80" | bc -l) )); then
            color=$YELLOW
        else
            color=$RED
        fi

        printf "%-20s %-25s %-10s ${color}%s${NC}\n" "$timestamp" "$branch" "$commit" "$coverage"
    done < "$TREND_FILE"

    echo ""

    # Show summary stats
    local first_coverage=$(head -1 "$TREND_FILE" | awk -F'|' '{print $4}' | xargs | sed 's/%//')
    local last_coverage=$(tail -1 "$TREND_FILE" | awk -F'|' '{print $4}' | xargs | sed 's/%//')
    local diff=$(echo "$last_coverage - $first_coverage" | bc -l)

    echo "Summary:"
    printf "  First recorded: %.1f%%\n" "$first_coverage"
    printf "  Latest:         %.1f%%\n" "$last_coverage"
    if (( $(echo "$diff > 0" | bc -l) )); then
        printf "  Change:         ${GREEN}+%.1f%%${NC} 📈\n" "$diff"
    elif (( $(echo "$diff < 0" | bc -l) )); then
        printf "  Change:         ${RED}%.1f%%${NC} 📉\n" "$diff"
    else
        printf "  Change:         %.1f%% ➡️\n" "$diff"
    fi
    echo ""
}

# Show ASCII graph
show_graph() {
    if [ ! -f "$TREND_FILE" ]; then
        echo "No coverage history found."
        exit 0
    fi

    if ! command -v bc &> /dev/null; then
        echo "❌ 'bc' command required for graph. Install with: apt-get install bc"
        exit 1
    fi

    echo ""
    echo "Coverage Trend (last 10 recordings):"
    echo ""

    # Get last 10 entries
    tail -10 "$TREND_FILE" | while IFS='|' read -r timestamp branch commit coverage; do
        coverage_num=$(echo "$coverage" | xargs | sed 's/%//')
        timestamp=$(echo "$timestamp" | xargs | cut -d' ' -f1)  # Date only

        # Create bar (scale: 1 char = 1%)
        bar_length=$(printf "%.0f" "$coverage_num")
        bar=$(printf "%-${bar_length}s" "" | tr ' ' '█')

        # Color code
        if (( $(echo "$coverage_num >= 85" | bc -l) )); then
            color=$GREEN
        elif (( $(echo "$coverage_num >= 80" | bc -l) )); then
            color=$YELLOW
        else
            color=$RED
        fi

        printf "%s │ ${color}%s${NC} %.1f%%\n" "$timestamp" "$bar" "$coverage_num"
    done

    echo ""
    echo "Scale: 0%──────────────────50%──────────────────100%"
    echo ""
}

# Main
case "${1:-}" in
    --show)
        show_trend
        ;;
    --graph)
        show_graph
        ;;
    --help)
        echo "Coverage Trend Tracker"
        echo ""
        echo "Usage:"
        echo "  $0              Record current coverage"
        echo "  $0 --show       Show trend history"
        echo "  $0 --graph      Show ASCII graph"
        echo "  $0 --help       Show this help"
        echo ""
        ;;
    *)
        coverage=$(get_coverage)
        record_coverage "$coverage"
        echo ""
        echo "Run '$0 --show' to see trend history"
        echo "Run '$0 --graph' to see trend graph"
        ;;
esac
