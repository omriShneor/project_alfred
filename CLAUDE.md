# Project Alfred

Multi-source calendar assistant using Claude AI to detect events from WhatsApp/Gmail, create pending events for review, and sync to Google Calendar.

## Quick Start

```bash
# Start backend (port 8080)
go run main.go

# Start mobile app (port 8081)
cd mobile && npm run web
```

**URLs:**
- Backend API: `http://localhost:8080` (no web UI)
- Mobile App: `http://localhost:8081` (all UI)
- Production: `https://alfred-production-d2c9.up.railway.app`

**Required Environment:**
```bash
export ANTHROPIC_API_KEY="sk-..."           # Required
# Place credentials.json in project root OR set GOOGLE_CREDENTIALS_JSON
```

---

## Architecture

### Data Flow
```
WhatsApp Message / Gmail Email
    ↓
Handler filters by tracked channels/sources
    ↓
Processor stores message, calls Claude
    ↓
Claude analyzes for events (create/update/delete)
    ↓
Database creates pending event
    ↓
User reviews in Mobile App
    ↓
Confirm → Sync to Google Calendar
```

### Components
| Component | Port | Purpose |
|-----------|------|---------|
| Go Backend | 8080 | API, WhatsApp connection, Claude analysis, Google Calendar sync |
| Mobile App | 8081 | All UI: onboarding, event review, settings |

**WhatsApp:** Uses pairing codes (not QR) - enter phone number, get 8-digit code for WhatsApp Linked Devices.

**Google OAuth:** Uses deep link (`alfred://oauth/callback`) - app opens browser, captures redirect.

---

## Common Tasks

### Add API Endpoint
1. Route: `internal/server/server.go` → `registerRoutes()`
2. Handler: `internal/server/handlers.go` (or domain-specific handler file)
3. Database: Add function in `internal/database/` if needed

### Add Database Table
1. Migration: `internal/database/database.go` → `migrate()`
2. CRUD: Create `internal/database/newtable.go` with types and functions

### Modify Event Detection
1. Prompt: `internal/claude/prompt.go` (system prompt)
2. Types: `internal/claude/client.go` (EventAnalysis, EventData)

### Add Mobile Screen
1. Screen: `mobile/src/screens/NewScreen.tsx`
2. Navigation: `mobile/src/navigation/` (add to navigator)
3. API hook: `mobile/src/hooks/useNewFeature.ts`

### Add Notification Channel
1. Notifier: `internal/notify/new_notifier.go` implementing `Notifier` interface
2. Register: `internal/notify/service.go`

### Add Configuration
1. Field: `internal/config/env.go` → `Config` struct
2. Load: `LoadFromEnv()` with helper functions

---

## API Reference

### Health & System
| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (DB, WhatsApp, GCal status) |

### Onboarding & App Status
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/onboarding/status` | Integration status during setup |
| GET | `/api/onboarding/stream` | SSE stream for real-time status |
| POST | `/api/onboarding/complete` | Mark onboarding complete |
| POST | `/api/onboarding/reset` | Reset onboarding (testing) |
| GET | `/api/app/status` | App status (onboarding_complete, integrations) |

### WhatsApp
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/whatsapp/status` | Connection status |
| POST | `/api/whatsapp/pair` | Generate pairing code (body: `{phone_number}`) |
| POST | `/api/whatsapp/reconnect` | Trigger reconnect |
| POST | `/api/whatsapp/disconnect` | Disconnect WhatsApp |

### Channels (WhatsApp Sources)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/channel` | List tracked channels (`?type=sender\|group`) |
| POST | `/api/channel` | Create channel |
| PUT | `/api/channel/{id}` | Update channel |
| DELETE | `/api/channel/{id}` | Delete channel |
| GET | `/api/discovery/channels` | List available (untracked) channels |

### Google Calendar
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/gcal/status` | Connection status |
| GET | `/api/gcal/calendars` | List available calendars |
| GET | `/api/gcal/events/today` | Today's calendar events |
| POST | `/api/gcal/connect` | Get OAuth URL (`?redirect_uri=`) |
| POST | `/api/gcal/callback` | Exchange OAuth code for token |
| POST | `/api/gcal/disconnect` | Disconnect Google account |
| GET | `/oauth/callback` | OAuth callback (browser redirect) |

### Events
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/events` | List events (`?status=pending\|confirmed\|synced\|rejected`, `?channel_id=`) |
| GET | `/api/events/today` | Today's merged events (Alfred + external) |
| GET | `/api/events/{id}` | Get event with trigger message |
| PUT | `/api/events/{id}` | Update pending event |
| POST | `/api/events/{id}/confirm` | Confirm and sync to Google Calendar |
| POST | `/api/events/{id}/reject` | Reject event |
| GET | `/api/events/channel/{channelId}/history` | Message context for channel |

### Notifications
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/notifications/preferences` | Get notification settings |
| PUT | `/api/notifications/email` | Update email preferences |
| POST | `/api/notifications/push/register` | Register Expo push token |
| PUT | `/api/notifications/push` | Update push preferences |

### Gmail
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/gmail/status` | Connection status |
| GET | `/api/gmail/settings` | Gmail settings |
| PUT | `/api/gmail/settings` | Update Gmail settings |

### Gmail Discovery
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/gmail/discover/categories` | List Gmail categories |
| GET | `/api/gmail/discover/senders` | List frequent senders |
| GET | `/api/gmail/discover/domains` | List frequent domains |

### Email Sources (Gmail Tracking)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/gmail/sources` | List tracked email sources |
| POST | `/api/gmail/sources` | Create email source |
| GET | `/api/gmail/sources/{id}` | Get email source |
| PUT | `/api/gmail/sources/{id}` | Update email source |
| DELETE | `/api/gmail/sources/{id}` | Delete email source |

### Features (Legacy)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/features` | Get feature settings |
| PUT | `/api/features/smart-calendar` | Update Smart Calendar settings |
| GET | `/api/features/smart-calendar/status` | Smart Calendar integration status |

---

## Database Schema

### Tables
| Table | Purpose |
|-------|---------|
| `channels` | Tracked WhatsApp contacts/groups |
| `message_history` | Last N messages per channel (Claude context) |
| `calendar_events` | Detected events with status lifecycle |
| `event_attendees` | Event participants |
| `user_notification_preferences` | Email/push settings (single row, id=1) |
| `gmail_settings` | Gmail integration settings (single row, id=1) |
| `email_sources` | Tracked email sources (categories, senders, domains) |
| `processed_emails` | Processed email IDs to avoid duplicates |
| `feature_settings` | App/feature settings, onboarding state (single row, id=1) |

### Event Status Lifecycle
```
pending → confirmed → synced
    ↓
rejected
```

### Key Types

```go
// CalendarEvent (database/events.go)
type CalendarEvent struct {
    ID, ChannelID     int64
    GoogleEventID     *string         // Set after sync
    CalendarID        string          // "primary" or calendar ID
    Title, Description, Location string
    StartTime         time.Time
    EndTime           *time.Time
    Status            EventStatus     // pending|confirmed|synced|rejected|deleted
    ActionType        EventActionType // create|update|delete
    OriginalMsgID     *int64
    LLMReasoning      string
    Attendees         []Attendee
}

// FeatureSettings (database/features.go)
type FeatureSettings struct {
    SmartCalendarEnabled, SmartCalendarSetupComplete bool
    WhatsAppInputEnabled, EmailInputEnabled, SMSInputEnabled bool
    AlfredCalendarEnabled, GoogleCalendarEnabled, OutlookCalendarEnabled bool
}

// AppStatus (database/features.go)
type AppStatus struct {
    OnboardingComplete bool
    WhatsAppEnabled, GmailEnabled, GoogleCalEnabled bool
}

// EventAnalysis (claude/client.go)
type EventAnalysis struct {
    HasEvent   bool
    Action     string   // create|update|delete|none
    Event      *EventData
    Reasoning  string
    Confidence float64
}

// FilteredMessage (whatsapp/handler.go)
type FilteredMessage struct {
    SourceType string  // sender|group
    SourceID   int64
    SenderJID, SenderName, Text string
    IsGroup    bool
    Timestamp  time.Time
}
```

---

## Project Structure

### Backend (Go)
| Directory | Key Files | Purpose |
|-----------|-----------|---------|
| `internal/config/` | `env.go` | Environment configuration loading |
| `internal/database/` | `database.go`, `channels.go`, `events.go`, `features.go`, `messages.go`, `notifications.go`, `gmail.go`, `email_sources.go`, `attendees.go` | SQLite data layer |
| `internal/server/` | `server.go`, `handlers.go`, `gmail_handlers.go`, `features_handlers.go` | HTTP API |
| `internal/claude/` | `client.go`, `prompt.go` | Claude AI event detection |
| `internal/processor/` | `processor.go`, `email_processor.go`, `history.go` | Message processing pipeline |
| `internal/whatsapp/` | `client.go`, `handler.go`, `groups.go`, `qr.go` | WhatsApp connection |
| `internal/gcal/` | `client.go`, `auth.go`, `events.go`, `calendars.go` | Google Calendar integration |
| `internal/gmail/` | `client.go`, `worker.go`, `scanner.go`, `discovery.go`, `parser.go` | Gmail integration |
| `internal/notify/` | `service.go`, `notifier.go`, `resend.go`, `expo_push.go` | Notifications (email, push) |
| `internal/onboarding/` | `onboarding.go`, `clients.go` | Setup orchestration |
| `internal/sse/` | `state.go` | Onboarding SSE state |

### Mobile (React Native/Expo)
| Directory | Key Files | Purpose |
|-----------|-----------|---------|
| `mobile/src/api/` | `client.ts`, `whatsapp.ts`, `gcal.ts`, `events.ts`, `channels.ts`, `gmail.ts`, `notifications.ts`, `app.ts` | API clients |
| `mobile/src/hooks/` | `useAppStatus.ts`, `useEvents.ts`, `useChannels.ts`, `usePushNotifications.ts`, `useOnboardingStatus.ts` | React Query hooks |
| `mobile/src/screens/` | `HomeScreen.tsx`, `SettingsScreen.tsx`, `PreferencesScreen.tsx`, `WhatsAppPreferencesScreen.tsx`, `GmailPreferencesScreen.tsx` | Main screens |
| `mobile/src/screens/onboarding/` | `WelcomeScreen.tsx`, `InputSelectionScreen.tsx`, `ConnectionScreen.tsx` | Onboarding flow |
| `mobile/src/components/` | `events/`, `channels/`, `common/`, `home/` | UI components |
| `mobile/src/navigation/` | `RootNavigator.tsx`, `MainNavigator.tsx`, `OnboardingNavigator.tsx` | Navigation |
| `mobile/src/theme/` | `colors.ts`, `typography.ts` | Styling |
| `mobile/src/types/` | `event.ts`, `channel.ts`, `app.ts`, `features.ts` | TypeScript types |

---

## Environment Variables

### Required
| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Claude API key for event detection |

### Optional
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` / `ALFRED_HTTP_PORT` | `8080` | HTTP server port |
| `GOOGLE_CREDENTIALS_FILE` | `./credentials.json` | OAuth credentials path |
| `GOOGLE_CREDENTIALS_JSON` | - | OAuth credentials as JSON (alternative) |
| `GOOGLE_TOKEN_FILE` | `./token.json` | OAuth token storage |
| `ALFRED_DB_PATH` | `./alfred.db` | SQLite database path |
| `ALFRED_WHATSAPP_DB_PATH` | `./whatsapp.db` | WhatsApp session DB |
| `ALFRED_CLAUDE_MODEL` | `claude-sonnet-4-20250514` | Claude model |
| `ALFRED_CLAUDE_TEMPERATURE` | `0.1` | Model temperature (0-1) |
| `ALFRED_MESSAGE_HISTORY_SIZE` | `25` | Messages per channel for context |
| `ALFRED_RESEND_API_KEY` | - | Resend API for email notifications |
| `ALFRED_EMAIL_FROM` | `Alfred <onboarding@resend.dev>` | Email sender |
| `ALFRED_GMAIL_POLL_INTERVAL` | `5` | Minutes between email checks |
| `ALFRED_GMAIL_MAX_EMAILS` | `10` | Max emails per poll |
| `ALFRED_DEBUG_ALL_MESSAGES` | `false` | Log all WhatsApp messages |

---

## Development Patterns

### Go Error Handling
```go
if err != nil {
    return fmt.Errorf("context: %w", err)
}
```

### HTTP Responses
```go
respondJSON(w, http.StatusOK, data)
respondError(w, http.StatusBadRequest, "message")
```

### Database Query
```go
rows, err := d.Query(`SELECT ... WHERE ...`, args...)
defer rows.Close()
for rows.Next() { ... }
```

### React Query Hook
```typescript
export function useFeature() {
  return useQuery({
    queryKey: ['feature'],
    queryFn: getFeature,
  });
}
```

### React Query Mutation
```typescript
export function useUpdateFeature() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: updateFeature,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['feature'] }),
  });
}
```

---

## Deployment (Railway)

### Setup
```bash
railway login
railway init
railway volume add --mount-path /data   # Persistent storage
railway up                               # Deploy
railway domain                           # Get URL
```

### Environment Variables (Production)
```bash
railway variables set ANTHROPIC_API_KEY="sk-..."
railway variables set ALFRED_DB_PATH="/data/alfred.db"
railway variables set ALFRED_WHATSAPP_DB_PATH="/data/whatsapp.db"
railway variables set GOOGLE_TOKEN_FILE="/data/token.json"
railway variables set GOOGLE_CREDENTIALS_JSON='{"web":{...}}'
```

### Google OAuth URIs
Add to Google Cloud Console:
```
https://alfred-production-d2c9.up.railway.app/oauth/callback
alfred://oauth/callback
```

### Persistent Storage
Volume at `/data` stores: `alfred.db`, `whatsapp.db`, `token.json`

---

## Mobile Development

### Commands
| Command | Purpose |
|---------|---------|
| `npm run web` | Web preview at localhost:8081 |
| `npm run ios` | iOS Simulator |
| `npm run android` | Android Emulator |
| `npm start` | Expo Go (scan QR on phone) |

### Environment
Create `mobile/.env.local`:
```
EXPO_PUBLIC_API_BASE_URL=http://localhost:8080
```

For physical device, use your computer's IP:
```
EXPO_PUBLIC_API_BASE_URL=http://192.168.x.x:8080
```

Find IP: `ipconfig getifaddr en0`

---

## Files by Modification Frequency

### High (API, UI changes)
| Task | Files |
|------|-------|
| API endpoints | `internal/server/handlers.go`, `internal/server/server.go` |
| Event detection | `internal/claude/prompt.go`, `internal/claude/client.go` |
| Mobile screens | `mobile/src/screens/**`, `mobile/src/components/**` |
| Navigation | `mobile/src/navigation/RootNavigator.tsx` |

### Medium (Feature changes)
| Task | Files |
|------|-------|
| Database schema | `internal/database/database.go` |
| WhatsApp processing | `internal/processor/processor.go`, `internal/whatsapp/handler.go` |
| Gmail processing | `internal/processor/email_processor.go`, `internal/gmail/worker.go` |
| Google Calendar | `internal/gcal/events.go` |
| Notifications | `internal/notify/service.go` |
| Mobile API hooks | `mobile/src/hooks/**` |

### Low (Configuration, deployment)
| Task | Files |
|------|-------|
| Configuration | `internal/config/env.go` |
| Deployment | `Dockerfile`, `railway.toml` |
