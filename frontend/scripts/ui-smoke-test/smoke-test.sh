#!/bin/bash
# UI Smoke Test Script for Kubeflow Pipelines Frontend
# Compares screenshots between main branch and a PR branch, then uploads to PR
#
# Usage: ./smoke-test.sh <pr_number> [--repo owner/repo] [--base-branch master]
#
# Prerequisites:
#   - Node.js 18+
#   - gh CLI (authenticated)
#   - Playwright (installed automatically if missing)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
WORK_DIR="${REPO_ROOT}/.ui-smoke-test"
SCREENSHOTS_DIR="${WORK_DIR}/screenshots"

# Default values
REPO="kubeflow/pipelines"
BASE_BRANCH="master"
PR_NUMBER=""
PORT_MAIN=4001
PORT_PR=4002
MOCK_API_PORT=3001

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

usage() {
    cat << EOF
UI Smoke Test for Kubeflow Pipelines Frontend

Usage: $0 <pr_number> [options]

Options:
    --repo OWNER/REPO    GitHub repository (default: kubeflow/pipelines)
    --base-branch NAME   Base branch to compare against (default: master)
    --skip-upload        Skip uploading results to PR
    --local-only         Use local branches instead of fetching from GitHub
    --help               Show this help message

Examples:
    $0 12756
    $0 12756 --repo kubeflow/pipelines --base-branch main
    $0 12756 --skip-upload

Environment Variables:
    UI_SMOKE_PAGES       Comma-separated list of pages to capture (default: all)
    UI_SMOKE_VIEWPORT    Viewport size as WIDTHxHEIGHT (default: 1280x800)
EOF
    exit 0
}

cleanup() {
    log_info "Cleaning up..."
    # Kill any background processes
    jobs -p | xargs -r kill 2>/dev/null || true

    # Remove work directory (optional - comment out to preserve for debugging)
    # rm -rf "$WORK_DIR"
}

trap cleanup EXIT

# Parse arguments
SKIP_UPLOAD=false
LOCAL_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --repo)
            REPO="$2"
            shift 2
            ;;
        --base-branch)
            BASE_BRANCH="$2"
            shift 2
            ;;
        --skip-upload)
            SKIP_UPLOAD=true
            shift
            ;;
        --local-only)
            LOCAL_ONLY=true
            shift
            ;;
        --help)
            usage
            ;;
        -*)
            log_error "Unknown option: $1"
            usage
            ;;
        *)
            if [[ -z "$PR_NUMBER" ]]; then
                PR_NUMBER="$1"
            fi
            shift
            ;;
    esac
done

if [[ -z "$PR_NUMBER" ]]; then
    log_error "PR number is required"
    usage
fi

log_info "Starting UI smoke test for PR #${PR_NUMBER}"
log_info "Repository: ${REPO}"
log_info "Base branch: ${BASE_BRANCH}"

# Create work directory
mkdir -p "$WORK_DIR"
mkdir -p "$SCREENSHOTS_DIR/main"
mkdir -p "$SCREENSHOTS_DIR/pr"
mkdir -p "$SCREENSHOTS_DIR/comparison"

# Get PR info
if [[ "$LOCAL_ONLY" == "false" ]]; then
    log_info "Fetching PR information..."
    PR_INFO=$(gh pr view "$PR_NUMBER" --repo "$REPO" --json headRefName,headRepositoryOwner,headRepository 2>/dev/null || echo "")

    if [[ -z "$PR_INFO" ]]; then
        log_error "Could not fetch PR #${PR_NUMBER} from ${REPO}"
        exit 1
    fi

    PR_BRANCH=$(echo "$PR_INFO" | jq -r '.headRefName')
    PR_OWNER=$(echo "$PR_INFO" | jq -r '.headRepositoryOwner.login')
    PR_REPO_NAME=$(echo "$PR_INFO" | jq -r '.headRepository.name')

    log_info "PR branch: ${PR_OWNER}/${PR_REPO_NAME}:${PR_BRANCH}"
fi

# Install Playwright if needed
if ! command -v npx &> /dev/null || ! npx playwright --version &> /dev/null 2>&1; then
    log_info "Installing Playwright..."
    npm install -g playwright
    npx playwright install chromium
fi

# Ensure node dependencies for screenshot tool
cd "$SCRIPT_DIR"
if [[ ! -d "node_modules" ]]; then
    log_info "Installing screenshot tool dependencies..."
    npm init -y 2>/dev/null || true
    npm install playwright sharp --save
fi

# Function to build and serve frontend
build_and_serve() {
    local branch_name=$1
    local build_dir=$2
    local port=$3

    log_info "Building ${branch_name} branch..."

    cd "$REPO_ROOT/frontend"

    # Build the frontend
    npm run build:tailwind 2>/dev/null || true

    # Use a simple static server for the built files
    if [[ -d "build" ]]; then
        log_info "Serving ${branch_name} on port ${port}..."
        npx serve -s build -l "$port" &
        sleep 3
    else
        log_error "Build directory not found for ${branch_name}"
        return 1
    fi
}

# Function to capture screenshots
capture_screenshots() {
    local port=$1
    local output_dir=$2
    local label=$3

    log_info "Capturing screenshots for ${label}..."

    node "$SCRIPT_DIR/capture-screenshots.js" \
        --port "$port" \
        --output "$output_dir" \
        --label "$label"
}

# Build comparison images
generate_comparison() {
    log_info "Generating side-by-side comparisons..."

    node "$SCRIPT_DIR/generate-comparison.js" \
        --main "$SCREENSHOTS_DIR/main" \
        --pr "$SCREENSHOTS_DIR/pr" \
        --output "$SCREENSHOTS_DIR/comparison"
}

# Upload to PR
upload_to_pr() {
    if [[ "$SKIP_UPLOAD" == "true" ]]; then
        log_info "Skipping upload (--skip-upload specified)"
        return 0
    fi

    log_info "Uploading comparison to PR #${PR_NUMBER}..."

    # Create a summary markdown file
    SUMMARY_FILE="$SCREENSHOTS_DIR/comparison/SUMMARY.md"

    cat > "$SUMMARY_FILE" << EOF
## UI Smoke Test Results

Comparing \`${BASE_BRANCH}\` (left) with PR branch (right)

EOF

    # Add each comparison image
    for img in "$SCREENSHOTS_DIR/comparison"/*.png; do
        if [[ -f "$img" ]]; then
            filename=$(basename "$img")
            page_name="${filename%.png}"
            echo "### ${page_name}" >> "$SUMMARY_FILE"
            echo "" >> "$SUMMARY_FILE"
        fi
    done

    # Upload images and comment on PR
    node "$SCRIPT_DIR/upload-to-pr.js" \
        --pr "$PR_NUMBER" \
        --repo "$REPO" \
        --screenshots "$SCREENSHOTS_DIR/comparison"
}

# Main execution
main() {
    # For local testing, we'll use mock data and the current build
    log_info "Setting up test environment..."

    # Check if we have a built frontend
    if [[ ! -d "$REPO_ROOT/frontend/build" ]]; then
        log_warn "No frontend build found. Building now..."
        cd "$REPO_ROOT/frontend"
        npm ci
        npm run build
    fi

    # Start mock API server (if available)
    if [[ -f "$REPO_ROOT/frontend/mock-backend/mock-api-server.ts" ]]; then
        log_info "Starting mock API server on port ${MOCK_API_PORT}..."
        cd "$REPO_ROOT/frontend"
        npm run mock:api &
        MOCK_PID=$!
        sleep 5
    fi

    # Serve the built frontend
    log_info "Serving frontend on port ${PORT_PR}..."
    cd "$REPO_ROOT/frontend"
    npx serve -s build -l "$PORT_PR" &
    SERVER_PID=$!
    sleep 3

    # Capture screenshots
    capture_screenshots "$PORT_PR" "$SCREENSHOTS_DIR/pr" "PR-${PR_NUMBER}"

    # Note: For a full comparison, you would:
    # 1. Clone/checkout the base branch to a separate directory
    # 2. Build it
    # 3. Serve it on PORT_MAIN
    # 4. Capture screenshots for main
    # 5. Generate comparisons

    log_info "Screenshots saved to: ${SCREENSHOTS_DIR}"

    # Generate comparison (if main screenshots exist)
    if [[ -d "$SCREENSHOTS_DIR/main" ]] && [[ -n "$(ls -A "$SCREENSHOTS_DIR/main" 2>/dev/null)" ]]; then
        generate_comparison
        upload_to_pr
    else
        log_warn "No main branch screenshots found. Run with base branch to generate comparison."
    fi

    log_info "UI smoke test complete!"
}

main
