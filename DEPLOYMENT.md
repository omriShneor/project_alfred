# Production Deployment Guide

This guide covers deploying Project Alfred to Railway and testing the mobile app on a physical device.

---

## Table of Contents

1. [Railway Deployment](#railway-deployment)
2. [Testing on iPhone with Expo Go](#testing-on-iphone-with-expo-go)
3. [End-to-End Production Testing](#end-to-end-production-testing)

---

## Railway Deployment

### Prerequisites

- Railway CLI installed: `brew install railway`
- Railway account: https://railway.app

### Step 1: Deploy to Railway

```bash
cd /Users/omrishneor/code/project_alfred

# Login to Railway (first time)
railway login

# Deploy
railway up

# Get your domain (if not set)
railway domain
```

### Step 2: Set Environment Variables

In Railway dashboard or CLI:

```bash
railway variables set ANTHROPIC_API_KEY="sk-ant-..."
railway variables set ALFRED_DB_PATH="/data/alfred.db"
railway variables set ALFRED_WHATSAPP_DB_PATH="/data/whatsapp.db"
railway variables set GOOGLE_CREDENTIALS_JSON='{"web":{...}}'
railway variables set GOOGLE_TOKEN_FILE="/data/token.json"
```

### Step 3: Verify Production Backend

Open in browser:

| URL | What to Check |
|-----|---------------|
| https://alfred-production-d2c9.up.railway.app | Dashboard loads |
| https://alfred-production-d2c9.up.railway.app/settings | Settings page loads |
| https://alfred-production-d2c9.up.railway.app/health | Returns `{"status":"healthy",...}` |

### Step 4: Check Logs

```bash
railway logs
```

Look for:
- `HTTP server starting on :8080`
- `WhatsApp client connected` (after QR scan)
- No error messages

### Step 5: Connect WhatsApp

1. Open https://alfred-production-d2c9.up.railway.app
2. Scan QR code with WhatsApp on your phone
3. Wait for "Connected" status

### Step 6: Connect Google Calendar

1. Go to https://alfred-production-d2c9.up.railway.app/settings
2. Click "Connect Google Calendar"
3. Complete OAuth flow
4. Verify "Connected" status

**Note:** Ensure your Google Cloud Console has the Railway callback URL:
```
https://alfred-production-d2c9.up.railway.app/oauth/callback
```

---

## Testing on iPhone with Expo Go

### Prerequisites

1. **Expo Go app** - Download from App Store
2. **Same WiFi network** - iPhone and computer must be connected to the same network

### Option A: Connect to Production Backend (Recommended)

This is the simplest setup - your phone connects directly to the Railway deployment.

#### Step 1: Configure Mobile App

Edit `mobile/.env.local`:
```bash
EXPO_PUBLIC_API_BASE_URL=https://alfred-production-d2c9.up.railway.app
```

#### Step 2: Start Expo

```bash
cd mobile
npm start
```

#### Step 3: Open on iPhone

1. Open **Camera app** on iPhone
2. Point at the QR code shown in terminal
3. Tap the notification banner to open in Expo Go
4. App loads and connects to production backend

### Option B: Connect to Local Backend

Use this when testing backend changes before deploying.

#### Step 1: Find Your Computer's IP Address

```bash
ipconfig getifaddr en0
```

Example output: `192.168.1.100`

#### Step 2: Configure Mobile App

Edit `mobile/.env.local`:
```bash
EXPO_PUBLIC_API_BASE_URL=http://192.168.1.100:8080
```

**Important:** Use your actual IP, not `localhost`.

#### Step 3: Start Local Backend

```bash
cd /Users/omrishneor/code/project_alfred
go run main.go
```

#### Step 4: Start Expo

```bash
cd mobile
npm start
```

#### Step 5: Open on iPhone

1. Scan QR code with Camera app
2. Open in Expo Go
3. App connects to your local backend

### Troubleshooting Expo Go

| Issue | Solution |
|-------|----------|
| QR code won't scan | Open Expo Go app, tap "Enter URL manually" |
| "Network request failed" | Verify iPhone and computer on same WiFi |
| Can't reach local backend | Check IP address, ensure no firewall blocking port 8080 |
| Slow first load | Normal - Metro bundler needs to compile, subsequent loads are faster |
| App crashes on start | Clear Expo Go cache: shake phone → "Reload" |

### Alternative: Enter URL Manually

If QR scanning doesn't work:

1. Open Expo Go app
2. Tap "Enter URL manually"
3. Enter: `exp://YOUR_COMPUTER_IP:8081`
4. Tap "Connect"

---

## End-to-End Production Testing

### Full Flow Test

1. **Deploy backend**
   ```bash
   railway up
   ```

2. **Verify backend health**
   ```bash
   curl https://alfred-production-d2c9.up.railway.app/health
   ```

3. **Connect WhatsApp** - Scan QR on production dashboard

4. **Connect Google Calendar** - Complete OAuth on settings page

5. **Configure mobile app for production**
   ```bash
   # mobile/.env.local
   EXPO_PUBLIC_API_BASE_URL=https://alfred-production-d2c9.up.railway.app
   ```

6. **Start mobile app**
   ```bash
   cd mobile && npm start
   ```

7. **Open on iPhone** - Scan QR with Camera

8. **Test channel tracking**
   - Open Channels tab
   - Find a WhatsApp contact/group
   - Enable tracking, select calendar

9. **Test event detection**
   - Send a message with event details to tracked contact/group
   - Example: "Let's meet tomorrow at 3pm at the coffee shop"
   - Wait for Claude to detect the event

10. **Review event in mobile app**
    - Open Events tab
    - Find the pending event
    - Review details, edit if needed
    - Confirm the event

11. **Verify Google Calendar**
    - Open Google Calendar
    - Confirm event appears on selected calendar

### Production Checklist

#### Backend
- [ ] Railway deployment successful
- [ ] Health endpoint returns healthy
- [ ] WhatsApp connected (QR scanned)
- [ ] Google Calendar connected (OAuth complete)
- [ ] Logs show no errors

#### Mobile App
- [ ] Connects to production backend
- [ ] Channel list loads
- [ ] Can track/untrack channels
- [ ] Events list loads
- [ ] Can confirm/reject events
- [ ] Events sync to Google Calendar

---

## Quick Reference

```bash
# Railway
railway login                     # Login (first time)
railway up                        # Deploy
railway logs                      # View logs
railway domain                    # Get/set domain

# Mobile App
cd mobile
npm start                         # Start Expo (for phone testing)
npm run web                       # Web preview (for quick testing)

# Get local IP (for local backend + phone testing)
ipconfig getifaddr en0
```

---

## URLs

| Environment | Backend URL |
|-------------|-------------|
| Local | http://localhost:8080 |
| Production | https://alfred-production-d2c9.up.railway.app |

| Environment | Mobile App URL |
|-------------|----------------|
| Web Preview | http://localhost:8081 |
| Expo Go | exp://YOUR_IP:8081 |
