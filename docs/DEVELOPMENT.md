# Local Development Guide

This guide covers setting up and running Project Alfred locally for development.

---

## Prerequisites

### Required Software

| Software | Purpose | Installation |
|----------|---------|--------------|
| Go 1.21+ | Backend server | `brew install go` |
| Node.js 18+ | Mobile app | `brew install node` |

### Required Credentials

- **Anthropic API Key** - For Claude event detection
- **Google Cloud Project** - For Calendar integration (OAuth credentials)

### Environment Setup

Create a `.env` file in the project root:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
# Google credentials.json should be in project root
```

---

## Running Locally

### Step 1: Start the Go Backend

```bash
cd /Users/omrishneor/code/project_alfred

# Load environment variables
source .env  # or export ANTHROPIC_API_KEY="sk-ant-..."

# Start the server
go run main.go
```

**Expected output:**
```
Starting Project Alfred...
HTTP server starting on :8080
WhatsApp client connecting...
```

### Step 2: Verify Backend API

Test the backend is running:

```bash
curl http://localhost:8080/health
```

**Expected response:**
```json
{"status":"healthy","whatsapp":"disconnected","gcal":"disconnected"}
```

### Step 3: Start the Mobile App

In a new terminal:

```bash
cd /Users/omrishneor/code/project_alfred/mobile

# Install dependencies (first time only)
npm install

# Start with web preview
npm run web
```

**Expected output:**
```
Starting Metro Bundler
env: load .env.local
Waiting on http://localhost:8081
```

### Step 4: Complete Setup via Mobile App

Open http://localhost:8081 in browser:

1. **Welcome Screen** - Tap "Get Started"

2. **WhatsApp Setup**
   - Enter your phone number (with country code)
   - Tap "Generate Pairing Code"
   - Open WhatsApp on your phone
   - Go to Settings > Linked Devices > Link a Device
   - Select "Link with phone number instead"
   - Enter the 8-digit code shown in the app
   - Wait for "Connected" status

3. **Google Calendar Setup**
   - Tap "Connect Google Calendar"
   - Complete Google OAuth in the browser popup
   - Wait for redirect back to app
   - Verify "Connected" status

4. **Notifications** (optional)
   - Enable email notifications if desired
   - Tap "Complete Setup"

### Step 5: Verify Everything Works

| Tab | What to Check |
|-----|---------------|
| Channels | List of WhatsApp contacts/groups loads |
| Events | Empty list (or previous events if database exists) |
| Settings | WhatsApp and Google Calendar show "Connected" |

---

## Mobile App Configuration

The mobile app reads the API URL from `mobile/.env.local`:

```bash
# For local backend
EXPO_PUBLIC_API_BASE_URL=http://localhost:8080
```

---

## Quick Reference Commands

```bash
# Backend
go run main.go                    # Start local backend
curl http://localhost:8080/health # Quick health check

# Mobile App
cd mobile
npm install                       # Install dependencies (first time)
npm run web                       # Web browser preview
npm run ios                       # iOS Simulator (requires Xcode)
npm run android                   # Android Emulator (requires Android Studio)
```

---

## Testing Checklist

### Backend Health
- [ ] `GET /health` returns 200 with status "healthy"
- [ ] `GET /api/whatsapp/status` returns connection status
- [ ] `GET /api/gcal/status` returns connection status

### Mobile App - Onboarding
- [ ] Welcome screen displays
- [ ] WhatsApp pairing code generates successfully
- [ ] WhatsApp connects after entering code
- [ ] Google Calendar OAuth flow completes
- [ ] Notification setup works

### Mobile App - Main Features
- [ ] Channel list loads from API
- [ ] Tracked channels appear at top of list
- [ ] Events list loads from API
- [ ] Can confirm/reject events
- [ ] Settings show correct connection status

---

## Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| "ANTHROPIC_API_KEY not set" | Missing env var | Export the API key |
| WhatsApp won't connect | Session expired | Delete `whatsapp.db`, restart |
| Pairing code fails | Already linked | Unlink device in WhatsApp first |
| Google Calendar errors | Token expired | Delete `token.json`, reconnect via app |
| Mobile app "Network error" | Wrong API URL | Check `.env.local` has correct URL |
| OAuth redirect fails | Missing redirect URI | Add `alfred://oauth/callback` to Google Console |

---

## Fresh Start

To reset everything and start fresh:

```bash
rm alfred.db whatsapp.db token.json
go run main.go
```

This clears:
- Database (channels, events, messages)
- WhatsApp session (will need to pair again)
- Google Calendar token (will need to reconnect)

---

## API Endpoints for Development

### WhatsApp
```bash
# Check status
curl http://localhost:8080/api/whatsapp/status

# Generate pairing code
curl -X POST http://localhost:8080/api/whatsapp/pair \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+1234567890"}'

# Disconnect
curl -X POST http://localhost:8080/api/whatsapp/disconnect
```

### Google Calendar
```bash
# Check status
curl http://localhost:8080/api/gcal/status

# Get OAuth URL (with custom redirect for mobile)
curl -X POST http://localhost:8080/api/gcal/connect \
  -H "Content-Type: application/json" \
  -d '{"redirect_uri": "alfred://oauth/callback"}'

# Exchange code for token
curl -X POST http://localhost:8080/api/gcal/callback \
  -H "Content-Type: application/json" \
  -d '{"code": "AUTH_CODE", "redirect_uri": "alfred://oauth/callback"}'
```

### Channels & Events
```bash
# List channels
curl http://localhost:8080/api/channel

# List events
curl http://localhost:8080/api/events

# List pending events
curl "http://localhost:8080/api/events?status=pending"
```
