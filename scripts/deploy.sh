#!/bin/bash
#
# Project Alfred - Production Deployment Script
# Usage: ./scripts/deploy.sh [command]
#
# Commands:
#   deploy    - Deploy to Railway (default)
#   status    - Check deployment status
#   logs      - View deployment logs
#   env       - Set/view environment variables
#   mobile    - Configure and start mobile app for production
#   test      - Run production health checks
#   help      - Show this help message
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Get the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Production URL
PRODUCTION_URL="https://alfred-production-d2c9.up.railway.app"

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

print_info() {
    echo -e "${CYAN}ℹ $1${NC}"
}

check_railway_cli() {
    if ! command -v railway &> /dev/null; then
        print_error "Railway CLI is not installed"
        echo ""
        echo "Install with: brew install railway"
        echo "Or visit: https://docs.railway.app/develop/cli"
        exit 1
    fi
    print_success "Railway CLI found"
}

check_railway_auth() {
    if ! railway whoami &> /dev/null 2>&1; then
        print_warning "Not logged in to Railway"
        echo ""
        echo "Run: railway login"
        exit 1
    fi
    RAILWAY_USER=$(railway whoami 2>/dev/null)
    print_success "Logged in as: $RAILWAY_USER"
}

deploy() {
    print_header "Deploying to Railway"

    check_railway_cli
    check_railway_auth

    echo -e "\n${BLUE}Building and deploying...${NC}\n"

    # Deploy to Railway
    railway up

    print_success "Deployment initiated!"

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Deployment Complete${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "  Production URL: $PRODUCTION_URL"
    echo ""
    echo "  Next steps:"
    echo "    1. Check logs: ./scripts/deploy.sh logs"
    echo "    2. Run health check: ./scripts/deploy.sh test"
    echo "    3. Configure mobile app: ./scripts/deploy.sh mobile"
    echo ""
}

show_status() {
    print_header "Deployment Status"

    check_railway_cli
    check_railway_auth

    echo "Fetching deployment status..."
    echo ""

    railway status

    echo ""
    echo "Production URL: $PRODUCTION_URL"
}

show_logs() {
    print_header "Deployment Logs"

    check_railway_cli
    check_railway_auth

    echo "Fetching logs (Ctrl+C to exit)..."
    echo ""

    railway logs
}

manage_env() {
    print_header "Environment Variables"

    check_railway_cli
    check_railway_auth

    if [ -z "$2" ]; then
        echo "Current environment variables:"
        echo ""
        railway variables
        echo ""
        echo -e "${CYAN}Required variables:${NC}"
        echo "  ANTHROPIC_API_KEY        - Claude API key for event detection"
        echo "  ALFRED_DB_PATH           - Database path (use /data/alfred.db)"
        echo "  ALFRED_WHATSAPP_DB_PATH  - WhatsApp DB path (use /data/whatsapp.db)"
        echo "  GOOGLE_CREDENTIALS_JSON  - Google OAuth credentials (JSON string)"
        echo "  GOOGLE_TOKEN_FILE        - Token file path (use /data/token.json)"
        echo ""
        echo -e "${CYAN}Optional variables:${NC}"
        echo "  ALFRED_RESEND_API_KEY    - Resend API key for email notifications"
        echo "  ALFRED_BASE_URL          - Base URL for OAuth callbacks"
        echo ""
        echo "To set a variable: ./scripts/deploy.sh env set VAR_NAME \"value\""
    elif [ "$2" = "set" ] && [ -n "$3" ] && [ -n "$4" ]; then
        echo "Setting $3..."
        railway variables set "$3=$4"
        print_success "Variable $3 set"
    else
        echo "Usage:"
        echo "  ./scripts/deploy.sh env              # List all variables"
        echo "  ./scripts/deploy.sh env set NAME VALUE  # Set a variable"
    fi
}

configure_mobile() {
    print_header "Configure Mobile App for Production"

    # Update .env.local
    ENV_FILE="$PROJECT_ROOT/mobile/.env.local"

    echo "Configuring mobile app to use production backend..."
    echo ""

    echo "EXPO_PUBLIC_API_BASE_URL=$PRODUCTION_URL" > "$ENV_FILE"
    print_success "Updated $ENV_FILE"
    echo "  EXPO_PUBLIC_API_BASE_URL=$PRODUCTION_URL"

    echo ""
    read -p "Start mobile app now? (Y/n) " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        echo ""
        start_mobile_with_qr
    else
        echo ""
        echo "To start later, run:"
        echo "  cd mobile && npm start"
    fi
}

start_mobile_with_qr() {
    # Get local IP address
    LOCAL_IP=$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || echo "localhost")

    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Mobile App - Expo Go Connection${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "  Web preview:    http://localhost:8081"
    echo "  Expo Go URL:    exp://${LOCAL_IP}:8081"
    echo ""
    echo -e "${CYAN}To test on your iPhone:${NC}"
    echo "  1. Make sure your phone is on the same WiFi as your computer"
    echo "  2. Install Expo Go from the App Store"
    echo "  3. Scan the QR code below or open this URL in your browser:"
    echo ""
    echo "     https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=exp%3A%2F%2F${LOCAL_IP}%3A8081"
    echo ""

    # Try to generate QR code in terminal
    if command -v qrencode &> /dev/null; then
        echo -e "${CYAN}QR Code:${NC}"
        qrencode -t ANSI "exp://${LOCAL_IP}:8081"
        echo ""
    fi

    echo "Press Ctrl+C to stop"
    echo ""

    cd "$PROJECT_ROOT/mobile"
    npx expo start
}

run_tests() {
    print_header "Production Health Checks"

    echo "Checking: $PRODUCTION_URL"
    echo ""

    # Health check
    echo "1. Health endpoint..."
    if HEALTH=$(curl -s -f "$PRODUCTION_URL/health" 2>/dev/null); then
        print_success "Backend is healthy"
        echo "   Response: $HEALTH"
    else
        print_error "Backend health check failed"
        echo "   URL: $PRODUCTION_URL/health"
    fi

    # WhatsApp status
    echo ""
    echo "2. WhatsApp status..."
    if WA_STATUS=$(curl -s -f "$PRODUCTION_URL/api/whatsapp/status" 2>/dev/null); then
        print_success "WhatsApp API responding"
        echo "   Response: $WA_STATUS"
    else
        print_error "WhatsApp API not responding"
    fi

    # Google Calendar status
    echo ""
    echo "3. Google Calendar status..."
    if GCAL_STATUS=$(curl -s -f "$PRODUCTION_URL/api/gcal/status" 2>/dev/null); then
        print_success "Google Calendar API responding"
        echo "   Response: $GCAL_STATUS"
    else
        print_error "Google Calendar API not responding"
    fi

    # Onboarding status
    echo ""
    echo "4. Onboarding status..."
    if ONBOARD_STATUS=$(curl -s -f "$PRODUCTION_URL/api/onboarding/status" 2>/dev/null); then
        print_success "Onboarding API responding"
        echo "   Response: $ONBOARD_STATUS"
    else
        print_error "Onboarding API not responding"
    fi

    echo ""
    echo -e "${BLUE}========================================${NC}"

    # Summary
    if curl -s -f "$PRODUCTION_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}Production backend is running!${NC}"
        echo ""
        echo "Next steps:"
        echo "  1. Configure mobile app: ./scripts/deploy.sh mobile"
        echo "  2. Complete setup in mobile app (WhatsApp + Google Calendar)"
    else
        echo -e "${RED}Production backend is not responding${NC}"
        echo ""
        echo "Troubleshooting:"
        echo "  1. Check logs: ./scripts/deploy.sh logs"
        echo "  2. Verify deployment: ./scripts/deploy.sh status"
        echo "  3. Check environment variables: ./scripts/deploy.sh env"
    fi
}

setup_guide() {
    print_header "Production Setup Guide"

    echo -e "${CYAN}Step 1: Deploy Backend${NC}"
    echo "  ./scripts/deploy.sh deploy"
    echo ""

    echo -e "${CYAN}Step 2: Set Environment Variables${NC}"
    echo "  ./scripts/deploy.sh env set ANTHROPIC_API_KEY \"sk-ant-...\""
    echo "  ./scripts/deploy.sh env set ALFRED_DB_PATH \"/data/alfred.db\""
    echo "  ./scripts/deploy.sh env set ALFRED_WHATSAPP_DB_PATH \"/data/whatsapp.db\""
    echo "  ./scripts/deploy.sh env set GOOGLE_TOKEN_FILE \"/data/token.json\""
    echo "  ./scripts/deploy.sh env set GOOGLE_CREDENTIALS_JSON '{\"web\":{...}}'"
    echo ""

    echo -e "${CYAN}Step 3: Configure Google Cloud Console${NC}"
    echo "  Add these redirect URIs to your OAuth client:"
    echo "    - $PRODUCTION_URL/oauth/callback"
    echo "    - alfred://oauth/callback"
    echo ""

    echo -e "${CYAN}Step 4: Test Deployment${NC}"
    echo "  ./scripts/deploy.sh test"
    echo ""

    echo -e "${CYAN}Step 5: Configure Mobile App${NC}"
    echo "  ./scripts/deploy.sh mobile"
    echo ""

    echo -e "${CYAN}Step 6: Complete Setup in Mobile App${NC}"
    echo "  1. Open mobile app on phone (scan QR with Expo Go)"
    echo "  2. Enter phone number and generate pairing code"
    echo "  3. Link WhatsApp using the pairing code"
    echo "  4. Connect Google Calendar via OAuth"
    echo "  5. Configure notification preferences"
    echo ""

    echo "For detailed instructions, see: docs/DEPLOYMENT.md"
}

show_help() {
    echo "Project Alfred - Production Deployment Script"
    echo ""
    echo "Usage: ./scripts/deploy.sh [command]"
    echo ""
    echo "Commands:"
    echo "  deploy    Deploy to Railway (default)"
    echo "  status    Check deployment status"
    echo "  logs      View deployment logs (live)"
    echo "  env       List environment variables"
    echo "  env set   Set an environment variable"
    echo "  mobile    Configure and start mobile app for production"
    echo "  test      Run production health checks"
    echo "  guide     Show step-by-step setup guide"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  ./scripts/deploy.sh                    # Deploy to Railway"
    echo "  ./scripts/deploy.sh test               # Check production health"
    echo "  ./scripts/deploy.sh env set KEY value  # Set environment variable"
    echo "  ./scripts/deploy.sh mobile             # Start mobile app for production"
    echo ""
    echo "Production URL: $PRODUCTION_URL"
}

# Main command handler
case "${1:-deploy}" in
    deploy)
        deploy
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs
        ;;
    env)
        manage_env "$@"
        ;;
    mobile)
        configure_mobile
        ;;
    test)
        run_tests
        ;;
    guide)
        setup_guide
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
