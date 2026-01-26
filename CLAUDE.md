# Project Alfred

WhatsApp-to-Google Calendar assistant that uses Claude AI to detect calendar events from conversations, creates pending events for review, and syncs confirmed events to Google Calendar.

## Quick Reference

### Key Commands
```bash
# Development (see docs/DEVELOPMENT.md)
./scripts/dev.sh setup      # First-time setup
./scripts/dev.sh start      # Start backend + mobile app
./scripts/dev.sh check      # Health checks

# Production (see docs/DEPLOYMENT.md)
./scripts/deploy.sh         # Deploy to Railway
./scripts/deploy.sh test    # Production health checks
./scripts/deploy.sh mobile  # Configure mobile for production
```

### Manual Commands
```bash
go run main.go              # Start backend only
cd mobile && npm run web    # Start mobile app only
```

### Important URLs

**Go Backend (port 8080):**
- `http://localhost:8080/health` - Health check endpoint
- `http://localhost:8080/api/*` - API endpoints (no web UI)

**Mobile App (port 8081):**
- `http://localhost:8081/` - Expo web preview (all UI)

**Production:**
- https://alfred-production-d2c9.up.railway.app (API only)

### Required Environment
```bash
export ANTHROPIC_API_KEY="sk-..."  # Required for event detection
# Place credentials.json in project root for Google Calendar
# Or set GOOGLE_CREDENTIALS_JSON env var with JSON content
```

---

## Architecture Overview

```
WhatsApp Message
    ↓
whatsapp/handler.go (filter by tracked channel)
    ↓
processor/processor.go (store message, call Claude)
    ↓
claude/client.go (analyze for events)
    ↓
database/events.go (create pending event)
    ↓
User reviews in Mobile App (Events tab)
    ↓
gcal/events.go (sync to Google Calendar)
```

### Mobile-Only UI
All UI functionality is in the mobile app. The Go backend is API-only.

| Component | Platform | Purpose |
|-----------|----------|---------|
| Go Backend | `localhost:8080` | API endpoints, WhatsApp connection, event detection |
| Mobile App | `localhost:8081` | All UI: onboarding, channels, events, settings |

**WhatsApp Connection:** Uses pairing codes (not QR scanning) - user enters phone number in app, gets 8-digit code to enter in WhatsApp.

**Google Calendar OAuth:** Uses deep link callback (`alfred://oauth/callback`) - app opens browser for auth, captures redirect back to app.

---

## Project Structure

```
project_alfred/
├── main.go                      # Entry point, initialization, shutdown
├── Dockerfile                   # Multi-stage Docker build for Railway
├── railway.toml                 # Railway deployment configuration
├── scripts/
│   ├── dev.sh                   # Local development script
│   ├── deploy.sh                # Production deployment script
│   └── README.md                # Scripts documentation
├── docs/
│   ├── DEVELOPMENT.md           # Local development guide
│   ├── DEPLOYMENT.md            # Production deployment guide
│   └── README.md                # Documentation index
├── internal/
│   ├── config/
│   │   └── env.go               # Environment configuration
│   ├── database/
│   │   ├── database.go          # SQLite setup, migrations
│   │   ├── channels.go          # Channel CRUD
│   │   ├── messages.go          # Message history storage
│   │   ├── events.go            # Calendar event CRUD
│   │   ├── attendees.go         # Event attendee management
│   │   └── notifications.go     # User notification preferences
│   ├── server/
│   │   ├── server.go            # HTTP server, routing, CORS middleware
│   │   └── handlers.go          # All HTTP handlers (API only, no static files)
│   ├── whatsapp/
│   │   ├── client.go            # WhatsApp connection
│   │   ├── handler.go           # Message filtering
│   │   ├── groups.go            # Channel discovery
│   │   └── qr.go                # QR code generation
│   ├── gcal/
│   │   ├── client.go            # Google Calendar client
│   │   ├── auth.go              # OAuth2 flow (supports env var credentials)
│   │   ├── calendars.go         # List calendars
│   │   └── events.go            # Event CRUD
│   ├── claude/
│   │   ├── client.go            # Claude API client
│   │   └── prompt.go            # System prompt for event detection
│   ├── processor/
│   │   ├── processor.go         # Message processing loop
│   │   └── history.go           # Message storage helper
│   ├── notify/
│   │   ├── notifier.go          # Notification interface
│   │   ├── resend.go            # Email notifications via Resend
│   │   └── service.go           # Notification service orchestrator
│   ├── onboarding/
│   │   ├── onboarding.go        # Setup orchestration
│   │   └── clients.go           # Client container
│   └── sse/
│       └── state.go             # Onboarding SSE state
├── mobile/                      # React Native mobile app (Expo)
│   ├── App.tsx                  # App entry point with RootNavigator
│   ├── app.config.ts            # Expo config (deep linking, scheme: alfred)
│   ├── package.json             # Dependencies
│   ├── .env.local               # Local dev environment (not in git)
│   └── src/
│       ├── api/                 # API client functions
│       │   ├── client.ts        # Axios client with base URL
│       │   ├── whatsapp.ts      # WhatsApp pairing code API
│       │   ├── gcal.ts          # Google Calendar OAuth API
│       │   ├── notifications.ts # Notification preferences API
│       │   └── onboarding.ts    # Onboarding status API
│       ├── components/          # Reusable UI components
│       │   ├── channels/        # Channel list, item, picker
│       │   ├── events/          # Event card, list, modals
│       │   ├── common/          # Button, Card, Modal, etc.
│       │   └── layout/          # Header, ConnectionStatus
│       ├── config/              # API configuration
│       ├── hooks/               # React Query hooks
│       │   └── useOnboardingStatus.ts  # WhatsApp/GCal status hooks
│       ├── navigation/
│       │   ├── RootNavigator.tsx    # Switches onboarding/main tabs
│       │   └── TopTabs.tsx          # Channels, Events, Settings tabs
│       ├── screens/
│       │   ├── ChannelsScreen.tsx
│       │   ├── EventsScreen.tsx
│       │   ├── SettingsScreen.tsx   # WhatsApp/GCal/Notifications
│       │   └── onboarding/          # Setup flow screens
│       │       ├── WelcomeScreen.tsx
│       │       ├── WhatsAppSetupScreen.tsx
│       │       ├── GoogleCalendarSetupScreen.tsx
│       │       └── NotificationSetupScreen.tsx
│       ├── theme/               # Colors, typography
│       └── types/               # TypeScript types
```

---

## Key Types

### Channel (database/channels.go)
```go
type Channel struct {
    ID         int64
    Type       ChannelType  // "sender" | "group"
    Identifier string       // WhatsApp JID
    Name       string
    CalendarID string       // Google Calendar ID
    Enabled    bool
    CreatedAt  time.Time
}
```

### CalendarEvent (database/events.go)
```go
type EventStatus string     // "pending" | "confirmed" | "synced" | "rejected" | "deleted"
type EventActionType string // "create" | "update" | "delete"

type CalendarEvent struct {
    ID            int64
    ChannelID     int64
    GoogleEventID *string         // Set after sync
    CalendarID    string
    Title         string
    Description   string
    StartTime     time.Time
    EndTime       *time.Time
    Location      string
    Status        EventStatus
    ActionType    EventActionType
    OriginalMsgID *int64
    LLMReasoning  string          // Claude's explanation
    Attendees     []Attendee
}
```

### EventAnalysis (claude/client.go)
```go
type EventAnalysis struct {
    HasEvent   bool
    Action     string   // "create" | "update" | "delete" | "none"
    Event      *EventData
    Reasoning  string
    Confidence float64
}
```

### FilteredMessage (whatsapp/handler.go)
```go
type FilteredMessage struct {
    SourceType string  // "sender" | "group"
    SourceID   int64   // Channel database ID
    SenderJID  string
    SenderName string
    Text       string
    IsGroup    bool
    Timestamp  time.Time
}
```

---

## Database Schema

### Tables
| Table | Purpose |
|-------|---------|
| `channels` | Tracked WhatsApp contacts/groups |
| `message_history` | Last N messages per channel (context for Claude) |
| `calendar_events` | Detected events with status lifecycle |
| `event_attendees` | Event participants |
| `user_notification_preferences` | Email/push/SMS notification settings |

### Event Status Lifecycle
```
pending → confirmed → synced
    ↓
rejected
```

---

## HTTP API Endpoints

### Health & System
| Path | Handler | Description |
|------|---------|-------------|
| GET `/health` | handleHealthCheck | Health check (DB, WhatsApp, GCal status) |

### Integration Status API
| Path | Handler | Description |
|------|---------|-------------|
| GET `/api/onboarding/status` | handleOnboardingStatus | Current integration status |
| GET `/api/onboarding/stream` | handleOnboardingSSE | SSE status updates for integrations |

### WhatsApp API
| Path | Handler | Description |
|------|---------|-------------|
| GET `/api/whatsapp/status` | handleWhatsAppStatus | Connection status |
| POST `/api/whatsapp/pair` | handleWhatsAppPair | Generate pairing code (phone number linking) |
| POST `/api/whatsapp/reconnect` | handleWhatsAppReconnect | Trigger reconnect |
| POST `/api/whatsapp/disconnect` | handleWhatsAppDisconnect | Disconnect WhatsApp |

### Channel API
| Path | Handler | Description |
|------|---------|-------------|
| GET `/api/channel` | handleListChannels | List tracked (filter: `?type=`) |
| POST `/api/channel` | handleCreateChannel | Add channel |
| PUT `/api/channel/{id}` | handleUpdateChannel | Update channel |
| DELETE `/api/channel/{id}` | handleDeleteChannel | Remove channel |
| GET `/api/discovery/channels` | handleDiscoverChannels | List available |

### Google Calendar API
| Path | Handler | Description |
|------|---------|-------------|
| GET `/api/gcal/status` | handleGCalStatus | Connection status |
| GET `/api/gcal/calendars` | handleGCalListCalendars | Available calendars |
| POST `/api/gcal/connect` | handleGCalConnect | Get OAuth URL (accepts custom redirect_uri) |
| POST `/api/gcal/callback` | handleGCalExchangeCode | Exchange OAuth code for token (mobile deep link) |
| GET `/oauth/callback` | handleOAuthCallback | OAuth callback (shows success page) |

### Events API
| Path | Handler | Description |
|------|---------|-------------|
| GET `/api/events` | handleListEvents | List events (filter: `?status=`, `?channel_id=`) |
| GET `/api/events/{id}` | handleGetEvent | Get with trigger message |
| PUT `/api/events/{id}` | handleUpdateEvent | Edit pending event |
| POST `/api/events/{id}/confirm` | handleConfirmEvent | Sync to Google Calendar |
| POST `/api/events/{id}/reject` | handleRejectEvent | Reject event |
| GET `/api/events/channel/{id}/history` | handleGetChannelHistory | Message context |

### Notification API
| Path | Handler | Description |
|------|---------|-------------|
| GET `/api/notifications/preferences` | handleGetNotificationPrefs | Get notification settings |
| PUT `/api/notifications/email` | handleUpdateEmailPrefs | Update email preferences |

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | - | Claude API key (required) |
| `GOOGLE_CREDENTIALS_FILE` | `./credentials.json` | OAuth credentials file path |
| `GOOGLE_CREDENTIALS_JSON` | - | OAuth credentials as JSON string (alternative to file) |
| `GOOGLE_TOKEN_FILE` | `./token.json` | OAuth token storage |
| `ALFRED_DB_PATH` | `./alfred.db` | SQLite database path |
| `ALFRED_WHATSAPP_DB_PATH` | `./whatsapp.db` | WhatsApp session database path |
| `PORT` | `8080` | HTTP server port (Railway sets this) |
| `ALFRED_HTTP_PORT` | `8080` | HTTP server port (fallback) |
| `ALFRED_DEBUG_ALL_MESSAGES` | `false` | Log all messages |
| `ALFRED_CLAUDE_MODEL` | `claude-sonnet-4-20250514` | Model for detection |
| `ALFRED_CLAUDE_TEMPERATURE` | `0.1` | Lower = more deterministic |
| `ALFRED_MESSAGE_HISTORY_SIZE` | `25` | Messages kept per channel |
| `ALFRED_RESEND_API_KEY` | - | Resend API key for email notifications |
| `ALFRED_EMAIL_FROM` | `Alfred <onboarding@resend.dev>` | Email sender address |

---

## Common Tasks

### Add New API Endpoint
1. Add route in `server/server.go` → `registerRoutes()`
2. Add handler in `server/handlers.go`
3. Add database function if needed

### Add Database Table
1. Add migration in `database/database.go` → `migrate()`
2. Create new file for CRUD (e.g., `database/newtable.go`)
3. Add types and functions

### Add Configuration
1. Add field to `Config` in `config/env.go`
2. Load in `LoadFromEnv()` using helpers

### Modify Event Detection
1. Update prompt in `claude/prompt.go`
2. Adjust types in `claude/client.go` if needed

### Add Calendar Features
1. Add to `gcal/events.go` for event operations
2. Add to `gcal/calendars.go` for calendar operations

---

## Key Functions by Package

### main.go
- `main()` - Bootstrap, initialize, start processing
- `waitForShutdown()` - Graceful shutdown on SIGINT/SIGTERM

### database/
- `New(dbPath)` - Open SQLite with migrations
- `CreateChannel()`, `GetChannelByIdentifier()`, `ListChannels()`
- `StoreMessage()`, `GetMessageHistory()`, `PruneMessages()`
- `CreatePendingEvent()`, `GetEventByID()`, `UpdateEventStatus()`
- `SetEventAttendees()`, `GetEventAttendees()`

### processor/
- `New()` - Create processor with dependencies
- `Start()` - Begin message processing goroutine
- `processMessage()` - Store → Claude → Create pending event

### claude/
- `NewClient()` - Create with API key, model, temperature
- `AnalyzeMessages()` - Send context to Claude, parse EventAnalysis

### gcal/
- `NewClient()` - Load OAuth config and token
- `CreateEvent()`, `UpdateEvent()`, `DeleteEvent()`
- `ListCalendars()`, `GetAuthURL()`, `GetAuthURLWithRedirect()`
- `ExchangeCode()`, `ExchangeCodeWithRedirect()` - Exchange OAuth code for token

### whatsapp/
- `NewClient()` - Create with message handler
- `HandleEvent()` - Dispatch WhatsApp events
- `GetDiscoverableChannels()` - List contacts and groups
- `PairWithPhone()` - Generate pairing code for phone-number linking

### sse/
- `NewState()` - Create onboarding state manager
- `Subscribe()`, `Unsubscribe()` - SSE connections
- `SetWhatsAppStatus()`, `SetGCalStatus()` - Broadcast updates

---

## Development Patterns

### Error Handling
```go
if err != nil {
    return fmt.Errorf("context: %w", err)
}
```

### HTTP Response
```go
respondJSON(w, http.StatusOK, data)
respondError(w, http.StatusBadRequest, "message")
```

### Database Query
```go
rows, err := d.Query(`SELECT ... FROM ... WHERE ...`, args...)
defer rows.Close()
for rows.Next() { ... }
```

### Time Parsing (handlers.go)
```go
parseEventTime(s string) // Handles RFC3339, ISO, local formats
```

---

## Files by Modification Frequency

| Task | Primary Files |
|------|---------------|
| API changes | `server/handlers.go`, `server/server.go` |
| Event detection | `claude/prompt.go`, `claude/client.go` |
| Mobile UI changes | `mobile/src/components/**`, `mobile/src/screens/**` |
| Onboarding flow | `mobile/src/screens/onboarding/**`, `mobile/src/navigation/RootNavigator.tsx` |
| Database schema | `database/database.go` |
| Message processing | `processor/processor.go` |
| Google Calendar | `gcal/events.go`, `gcal/client.go` |
| WhatsApp | `whatsapp/handler.go`, `whatsapp/client.go` |
| Configuration | `config/env.go` |
| Notifications | `notify/service.go`, `notify/resend.go` |
| Deployment | `Dockerfile`, `railway.toml`, `scripts/deploy.sh` |
| Development scripts | `scripts/dev.sh`, `scripts/deploy.sh` |
| Documentation | `docs/*.md`, `CLAUDE.md` |

---

## Railway Deployment

### Deployment Files
- `Dockerfile` - Multi-stage build with CGO for SQLite
- `railway.toml` - Railway configuration with health check

### Setup Steps
1. Install Railway CLI: `brew install railway`
2. Login: `railway login`
3. Initialize project: `railway init`
4. Link service: `railway link`
5. Create volume: `railway volume add --mount-path /data`
6. Set environment variables:
   ```bash
   railway variables set ANTHROPIC_API_KEY="sk-..."
   railway variables set ALFRED_DB_PATH="/data/alfred.db"
   railway variables set ALFRED_WHATSAPP_DB_PATH="/data/whatsapp.db"
   railway variables set GOOGLE_CREDENTIALS_FILE="/data/credentials.json"
   railway variables set GOOGLE_TOKEN_FILE="/data/token.json"
   railway variables set GOOGLE_CREDENTIALS_JSON='{"web":{...}}'  # Alternative to file
   ```
7. Deploy: `railway up`
8. Get domain: `railway domain`

### Persistent Storage
Railway volume mounted at `/data` stores:
- `alfred.db` - Application database
- `whatsapp.db` - WhatsApp session (preserves login)
- `token.json` - Google OAuth token

### Google OAuth Configuration
Add these redirect URIs to Google Cloud Console:
```
https://alfred-production-d2c9.up.railway.app/oauth/callback
alfred://oauth/callback
```
The `alfred://` URI is for mobile app deep link OAuth flow.

### Health Check
Railway uses `GET /health` endpoint which returns:
```json
{"status":"healthy","whatsapp":"connected|disconnected","gcal":"connected|disconnected"}
```

---

## Mobile App Development

### Quick Start
```bash
cd mobile
npm install                    # Install dependencies
npm run web                    # Start web preview (http://localhost:8081)
```

### Viewing Options
| Method | Command | Notes |
|--------|---------|-------|
| Web browser | `npm run web` | Opens at http://localhost:8081 |
| iOS Simulator | `npm run ios` | Requires Xcode |
| Android Emulator | `npm run android` | Requires Android Studio |
| iPhone (Expo Go) | `npm start` + scan QR | Phone + computer on same WiFi |

### Environment Configuration
The mobile app reads `EXPO_PUBLIC_API_BASE_URL` from environment:

**Local development** (`mobile/.env.local`):
```
EXPO_PUBLIC_API_BASE_URL=http://localhost:8080
```

**Testing on physical device** - use your computer's IP:
```
EXPO_PUBLIC_API_BASE_URL=http://192.168.x.x:8080
```

Find your IP: `ipconfig getifaddr en0`

### Key Mobile Files
| Task | Primary Files |
|------|---------------|
| API configuration | `src/config/api.ts`, `src/api/client.ts` |
| Channel list/sorting | `src/components/channels/ChannelList.tsx` |
| Event management | `src/components/events/EventCard.tsx`, `EventList.tsx` |
| Navigation | `src/navigation/RootNavigator.tsx`, `TopTabs.tsx` |
| Onboarding | `src/screens/onboarding/*`, `src/hooks/useOnboardingStatus.ts` |
| Settings | `src/screens/SettingsScreen.tsx` |
| API hooks | `src/hooks/useChannels.ts`, `useEvents.ts`, `useOnboardingStatus.ts` |
