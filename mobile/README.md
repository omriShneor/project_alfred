# Project Alfred Mobile App

React Native mobile app for Project Alfred - the primary interface for managing WhatsApp-to-Calendar integration.

## Overview

The mobile app provides **all UI functionality** for Project Alfred:
- **Onboarding**: Connect WhatsApp (via pairing code) and Google Calendar (via OAuth)
- **Channels**: Manage which WhatsApp contacts/groups to monitor
- **Events**: Review and approve detected calendar events
- **Settings**: Reconnect services, manage notifications

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

2. Configure API URL in `.env.local`:
   ```bash
   # For local development
   EXPO_PUBLIC_API_BASE_URL=http://localhost:8080

   # For production
   # EXPO_PUBLIC_API_BASE_URL=https://alfred-production-d2c9.up.railway.app
   ```

3. Start the development server:
   ```bash
   npm start
   ```

4. Run on device/simulator:
   - Press `w` for web browser
   - Press `i` for iOS Simulator
   - Press `a` for Android Emulator
   - Scan QR code with Expo Go app on physical device

## App Structure

### Navigation

The app uses conditional navigation based on setup status:

```
App Launch
    │
    ├── Not Set Up? → Onboarding Flow
    │                   ├── Welcome Screen
    │                   ├── WhatsApp Setup (pairing code)
    │                   ├── Google Calendar Setup (OAuth)
    │                   └── Notification Setup
    │
    └── Set Up? → Main Tabs
                    ├── Channels Tab
                    ├── Events Tab
                    └── Settings Tab
```

### Tabs

#### Channels Tab
- Search and filter WhatsApp contacts/groups
- Track/untrack channels for event detection
- Assign Google Calendars to tracked channels

#### Events Tab
- View detected calendar events
- Filter by status (Pending, Synced, Rejected)
- Filter by channel
- Edit event details before syncing
- Confirm events to sync to Google Calendar
- Reject unwanted events
- View message context that triggered the event

#### Settings Tab
- WhatsApp connection status and reconnect
- Google Calendar connection status and reconnect
- Email notification preferences
- App info

## Key Features

### WhatsApp Pairing (No QR Scanning)

Instead of scanning a QR code, users connect WhatsApp using a pairing code:

1. Enter phone number in the app
2. App generates an 8-digit pairing code
3. User enters the code in WhatsApp > Linked Devices > Link with phone number
4. Connection established automatically

This is ideal for mobile-first setup where QR scanning isn't practical.

### Google Calendar OAuth

The app handles OAuth authentication using deep links:

1. User taps "Connect Google Calendar"
2. App opens browser for Google authorization
3. After authorization, Google redirects to `alfred://oauth/callback`
4. App captures the redirect and exchanges the code for tokens

### Real-time Status Updates

The app polls for connection status changes and automatically navigates when services connect.

## Architecture

```
src/
├── api/           # API client and endpoints
│   ├── client.ts      # Axios client with base URL
│   ├── whatsapp.ts    # WhatsApp pairing code API
│   ├── gcal.ts        # Google Calendar OAuth API
│   ├── notifications.ts # Notification preferences
│   ├── channels.ts    # Channel CRUD
│   └── events.ts      # Event CRUD
├── components/    # Reusable UI components
│   ├── common/        # Buttons, Cards, Modals, etc.
│   ├── layout/        # Header, ConnectionStatus
│   ├── channels/      # Channel-specific components
│   └── events/        # Event-specific components
├── config/        # API configuration
├── hooks/         # React Query hooks
│   ├── useChannels.ts
│   ├── useEvents.ts
│   └── useOnboardingStatus.ts
├── navigation/
│   ├── RootNavigator.tsx   # Onboarding vs Main tabs
│   └── TopTabs.tsx         # Channels, Events, Settings
├── screens/
│   ├── ChannelsScreen.tsx
│   ├── EventsScreen.tsx
│   ├── SettingsScreen.tsx
│   └── onboarding/
│       ├── WelcomeScreen.tsx
│       ├── WhatsAppSetupScreen.tsx
│       ├── GoogleCalendarSetupScreen.tsx
│       └── NotificationSetupScreen.tsx
├── theme/         # Colors and typography
└── types/         # TypeScript types
```

## Deep Linking

The app uses the `alfred://` URL scheme for OAuth callbacks:

```typescript
// app.config.ts
scheme: 'alfred',
plugins: ['expo-web-browser'],
```

Supported deep links:
- `alfred://oauth/callback?code=...` - Google Calendar OAuth callback

## Backend Requirements

The mobile app connects to the Project Alfred Go backend API. Required endpoints:

| Endpoint | Purpose |
|----------|---------|
| `GET /api/whatsapp/status` | Check WhatsApp connection |
| `POST /api/whatsapp/pair` | Generate pairing code |
| `GET /api/gcal/status` | Check Google Calendar connection |
| `POST /api/gcal/connect` | Get OAuth URL |
| `POST /api/gcal/callback` | Exchange OAuth code |
| `GET /api/whatsapp/channel` | List channels |
| `GET /api/events` | List events |

## Testing on Physical Device

1. Ensure phone and computer are on the same WiFi network
2. Find your computer's IP: `ipconfig getifaddr en0`
3. Update `.env.local`:
   ```bash
   EXPO_PUBLIC_API_BASE_URL=http://192.168.x.x:8080
   ```
4. Start Expo: `npm start`
5. Scan QR code with Expo Go app

## Building for Production

```bash
# Build for iOS
npx expo build:ios

# Build for Android
npx expo build:android

# Or use EAS Build
npx eas build --platform all
```
