# Project Alfred Mobile App

React Native mobile companion app for Project Alfred - manage channels and review calendar events on the go.

## Prerequisites

- Node.js 18+
- npm or yarn
- Expo CLI (`npm install -g expo-cli`)
- iOS Simulator (macOS) or Android Emulator

## Setup

1. Install dependencies:
   ```bash
   cd mobile
   npm install
   ```

2. Add placeholder assets (required):
   ```bash
   # Create placeholder icons (replace with actual icons later)
   # icon.png: 1024x1024
   # adaptive-icon.png: 1024x1024
   # splash-icon.png: 1284x2778
   # favicon.png: 48x48
   ```

3. Start the development server:
   ```bash
   npm start
   ```

4. Run on device/simulator:
   - Press `i` for iOS Simulator
   - Press `a` for Android Emulator
   - Scan QR code with Expo Go app on physical device

## Configuration

Edit `src/config/api.ts` to change the API base URL:

```typescript
// Production (Railway)
export const API_BASE_URL = 'https://alfred-production-d2c9.up.railway.app';

// Local development
// export const API_BASE_URL = 'http://localhost:8080';
```

## Features

### Channels Tab
- Search and filter WhatsApp contacts/groups
- Track/untrack channels for event detection
- Assign calendars to tracked channels

### Events Tab
- View detected calendar events
- Filter by status (Pending, Synced, Rejected)
- Filter by channel
- Edit event details
- Confirm events to sync to Google Calendar
- Reject unwanted events
- View message context that triggered the event

### Connection Status
- Green dot: All services connected
- Red dot: Connection issue - tap for details
- "Open Web Settings" button to fix issues in browser

## Architecture

```
src/
├── api/           # API client and endpoints
├── components/    # Reusable UI components
│   ├── common/    # Buttons, Cards, Modals, etc.
│   ├── layout/    # Header, ConnectionStatus
│   ├── channels/  # Channel-specific components
│   └── events/    # Event-specific components
├── config/        # API configuration
├── hooks/         # React Query hooks
├── navigation/    # Tab navigation
├── screens/       # Screen components
├── theme/         # Colors and typography
└── types/         # TypeScript types
```

## User Flow

1. **First time (Web):** Set up WhatsApp QR + Google Calendar on web UI
2. **Daily use (Mobile):** Review/approve events, manage channels
3. **If disconnected:** Tap red status dot → "Open Web Settings"

## Backend Requirements

The mobile app connects to the existing Project Alfred Go backend. No backend changes required - all needed endpoints already exist with CORS enabled.
