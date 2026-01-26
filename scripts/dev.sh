#!/bin/bash
#
# Project Alfred - Local Development Script
# Usage: ./scripts/dev.sh [command]
#
# Commands:
#   start     - Start both backend and mobile app (default)
#   backend   - Start only the Go backend
#   mobile    - Start only the mobile app
#   setup     - Install dependencies and setup environment
#   check     - Run health checks
#   reset     - Reset all data (database, WhatsApp session, tokens)
#   help      - Show this help message
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Change to project root
cd "$PROJECT_ROOT"

# Helper functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}! $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

check_dependencies() {
    print_header "Checking Dependencies"

    # Check Go
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}')
        print_success "Go installed: $GO_VERSION"
    else
        print_error "Go is not installed. Install with: brew install go"
        exit 1
    fi

    # Check Node.js
    if command -v node &> /dev/null; then
        NODE_VERSION=$(node --version)
        print_success "Node.js installed: $NODE_VERSION"
    else
        print_error "Node.js is not installed. Install with: brew install node"
        exit 1
    fi

    # Check npm
    if command -v npm &> /dev/null; then
        NPM_VERSION=$(npm --version)
        print_success "npm installed: $NPM_VERSION"
    else
        print_error "npm is not installed"
        exit 1
    fi
}

check_env() {
    print_header "Checking Environment"

    # Check for .env file
    if [ -f "$PROJECT_ROOT/.env" ]; then
        print_success ".env file found"
        source "$PROJECT_ROOT/.env"
    else
        print_warning ".env file not found"
    fi

    # Check ANTHROPIC_API_KEY
    if [ -n "$ANTHROPIC_API_KEY" ]; then
        print_success "ANTHROPIC_API_KEY is set"
    else
        print_warning "ANTHROPIC_API_KEY is not set - event detection will be disabled"
    fi

    # Check Google credentials
    if [ -f "$PROJECT_ROOT/credentials.json" ]; then
        print_success "credentials.json found"
    elif [ -n "$GOOGLE_CREDENTIALS_JSON" ]; then
        print_success "GOOGLE_CREDENTIALS_JSON is set"
    else
        print_warning "Google credentials not found - Calendar integration will be disabled"
    fi

    # Check mobile .env.local
    if [ -f "$PROJECT_ROOT/mobile/.env.local" ]; then
        print_success "mobile/.env.local found"
    else
        print_warning "mobile/.env.local not found - creating with localhost URL"
        echo "EXPO_PUBLIC_API_BASE_URL=http://localhost:8080" > "$PROJECT_ROOT/mobile/.env.local"
        print_success "Created mobile/.env.local"
    fi
}

setup() {
    print_header "Setting Up Project Alfred"

    check_dependencies

    # Install Go dependencies
    echo -e "\n${BLUE}Installing Go dependencies...${NC}"
    go mod download
    print_success "Go dependencies installed"

    # Install mobile dependencies
    echo -e "\n${BLUE}Installing mobile app dependencies...${NC}"
    cd "$PROJECT_ROOT/mobile"
    npm install
    print_success "Mobile dependencies installed"

    cd "$PROJECT_ROOT"

    check_env

    print_header "Setup Complete!"
    echo "To start development, run: ./scripts/dev.sh start"
}

start_backend() {
    print_header "Starting Go Backend"

    # Source .env if exists
    if [ -f "$PROJECT_ROOT/.env" ]; then
        source "$PROJECT_ROOT/.env"
    fi

    echo "Backend will be available at: http://localhost:8080"
    echo "Health check: http://localhost:8080/health"
    echo ""
    echo "Press Ctrl+C to stop"
    echo ""

    go run main.go
}

start_mobile() {
    print_header "Starting Mobile App"

    cd "$PROJECT_ROOT/mobile"

    # Get local IP address
    LOCAL_IP=$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || echo "localhost")

    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Mobile App - Connection Options${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "  Web preview:    http://localhost:8081"
    echo "  Expo Go URL:    exp://${LOCAL_IP}:8081"
    echo ""
    echo -e "${YELLOW}To test on your iPhone:${NC}"
    echo "  1. Make sure your phone is on the same WiFi as your computer"
    echo "  2. Install Expo Go from the App Store"
    echo "  3. Open this URL in your browser to see the QR code:"
    echo ""
    echo "     https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=exp%3A%2F%2F${LOCAL_IP}%3A8081"
    echo ""

    # Try to generate QR code in terminal
    if command -v qrencode &> /dev/null; then
        echo -e "${YELLOW}QR Code:${NC}"
        qrencode -t ANSI "exp://${LOCAL_IP}:8081"
        echo ""
    fi

    echo "Press Ctrl+C to stop"
    echo ""

    npx expo start
}

start_all() {
    print_header "Starting Project Alfred"

    check_env

    echo -e "${BLUE}Starting backend and mobile app...${NC}\n"

    # Start backend in background
    echo "Starting Go backend..."
    if [ -f "$PROJECT_ROOT/.env" ]; then
        source "$PROJECT_ROOT/.env"
    fi
    go run main.go &
    BACKEND_PID=$!

    # Wait for backend to start
    sleep 3

    # Check if backend is running
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        print_success "Backend started (PID: $BACKEND_PID)"
    else
        print_warning "Backend may still be starting..."
    fi

    # Start mobile app
    echo -e "\nStarting mobile app..."
    cd "$PROJECT_ROOT/mobile"
    npm run web &
    MOBILE_PID=$!

    print_success "Mobile app started (PID: $MOBILE_PID)"

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Project Alfred is running!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "  Backend API:  http://localhost:8080"
    echo "  Mobile App:   http://localhost:8081"
    echo ""
    echo "  Press Ctrl+C to stop all services"
    echo ""

    # Handle Ctrl+C
    trap "echo ''; echo 'Stopping services...'; kill $BACKEND_PID $MOBILE_PID 2>/dev/null; exit 0" INT TERM

    # Wait for processes
    wait
}

health_check() {
    print_header "Running Health Checks"

    # Check backend
    echo "Checking backend..."
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        HEALTH=$(curl -s http://localhost:8080/health)
        print_success "Backend is running"
        echo "  Response: $HEALTH"
    else
        print_error "Backend is not running"
    fi

    # Check WhatsApp status
    echo -e "\nChecking WhatsApp..."
    if curl -s http://localhost:8080/api/whatsapp/status > /dev/null 2>&1; then
        WA_STATUS=$(curl -s http://localhost:8080/api/whatsapp/status)
        print_success "WhatsApp API responding"
        echo "  Response: $WA_STATUS"
    else
        print_error "WhatsApp API not responding"
    fi

    # Check Google Calendar status
    echo -e "\nChecking Google Calendar..."
    if curl -s http://localhost:8080/api/gcal/status > /dev/null 2>&1; then
        GCAL_STATUS=$(curl -s http://localhost:8080/api/gcal/status)
        print_success "Google Calendar API responding"
        echo "  Response: $GCAL_STATUS"
    else
        print_error "Google Calendar API not responding"
    fi

    # Check mobile app
    echo -e "\nChecking mobile app..."
    if curl -s http://localhost:8081 > /dev/null 2>&1; then
        print_success "Mobile app is running at http://localhost:8081"
    else
        print_warning "Mobile app is not running"
    fi
}

reset_data() {
    print_header "Resetting Project Data"

    echo -e "${YELLOW}This will delete:${NC}"
    echo "  - alfred.db (database)"
    echo "  - whatsapp.db (WhatsApp session)"
    echo "  - token.json (Google Calendar token)"
    echo ""

    read -p "Are you sure? (y/N) " -n 1 -r
    echo ""

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        [ -f "$PROJECT_ROOT/alfred.db" ] && rm "$PROJECT_ROOT/alfred.db" && print_success "Deleted alfred.db"
        [ -f "$PROJECT_ROOT/whatsapp.db" ] && rm "$PROJECT_ROOT/whatsapp.db" && print_success "Deleted whatsapp.db"
        [ -f "$PROJECT_ROOT/token.json" ] && rm "$PROJECT_ROOT/token.json" && print_success "Deleted token.json"

        echo ""
        print_success "Reset complete. Run './scripts/dev.sh start' to start fresh."
    else
        echo "Cancelled."
    fi
}

show_help() {
    echo "Project Alfred - Local Development Script"
    echo ""
    echo "Usage: ./scripts/dev.sh [command]"
    echo ""
    echo "Commands:"
    echo "  start     Start both backend and mobile app (default)"
    echo "  backend   Start only the Go backend"
    echo "  mobile    Start only the mobile app"
    echo "  setup     Install dependencies and setup environment"
    echo "  check     Run health checks"
    echo "  reset     Reset all data (database, WhatsApp session, tokens)"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  ./scripts/dev.sh              # Start everything"
    echo "  ./scripts/dev.sh setup        # First-time setup"
    echo "  ./scripts/dev.sh backend      # Start only backend"
    echo "  ./scripts/dev.sh check        # Check if services are running"
}

# Main command handler
case "${1:-start}" in
    start)
        start_all
        ;;
    backend)
        start_backend
        ;;
    mobile)
        start_mobile
        ;;
    setup)
        setup
        ;;
    check)
        health_check
        ;;
    reset)
        reset_data
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
