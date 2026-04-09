#!/usr/bin/env bash
#
# act.sh - GitHub Actions local testing wrapper
# Usage: ./scripts/act.sh [command] [args]
#
# Commands:
#   list              List all available workflows
#   list-detailed     Show detailed workflow information
#   test-ci           Test CI workflow (pull_request event)
#   test-release      Test release workflow (push event)
#   test-all          Test all workflows
#   create-secrets    Generate .actrc secrets template

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ACTRC_FILE="${PROJECT_ROOT}/.actrc"
ACT_IMAGE="ghcr.io/catthehacker/ubuntu:full-latest"

# ============================================
# Utility Functions
# ============================================

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

check_dependencies() {
    if ! command -v act &> /dev/null; then
        log_error "act not found. Install with: brew install act"
        exit 1
    fi

    if ! command -v docker &> /dev/null; then
        log_error "Docker not found. Install Docker Desktop"
        exit 1
    fi
}

check_docker_running() {
    if ! docker ps &> /dev/null; then
        log_error "Docker is not running. Start Docker Desktop"
        exit 1
    fi
}

# ============================================
# Act Commands
# ============================================

list_workflows() {
    log_info "Available workflows:"
    act --list
}

list_workflows_detailed() {
    log_info "Detailed workflow information:"
    echo ""
    
    # Parse .github/workflows/*.yml and show details
    for workflow in .github/workflows/*.yml; do
        if [ -f "$workflow" ]; then
            echo -e "${BLUE}📋 $(basename "$workflow")${NC}"
            cat "$workflow" | grep -E '^\s*(name|on:|jobs:)' | head -5
            echo ""
        fi
    done
}

test_ci() {
    log_info "Testing CI workflow (pull_request event)..."
    
    check_docker_running
    
    if [ ! -f "$ACTRC_FILE" ]; then
        log_warn ".actrc not found. Creating with defaults..."
        create_secrets
    fi
    
    log_info "Running CI tests with act..."
    act pull_request \
        -P ubuntu-latest="$ACT_IMAGE" \
        --secret-file "$ACTRC_FILE" \
        --rm \
        || log_error "CI workflow test failed"
}

test_release() {
    log_info "Testing release workflow (push event)..."
    
    check_docker_running
    
    if [ ! -f "$ACTRC_FILE" ]; then
        log_warn ".actrc not found. Creating with defaults..."
        create_secrets
    fi
    
    log_info "Running release workflow with act..."
    act push \
        -j goreleaser \
        -P ubuntu-latest="$ACT_IMAGE" \
        --container-architecture linux/amd64 \
        --env RUNNER_TOOL_CACHE=/tmp/toolcache \
        --env AGENT_TOOLSDIRECTORY=/tmp/toolcache \
        --secret-file "$ACTRC_FILE" \
        --rm \
        || log_error "Release workflow test failed"
}

test_all() {
    log_info "Testing ALL workflows..."
    
    check_docker_running
    
    if [ ! -f "$ACTRC_FILE" ]; then
        log_warn ".actrc not found. Creating with defaults..."
        create_secrets
    fi
    
    log_info "Running all workflows with act..."
    act \
        -P ubuntu-latest="$ACT_IMAGE" \
        --secret-file "$ACTRC_FILE" \
        --rm \
        || log_warn "Some workflows may have failed"
}

create_secrets() {
    log_info "Creating .actrc secrets template..."
    
    if [ -f "$ACTRC_FILE" ]; then
        log_warn "$ACTRC_FILE already exists. Backing up to .actrc.bak"
        cp "$ACTRC_FILE" "${ACTRC_FILE}.bak"
    fi
    
    cat > "$ACTRC_FILE" << 'EOF'
# GitHub Actions Secrets for local testing with act
# Fill in your actual secrets below (keep in .gitignore)

# GitHub API token (for releases, GitHub API calls)
# Get from: https://github.com/settings/tokens
# Scopes: repo, workflow
GITHUB_TOKEN=ghp_YOUR_TOKEN_HERE

# GoReleaser configuration
GORELEASER_CURRENT_TAG=v0.0.0-local
GORELEASER_PRE_RELEASE=true

# AWS Credentials (for Mirror-Cloud testing)
# Warning: Only store test/sandbox credentials here!
AWS_ACCESS_KEY_ID=test-key-id
AWS_SECRET_ACCESS_KEY=test-secret-key
AWS_REGION=eu-central-1

# Docker credentials (for pushing images)
# Get from: https://hub.docker.com/settings/security
DOCKER_USERNAME=your-docker-username
DOCKER_PASSWORD=your-docker-password

# Optional: Homebrew tap credentials (if publishing formulas)
HOMEBREW_GITHUB_TOKEN=ghp_YOUR_HOMEBREW_TOKEN

# Optional: GoFish credentials
GOFISH_GITHUB_TOKEN=ghp_YOUR_GOFISH_TOKEN

# Custom secrets for specific workflows
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
EOF
    
    log_success "Created $ACTRC_FILE"
    log_warn "⚠️  IMPORTANT: Edit .actrc and fill in your actual secrets!"
    log_warn "⚠️  Never commit .actrc to git (it's in .gitignore)"
    echo ""
    log_info "To fill in secrets:"
    echo "  1. GitHub Token: https://github.com/settings/tokens"
    echo "  2. AWS Credentials: Use test/sandbox account only"
    echo "  3. Docker Credentials: https://hub.docker.com/settings/security"
    echo ""
    log_info "After editing, test with: task act:ci"
}

# ============================================
# Main
# ============================================

main() {
    local command="${1:-list}"
    
    check_dependencies
    
    case "$command" in
        list)
            list_workflows
            ;;
        list-detailed)
            list_workflows_detailed
            ;;
        test-ci)
            test_ci
            ;;
        test-release)
            test_release
            ;;
        test-all)
            test_all
            ;;
        create-secrets)
            create_secrets
            ;;
        *)
            log_error "Unknown command: $command"
            echo ""
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  list              List all available workflows"
            echo "  list-detailed     Show detailed workflow info"
            echo "  test-ci           Test CI workflow"
            echo "  test-release      Test release workflow"
            echo "  test-all          Test all workflows"
            echo "  create-secrets    Generate .actrc template"
            echo ""
            exit 1
            ;;
    esac
}

main "$@"
