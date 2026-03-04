#!/bin/bash
# Semarang YouTube Shorts Production Pipeline
# Usage: ./run.sh [NUMBER] | ./run.sh [-h|--help]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Print help
print_help() {
    echo -e "${BLUE}Semarang - YouTube Shorts Production Pipeline${NC}"
    echo ""
    echo "Usage:"
    echo "  ./run.sh [NUMBER]     Generate video for specific file number"
    echo "  ./run.sh              Generate video for random unprocessed file"
    echo "  ./run.sh -h, --help  Show this help message"
    echo ""
    echo "Examples:"
    echo "  ./run.sh 43          Generate video for Go_Secret__line-043.md"
    echo "  ./run.sh             Generate video for random unprocessed file"
    echo ""
    echo "Files are selected from raw/ directory by pattern: *-line-<NUM>.md"
    echo ""
    echo "Environment:"
    echo "  Make sure .env file exists with ZAI_API_KEY and YANDEX_API_KEY"
}

# Print error
print_error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
}

# Print info
print_info() {
    echo -e "${BLUE}INFO: $1${NC}"
}

# Print success
print_success() {
    echo -e "${GREEN}SUCCESS: $1${NC}"
}

# Print warning
print_warning() {
    echo -e "${YELLOW}WARNING: $1${NC}"
}

# Check if .env exists
check_env() {
    if [ ! -f "$PROJECT_DIR/.env" ]; then
        print_error ".env file not found"
        echo ""
        echo "Please create .env from .env.example:"
        echo "  cp .env.example .env"
        echo "  # Edit .env with your API keys"
        exit 1
    fi
}

# Check if Go is available (for local run)
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        echo "  Please install Go 1.21+ or run via Docker"
        exit 1
    fi
}

# Check if FFmpeg is available
check_ffmpeg() {
    if ! command -v ffmpeg &> /dev/null; then
        print_warning "FFmpeg is not installed or not in PATH"
        echo "  Some features may not work properly"
    fi
}

# Parse arguments
FILE_NUMBER=""

case "$1" in
    -h|--help)
        print_help
        exit 0
        ;;
    "")
        # No argument - use random unprocessed file
        FILE_NUMBER=""
        ;;
    *)
        # Check if argument is a number
        if ! [[ "$1" =~ ^[0-9]+$ ]]; then
            print_error "Invalid argument: '$1'"
            echo ""
            print_help
            exit 1
        fi
        FILE_NUMBER="$1"
        ;;
esac

# Change to project directory
cd "$PROJECT_DIR"

# Check prerequisites
check_env

# Check if we're running in Docker or locally
if [ -f /.dockerenv ]; then
    # Running in Docker
    print_info "Running in Docker environment"
else
    # Running locally
    check_go
    check_ffmpeg
    print_info "Running locally"
fi

# Build the Go application
print_info "Building semarang..."
cd "$PROJECT_DIR"
go build -o semarang ./cmd/main.go

if [ $? -ne 0 ]; then
    print_error "Failed to build semarang"
    exit 1
fi

print_success "Build completed"

# Run the application
print_info "Starting video generation..."

if [ -z "$FILE_NUMBER" ]; then
    ./semarang
else
    ./semarang --number="$FILE_NUMBER"
fi

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    print_success "Video generation completed!"
else
    print_error "Video generation failed with exit code: $EXIT_CODE"
    exit $EXIT_CODE
fi
