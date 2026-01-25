# Testing Guide for Project Alfred

This guide covers how to test both the Go backend (web) and React Native mobile app in local and production environments.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development Testing](#local-development-testing)
3. [Production Testing (Railway)](#production-testing-railway)
4. [Mobile App Testing](#mobile-app-testing)
5. [End-to-End Testing Checklist](#end-to-end-testing-checklist)

---

## Prerequisites

### Required Software

| Software | Purpose | Installation |
|----------|---------|--------------|
| Go 1.21+ | Backend server | `brew install go` |
| Node.js 18+ | Mobile app | `brew install node` |
| Expo Go app | Mobile testing | App Store / Play Store |

### Required Accounts & Credentials

- **Anthropic API Key** - For Claude event detection
- **Google Cloud Project** - For Calendar integration (OAuth credentials)
- **Railway Account** - For production deployment (optional)

### Environment Setup

Create a `.env` file in the project root (for local testing):
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
# Google credentials.json should be in project root
```

---

## Local Development Testing

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

### Step 4: Verify Mobile App (Web Preview)

Open http://localhost:8081 in browser:

| Tab | What to Check |
|-----|---------------|
| Channels | List of WhatsApp contacts/groups loads |
| Events | List of detected events loads |
| Connection indicator | Shows green if backend is connected |

---

## Production Testing (Railway)

### Step 1: Deploy to Railway

```bash
cd /Users/omrishneor/code/project_alfred

# Login to Railway
railway login

# Deploy
railway up

# Get your domain (if not set)
railway domain
```

### Step 2: Verify Production Backend

Open in browser:

| URL | What to Check |
|-----|---------------|
| https://alfred-production-d2c9.up.railway.app | Dashboard loads |
| https://alfred-production-d2c9.up.railway.app/settings | Settings page loads |
| https://alfred-production-d2c9.up.railway.app/health | Returns healthy status |

### Step 3: Check Railway Logs

```bash
railway logs
```

Look for:
- `HTTP server starting on :8080`
- `WhatsApp client connected` (after QR scan)
- No error messages

### Step 4: Connect WhatsApp (Production)

1. Open https://alfred-production-d2c9.up.railway.app
2. Scan QR code with WhatsApp on your phone
3. Wait for "Connected" status

### Step 5: Connect Google Calendar (Production)

1. Go to https://alfred-production-d2c9.up.railway.app/settings
2. Click "Connect Google Calendar"
3. Complete OAuth flow
4. Verify "Connected" status

---

## Mobile App Testing

### Option A: Web Browser (Quickest)

Best for rapid UI testing without device setup.

```bash
cd mobile
npm run web
```

Open http://localhost:8081

### Option B: iOS Simulator

Requires Xcode installed.

```bash
cd mobile
npm run ios
```

### Option C: Android Emulator

Requires Android Studio installed.

```bash
cd mobile
npm run android
```

### Option D: Physical iPhone with Expo Go (Recommended for Real Testing)

This tests the actual mobile experience.

#### Prerequisites
1. Install "Expo Go" from App Store on your iPhone
2. iPhone and computer must be on the **same WiFi network**

#### Step 1: Find Your Computer's IP Address

```bash
ipconfig getifaddr en0
```

Example output: `192.168.1.100`

#### Step 2: Configure Mobile App for Your Network

Edit `mobile/.env.local`:

```bash
# For local backend testing
EXPO_PUBLIC_API_BASE_URL=http://192.168.1.100:8080

# OR for production backend testing
EXPO_PUBLIC_API_BASE_URL=https://alfred-production-d2c9.up.railway.app
```

#### Step 3: Start Expo Development Server

```bash
cd mobile
npm start
```

**Expected output:**
```
Metro waiting on exp://192.168.1.100:8081
Scan the QR code above with Expo Go (Android) or the Camera app (iOS)
```

#### Step 4: Open on iPhone

1. Open Camera app on iPhone
2. Point at the QR code in terminal
3. Tap the notification to open in Expo Go
4. App loads and connects to your backend

#### Troubleshooting Expo Go

| Issue | Solution |
|-------|----------|
| QR code won't scan | Type URL manually in Expo Go |
| "Network request failed" | Check iPhone and computer on same WiFi |
| Can't reach backend | Verify IP address, check firewall settings |
| Slow loading | First load takes time, subsequent loads are faster |

---

## End-to-End Testing Checklist

### Backend Health Check

- [ ] `GET /health` returns 200 with status "healthy"
- [ ] WhatsApp status shows "connected" (after QR scan)
- [ ] Google Calendar status shows "connected" (after OAuth)

### WhatsApp Integration

- [ ] QR code displays on dashboard
- [ ] QR code can be scanned with WhatsApp
- [ ] Connection persists after page refresh
- [ ] Reconnect button generates new QR if needed

### Google Calendar Integration

- [ ] "Connect" button initiates OAuth flow
- [ ] OAuth callback completes successfully
- [ ] Calendar list loads after connection
- [ ] Selected calendar is saved

### Channel Management (Mobile App)

- [ ] Channel list loads from API
- [ ] Tracked channels appear at top of list
- [ ] Can track a new channel (select calendar, enable)
- [ ] Can untrack a channel
- [ ] Search/filter works

### Event Detection & Review (Mobile App)

- [ ] Events list loads from API
- [ ] Filter by status works (Pending, Confirmed, etc.)
- [ ] Can view event details
- [ ] Can edit pending event
- [ ] Can confirm event (syncs to Google Calendar)
- [ ] Can reject event

### Notifications (Settings Page)

- [ ] Can enter email address
- [ ] Can toggle notification preferences
- [ ] Settings persist after page refresh

---

## Testing Scenarios

### Scenario 1: New User Setup

1. Start fresh (delete `alfred.db`, `whatsapp.db`, `token.json`)
2. Start backend: `go run main.go`
3. Open http://localhost:8080
4. Scan WhatsApp QR code
5. Connect Google Calendar
6. Open mobile app
7. Track a WhatsApp contact/group
8. Send a test message with event details
9. Verify event appears in mobile app
10. Confirm event, verify it syncs to Google Calendar

### Scenario 2: Production Deployment

1. Deploy to Railway: `railway up`
2. Open production URL
3. Scan WhatsApp QR (note: session may persist)
4. Verify Google Calendar connected
5. Configure mobile app with production URL
6. Test channel tracking and event flow

### Scenario 3: Mobile-Only Testing

1. Start backend (local or production)
2. Start mobile app: `npm run web` or Expo Go
3. Verify connection status indicator
4. Test all channel operations
5. Test all event operations
6. Verify changes reflect in backend

---

## Quick Reference Commands

```bash
# Backend
go run main.go                    # Start local backend
railway up                        # Deploy to Railway
railway logs                      # View production logs

# Mobile App
cd mobile
npm install                       # Install dependencies
npm run web                       # Web browser preview
npm run ios                       # iOS Simulator
npm run android                   # Android Emulator
npm start                         # Expo Go (scan QR)

# Utilities
ipconfig getifaddr en0            # Get local IP (macOS)
curl http://localhost:8080/health # Quick health check
```

---

## Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| "ANTHROPIC_API_KEY not set" | Missing env var | Export the API key |
| WhatsApp won't connect | Session expired | Delete `whatsapp.db`, restart |
| Google Calendar errors | Token expired | Delete `token.json`, re-auth |
| Mobile app "Network error" | Wrong API URL | Check `.env.local` has correct URL |
| Expo Go can't connect | Different networks | Ensure same WiFi network |
| Events not detecting | Claude API issue | Check API key, view backend logs |
