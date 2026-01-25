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

### Step 2: Verify Web Backend

Open in browser:

| URL | What to Check |
|-----|---------------|
| http://localhost:8080 | Dashboard loads, shows WhatsApp QR or connection status |
| http://localhost:8080/settings | Settings page loads with Integrations and Notifications tabs |
| http://localhost:8080/health | Returns JSON: `{"status":"healthy",...}` |

### Step 3: Start the Mobile App (Web Preview)

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

### Step 4: Verify Mobile App

Open http://localhost:8081 in browser:

| Tab | What to Check |
|-----|---------------|
| Channels | List of WhatsApp contacts/groups loads |
| Events | List of detected events loads |
| Connection indicator | Shows green if backend is connected |

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
- [ ] WhatsApp QR code displays on dashboard
- [ ] Google Calendar OAuth flow works

### Mobile App
- [ ] Channel list loads from API
- [ ] Tracked channels appear at top of list
- [ ] Events list loads from API
- [ ] Can confirm/reject events

---

## Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| "ANTHROPIC_API_KEY not set" | Missing env var | Export the API key |
| WhatsApp won't connect | Session expired | Delete `whatsapp.db`, restart |
| Google Calendar errors | Token expired | Delete `token.json`, re-auth |
| Mobile app "Network error" | Wrong API URL | Check `.env.local` has correct URL |

---

## Fresh Start

To reset everything and start fresh:

```bash
rm alfred.db whatsapp.db token.json
go run main.go
```

This clears:
- Database (channels, events, messages)
- WhatsApp session (will show new QR)
- Google Calendar token (will need to re-auth)
