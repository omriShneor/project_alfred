# Project Alfred Scripts

This folder contains automation scripts for development and deployment.

## Scripts

### `dev.sh` - Local Development

```bash
./scripts/dev.sh [command]
```

| Command | Description |
|---------|-------------|
| `start` | Start both backend and mobile app (default) |
| `backend` | Start only the Go backend |
| `mobile` | Start only the mobile app |
| `setup` | Install dependencies and setup environment |
| `check` | Run health checks on running services |
| `reset` | Reset all data (database, WhatsApp, tokens) |
| `help` | Show help message |

**Examples:**

```bash
# First-time setup
./scripts/dev.sh setup

# Start development (backend + mobile)
./scripts/dev.sh start

# Start only backend
./scripts/dev.sh backend

# Check if services are running
./scripts/dev.sh check

# Reset and start fresh
./scripts/dev.sh reset
```

### `deploy.sh` - Production Deployment

```bash
./scripts/deploy.sh [command]
```

| Command | Description |
|---------|-------------|
| `deploy` | Deploy to Railway (default) |
| `status` | Check deployment status |
| `logs` | View deployment logs (live) |
| `env` | List environment variables |
| `env set` | Set an environment variable |
| `mobile` | Configure and start mobile app for production |
| `test` | Run production health checks |
| `guide` | Show step-by-step setup guide |
| `help` | Show help message |

**Examples:**

```bash
# Deploy to Railway
./scripts/deploy.sh

# Check production health
./scripts/deploy.sh test

# View live logs
./scripts/deploy.sh logs

# Set environment variable
./scripts/deploy.sh env set ANTHROPIC_API_KEY "sk-ant-..."

# Configure mobile app for production
./scripts/deploy.sh mobile

# Show full setup guide
./scripts/deploy.sh guide
```

## Prerequisites

### For Development (`dev.sh`)

- Go 1.21+
- Node.js 18+
- npm

### For Deployment (`deploy.sh`)

- Railway CLI (`brew install railway`)
- Railway account (https://railway.app)
- Node.js 18+ (for mobile app)

## Quick Start

### Local Development

```bash
# 1. Setup (first time only)
./scripts/dev.sh setup

# 2. Start development
./scripts/dev.sh start

# 3. Open mobile app at http://localhost:8081
# 4. Complete onboarding (WhatsApp + Google Calendar)
```

### Production Deployment

```bash
# 1. Deploy backend
./scripts/deploy.sh deploy

# 2. Set environment variables
./scripts/deploy.sh env set ANTHROPIC_API_KEY "sk-ant-..."
./scripts/deploy.sh env set GOOGLE_CREDENTIALS_JSON '{"web":{...}}'

# 3. Test deployment
./scripts/deploy.sh test

# 4. Start mobile app for production
./scripts/deploy.sh mobile
```

## Environment Variables

### Required for Backend

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Claude API key for event detection |
| `GOOGLE_CREDENTIALS_JSON` | Google OAuth credentials (JSON string) |

### Required for Railway

| Variable | Value |
|----------|-------|
| `ALFRED_DB_PATH` | `/data/alfred.db` |
| `ALFRED_WHATSAPP_DB_PATH` | `/data/whatsapp.db` |
| `GOOGLE_TOKEN_FILE` | `/data/token.json` |

### Mobile App

The mobile app reads `EXPO_PUBLIC_API_BASE_URL` from `mobile/.env.local`:

```bash
# Local development
EXPO_PUBLIC_API_BASE_URL=http://localhost:8080

# Production
EXPO_PUBLIC_API_BASE_URL=https://alfred-production-d2c9.up.railway.app
```
