# Project Alfred Documentation

This folder contains detailed documentation for Project Alfred.

## Documents

| Document | Description |
|----------|-------------|
| [DEVELOPMENT.md](DEVELOPMENT.md) | Local development setup and workflow |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Production deployment to Railway |

## Quick Links

### For Developers

```bash
# First-time setup
./scripts/dev.sh setup

# Start development environment
./scripts/dev.sh start

# Run health checks
./scripts/dev.sh check
```

See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed instructions.

### For Production

```bash
# Deploy to Railway
./scripts/deploy.sh deploy

# Check production status
./scripts/deploy.sh test

# Configure mobile app for production
./scripts/deploy.sh mobile
```

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed instructions.

## Architecture Overview

Project Alfred is a WhatsApp-to-Google Calendar assistant with:

- **Go Backend** (API only, no web UI)
  - WhatsApp connection via whatsmeow
  - Google Calendar integration via OAuth
  - Claude AI for event detection
  - SQLite database

- **React Native Mobile App** (all UI)
  - Onboarding: WhatsApp pairing codes, Google OAuth
  - Channel management
  - Event review and approval
  - Settings and notifications

## Key Concepts

### WhatsApp Pairing Codes

Instead of QR codes, users connect WhatsApp using 8-digit pairing codes:

1. Enter phone number in mobile app
2. App generates pairing code via backend
3. User enters code in WhatsApp > Linked Devices

### Google Calendar OAuth with Deep Links

Mobile app handles OAuth using deep links (`alfred://oauth/callback`):

1. App opens browser for Google authorization
2. Google redirects to `alfred://oauth/callback?code=...`
3. App captures redirect and exchanges code for tokens

### Mobile-First Architecture

All UI functionality is in the mobile app:
- No web dashboard
- Backend is API-only
- Onboarding, channels, events, and settings all in mobile app
