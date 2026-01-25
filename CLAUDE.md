# Project Alfred

WhatsApp-to-Google Calendar assistant that uses Claude AI to detect calendar events from conversations, creates pending events for review, and syncs confirmed events to Google Calendar.

## Quick Reference

### Key Commands
```bash
go run main.go              # Start the application
go build -o alfred && ./alfred  # Build and run
```

### Railway Deployment
```bash
railway login               # Login to Railway
railway up                  # Deploy to Railway
railway logs                # View deployment logs
railway domain              # Get/create public URL
```

### Important URLs (default port 8080)
- `http://localhost:8080/settings` - Integrations (WhatsApp + Google Calendar) and notification preferences
- `http://localhost:8080/admin` - Channel management
- `http://localhost:8080/events` - Review and confirm detected events
- `http://localhost:8080/health` - Health check endpoint
- `/` redirects to `/settings`

### Production URL
- https://alfred-production-d2c9.up.railway.app

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
User reviews at /events
    ↓
gcal/events.go (sync to Google Calendar)
```

---

## Project Structure

```
project_alfred/
├── main.go                      # Entry point, initialization, shutdown
├── Dockerfile                   # Multi-stage Docker build for Railway
├── railway.toml                 # Railway deployment configuration
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
│   │   ├── handlers.go          # All HTTP handlers
│   │   └── static/
│   │       ├── admin.html       # Channel management UI
│   │       ├── events.html      # Event review UI
│   │       └── settings.html    # Integrations + notification settings UI
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

### UI Pages
| Path | Handler | Description |
|------|---------|-------------|
| GET `/` | handleRootRedirect | Redirects to settings |
| GET `/admin` | handleAdminPage | Channel management |
| GET `/events` | handleEventsPage | Event review |
| GET `/settings` | handleSettingsPage | Integrations + notification settings |

### Integration Status API
| Path | Handler | Description |
|------|---------|-------------|
| GET `/api/onboarding/status` | handleOnboardingStatus | Current integration status |
| GET `/api/onboarding/stream` | handleOnboardingSSE | SSE status updates for integrations |

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
| POST `/api/gcal/connect` | handleGCalConnect | Get OAuth URL |

### Events API
| Path | Handler | Description |
|------|---------|-------------|
| GET `/api/events` | handleListEvents | List events (filter: `?status=`, `?channel_id=`) |
| GET `/api/events/{id}` | handleGetEvent | Get with trigger message |
| PUT `/api/events/{id}` | handleUpdateEvent | Edit pending event |
| POST `/api/events/{id}/confirm` | handleConfirmEvent | Sync to Google Calendar |
| POST `/api/events/{id}/reject` | handleRejectEvent | Reject event |
| GET `/api/events/channel/{id}/history` | handleGetChannelHistory | Message context |

### WhatsApp API
| Path | Handler | Description |
|------|---------|-------------|
| POST `/api/whatsapp/reconnect` | handleWhatsAppReconnect | Trigger new QR |

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
| `ALFRED_DEV_MODE` | `false` | Hot reload static files |
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
- `ListCalendars()`, `GetAuthURL()`

### whatsapp/
- `NewClient()` - Create with message handler
- `HandleEvent()` - Dispatch WhatsApp events
- `GetDiscoverableChannels()` - List contacts and groups

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
| UI changes | `server/static/*.html` |
| Database schema | `database/database.go` |
| Message processing | `processor/processor.go` |
| Google Calendar | `gcal/events.go` |
| WhatsApp | `whatsapp/handler.go` |
| Configuration | `config/env.go` |
| Notifications | `notify/service.go`, `notify/resend.go` |
| Deployment | `Dockerfile`, `railway.toml` |

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

### Google OAuth for Production
Add Railway callback URL to Google Cloud Console:
```
https://alfred-production-d2c9.up.railway.app/oauth/callback
```

### Health Check
Railway uses `GET /health` endpoint which returns:
```json
{"status":"healthy","whatsapp":"connected|disconnected","gcal":"connected|disconnected"}
```
