# Project Alfred

Multi-source calendar assistant using Claude AI to detect events and reminders from WhatsApp/Telegram/Gmail, create pending items for review, and sync to Google Calendar.

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
export ANTHROPIC_API_KEY="sk-..."           # Required for AI analysis
# Place credentials.json in project root OR set GOOGLE_CREDENTIALS_JSON
```

**Authentication:**
- All API endpoints require authentication (except `/health` and `/api/auth/*`)
- Users log in with Google OAuth
- Development mode: Set `ALFRED_DEV_MODE=true` to bypass auth (auto-injects user ID 1)

---

## Architecture

### Data Flow
```
User Login (Google OAuth - Profile scopes)
    ↓
Session Token Created (30-day bearer token)
    ↓
Per-User Clients Initialized (WhatsApp, Telegram, Gmail, GCal)
    ↓
WhatsApp/Telegram Message OR Gmail Email
    ↓
Handler filters by user's tracked channels/sources
    ↓
Agent Analyzers Run (Event + Reminder detection in parallel)
    ↓
Claude with Tools → Multi-turn extraction
    ↓
Pending Events/Reminders Created (user_id scoped)
    ↓
User Reviews in Mobile App (authenticated session)
    ↓
Confirm → Sync to User's Google Calendar
```

### Components
| Component | Port | Purpose |
|-----------|------|---------|
| Go Backend | 8080 | API, Authentication, WhatsApp/Telegram connection, Claude analysis, Google Calendar sync |
| Mobile App | 8081 | All UI: login, onboarding, event/reminder review, settings |

**Authentication:** Session-based with bearer tokens (30-day expiry). Per-user client management for all integrations.

**WhatsApp:** Uses pairing codes (not QR) - enter phone number, get 8-digit code for WhatsApp Linked Devices. Per-user session files.

**Telegram:** Uses phone verification - enter phone number, receive code via Telegram, verify to link. Per-user session files.

**Google OAuth:** Incremental authorization with scope separation:
- Phase 1: Login with profile scopes → session token
- Phase 2: Add Gmail scopes (when user enables email scanning)
- Phase 3: Add Calendar scopes (when user confirms first event)

OAuth flow uses deep link (`alfred://oauth/callback`) - app opens browser, captures redirect via `/api/auth/callback`.

**Gmail:** Fetches full email threads (up to 10 messages) for context when analyzing emails. Thread history is passed to Claude for better event detection.

**Agent Framework:** Tool-calling architecture where Claude can use tools to extract structured data (calendar events, date/time parsing, location lookup, attendee resolution, reminders).

---

## Multi-User Architecture

Project Alfred supports multiple users with complete data isolation:

### User Isolation
- Each user has separate: channels, events, reminders, settings, OAuth tokens
- All database tables include `user_id` foreign key
- Database queries automatically filter by authenticated user
- Per-user WhatsApp/Telegram clients with isolated session files
- Per-user Gmail workers and GCal clients

### Client Management
The `ClientManager` ([internal/clients/manager.go](internal/clients/manager.go)) handles per-user service instances:
- Creates clients on-demand for authenticated users
- Maintains separate WhatsApp/Telegram/Gmail/GCal clients per user
- Uses per-user database files: `whatsapp.db.user_2`, `telegram.db.user_2`
- User 1 uses legacy paths (`whatsapp.db`, `telegram.db`) for backward compatibility

### Authentication Flow
1. User logs in with Google OAuth (profile scopes only)
2. Backend creates session token (30-day expiry, stored in `user_sessions`)
3. Mobile app stores token in secure storage
4. Token included in all API requests: `Authorization: Bearer <token>`
5. Server validates token via middleware and loads user context
6. All operations automatically scoped to authenticated user

### Service Lifecycle
- Services start when user first accesses them (lazy initialization)
- WhatsApp/Telegram maintain persistent connections per user
- Gmail workers poll per-user mailboxes independently (1-minute interval)
- Services automatically restart on reconnection
- Session data persists in database (encrypted OAuth tokens) and per-user files

---

## Agent Framework

Project Alfred uses a tool-calling agent architecture for intelligent extraction:

### Architecture
```
Message/Email → Agent Analyzers
                    ↓
        Claude API with Tools
                    ↓
    Multi-turn Conversation
                    ↓
Tool Calls (calendar, datetime, location, attendees, reminder)
                    ↓
        Structured Output
                    ↓
    Event/Reminder Created
```

### Analyzers
- **EventAnalyzer** ([internal/agent/event/](internal/agent/event/)): Detects calendar events (create/update/delete)
- **ReminderAnalyzer** ([internal/agent/reminder/](internal/agent/reminder/)): Detects reminders/todos (create/update/delete)
- Both run in parallel on incoming messages for comprehensive detection

### Tools
| Tool | Purpose | Implementation |
|------|---------|----------------|
| `search_existing_events` | Find pending/synced events in Alfred | [internal/agent/tools/calendar.go](internal/agent/tools/calendar.go) |
| `get_current_datetime` | Get current date/time in user's timezone | [internal/agent/tools/datetime.go](internal/agent/tools/datetime.go) |
| `parse_relative_time` | Convert "tomorrow", "next week" to dates | [internal/agent/tools/datetime.go](internal/agent/tools/datetime.go) |
| `lookup_location` | Geocode locations for event details | [internal/agent/tools/location.go](internal/agent/tools/location.go) |
| `lookup_attendees` | Resolve contact names to email addresses | [internal/agent/tools/attendees.go](internal/agent/tools/attendees.go) |
| `search_existing_reminders` | Find pending/synced reminders | [internal/agent/tools/reminder.go](internal/agent/tools/reminder.go) |

### Benefits
- **Context-aware extraction**: Claude can search existing events/reminders for updates
- **Complex temporal references**: Handles "next Tuesday", "in 2 weeks", etc.
- **Update/delete intents**: Detects when messages modify or cancel existing items
- **Attendee resolution**: Converts "invite John" to actual email addresses
- **Transparent reasoning**: LLM explains why it detected or didn't detect items

---

## Reminders Feature

Alfred detects and manages reminders/todos from messages and emails:

### Detection
- Runs independently from event detection (parallel analysis)
- Extracts: title, description, due date, priority (low/normal/high)
- Optional reminder_time for notifications
- Links to original message/email for context

### Status Lifecycle
```
pending → confirmed → synced → completed
    ↓                              ↓
rejected                      dismissed
```

### Statuses
- **pending**: Awaiting user review
- **confirmed**: Approved by user, ready to sync
- **synced**: Created in Google Calendar/Tasks
- **rejected**: User rejected the reminder
- **completed**: User marked as done
- **dismissed**: User dismissed without completing

### Priorities
- **low**: Nice to have, no urgency
- **normal**: Standard reminder (default)
- **high**: Important/urgent task

### Sync Options
- **Google Calendar**: Synced as all-day events with reminders
- **Google Tasks**: Future enhancement (not yet implemented)
- **Local reminders**: App-based notifications (future enhancement)

### Use Cases
- Follow-up tasks: "Remind me to follow up with client next week"
- Shopping lists: "Add milk to shopping list"
- Personal todos: "Don't forget to call mom tomorrow"
- Work reminders: "Prepare presentation by Friday"

---

## Common Tasks

### Working with Authentication
All API endpoints (except `/health` and `/api/auth/*`) require authentication:

```go
// Get authenticated user from context
user := auth.GetUserFromContext(r.Context())
if user == nil {
    respondError(w, http.StatusUnauthorized, "authentication required")
    return
}
userID := user.ID

// Or use helper
userID := getUserID(r)
if userID == 0 {
    respondError(w, http.StatusUnauthorized, "authentication required")
    return
}

// Access user's services
services, err := s.userServiceManager.GetServicesForUser(userID)
if err != nil {
    respondError(w, http.StatusServiceUnavailable, "services not available")
    return
}
wa := services.WhatsApp  // User's WhatsApp client
```

**Dev mode:** Set `ALFRED_DEV_MODE=true` to bypass auth (auto-injects user ID 1)

### Add API Endpoint
1. Route: [internal/server/server.go](internal/server/server.go) → `registerRoutes()`
2. Handler: [internal/server/handlers.go](internal/server/handlers.go) (or domain-specific handler file)
3. **Authentication**: Wrap with `s.requireAuth(handler)` middleware
4. **User context**: Access via `getUserID(r)` or `auth.GetUserFromContext(r.Context())`
5. **Per-user operations**: Use `s.userServiceManager.GetServicesForUser(userID)`
6. Database: Add function in `internal/database/` if needed

### Add Database Table
1. Migration: Create `internal/database/migrations/NNN_name.go` with `Register()` call
2. **User isolation**: Include `user_id INTEGER NOT NULL` with `FOREIGN KEY(user_id) REFERENCES users(id)`
3. **Index**: Add `CREATE INDEX idx_{table}_user ON {table}(user_id)` for performance
4. CRUD: Create `internal/database/newtable.go` with types and functions
5. **Query pattern**: Always filter by user_id: `WHERE user_id = ?`

### Modify Event/Reminder Detection
1. **Agent tools**: [internal/agent/tools/](internal/agent/tools/) - add or modify tool implementations
2. **Analyzer logic**: [internal/agent/event/](internal/agent/event/) or [internal/agent/reminder/](internal/agent/reminder/)
3. **Tool registration**: Register tools with agent in analyzer constructor
4. **Types**: [internal/agent/types.go](internal/agent/types.go) - define input/output types
5. **Legacy prompt** (deprecated): [internal/claude/prompt.go](internal/claude/prompt.go) - old non-agent approach

### Add Agent Analyzer
1. **Define tool**:
```go
tool := agent.Tool{
    Name: "get_current_time",
    Description: "Get current date and time",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "timezone": map[string]interface{}{
                "type": "string",
                "description": "IANA timezone (e.g., America/New_York)",
            },
        },
    },
}
```

2. **Implement handler**:
```go
handler := func(ctx context.Context, input map[string]interface{}) (string, error) {
    tz := input["timezone"].(string)
    // ... implementation
    return result, nil
}
```

3. **Register with agent**:
```go
agent.MustRegisterTool(tool, handler)
```

4. **Multi-turn conversations**: Agent can make multiple tool calls before final response

### Add Mobile Screen
1. Screen: `mobile/src/screens/NewScreen.tsx`
2. Navigation: `mobile/src/navigation/` (add to navigator)
3. API hook: `mobile/src/hooks/useNewFeature.ts`
4. **Authentication**: Use `AuthContext` to check login state

### Add Notification Channel
1. Notifier: `internal/notify/new_notifier.go` implementing `Notifier` interface
2. Register: `internal/notify/service.go`

### Add Configuration
1. Field: [internal/config/env.go](internal/config/env.go) → `Config` struct
2. Load: `LoadFromEnv()` with helper functions

### Add Message Source
The project uses unified source types in [internal/source/source.go](internal/source/source.go).
1. Source constant: [internal/source/source.go](internal/source/source.go) → add `SourceType`
2. Client: Create `internal/newsource/client.go`, `handler.go`
3. Handlers: Add `internal/server/newsource_handlers.go`
4. Routes: [internal/server/server.go](internal/server/server.go) → `registerRoutes()`
5. **Per-user client**: Integrate with `ClientManager` for multi-user support
6. Mobile API: `mobile/src/api/newsource.ts`
7. Mobile Hook: `mobile/src/hooks/useNewSource.ts`

---

## API Reference

**Note:** All endpoints require authentication via `Authorization: Bearer <token>` header except where noted as "No" in Auth Required column.

### Health & System
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | `/health` | No | Health check (DB, WhatsApp, GCal status) |

### Authentication
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| POST | `/api/auth/google/login` | No | Get OAuth URL for login (profile scopes only). Body: `{ "redirect_uri": "alfred://oauth/callback" }` (optional) |
| POST | `/api/auth/google/callback` | No | Exchange OAuth code for session token. Body: `{ "code": "...", "redirect_uri": "..." }`. Returns: `{ "session_token": "...", "user": {...} }` |
| GET | `/api/auth/me` | Yes | Get current authenticated user info. Returns: `{ "id": 1, "email": "...", "name": "...", "avatar_url": "..." }` |
| POST | `/api/auth/google/logout` | No | Invalidate session token |
| POST | `/api/auth/google/add-scopes` | Yes | Request additional scopes (Gmail/Calendar). Body: `{ "scopes": ["gmail" \| "calendar"], "redirect_uri": "..." }`. Returns: `{ "auth_url": "https://..." }` |
| POST | `/api/auth/google/add-scopes/callback` | Yes | Exchange code for incremental scopes. Body: `{ "code": "...", "scopes": ["gmail" \| "calendar"], "redirect_uri": "..." }` |
| GET | `/api/auth/callback` | No | OAuth callback handler (browser redirect to deep link) |

**OAuth Flow:**
1. Login: `/api/auth/google/login` → Google OAuth (profile scopes) → `/api/auth/callback` → deep link → mobile exchanges code
2. Add Gmail: `/api/auth/google/add-scopes` with `scopes: ["gmail"]` → Google OAuth with `include_granted_scopes=true`
3. Add Calendar: `/api/auth/google/add-scopes` with `scopes: ["calendar"]` → Google OAuth with `include_granted_scopes=true`

### Onboarding & App Status
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | `/api/onboarding/status` | No | Integration status during setup |
| GET | `/api/onboarding/stream` | No | SSE stream for real-time status |
| POST | `/api/onboarding/complete` | Yes | Mark onboarding complete |
| POST | `/api/onboarding/reset` | No (dev mode only) | Reset onboarding (testing only, requires `ALFRED_DEV_MODE=true` in production) |
| GET | `/api/app/status` | No/Optional | App status (onboarding_complete, integrations). Works for both authenticated and anonymous users. |

### WhatsApp
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | `/api/whatsapp/status` | Yes | Connection status for current user |
| POST | `/api/whatsapp/pair` | Yes | Generate pairing code. Body: `{ "phone_number": "+1234567890" }` |
| POST | `/api/whatsapp/reconnect` | Yes | Trigger reconnect for user's WhatsApp |
| POST | `/api/whatsapp/disconnect` | Yes | Disconnect user's WhatsApp |
| GET | `/api/whatsapp/top-contacts` | Yes | Get top contacts from user's history |
| POST | `/api/whatsapp/sources/custom` | Yes | Add custom source by phone number |

### Channels (WhatsApp/Telegram Sources)
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | `/api/channel` | Yes | List user's tracked channels. Query: `?type=sender\|group`, `?source_type=whatsapp\|telegram` |
| POST | `/api/channel` | Yes | Create channel for user |
| PUT | `/api/channel/{id}` | Yes | Update user's channel |
| DELETE | `/api/channel/{id}` | Yes | Delete user's channel |
| GET | `/api/discovery/channels` | Yes | List available (untracked) WhatsApp channels for user |

### Telegram
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | `/api/telegram/status` | Yes | Connection status for current user |
| POST | `/api/telegram/send-code` | Yes | Send verification code. Body: `{ "phone_number": "+1234567890" }` |
| POST | `/api/telegram/verify-code` | Yes | Verify code. Body: `{ "phone_number": "+1234567890", "code": "12345" }` |
| POST | `/api/telegram/disconnect` | Yes | Disconnect user's Telegram |
| POST | `/api/telegram/reconnect` | Yes | Reconnect user's Telegram |
| GET | `/api/telegram/discovery/channels` | Yes | List available Telegram chats for user |
| GET | `/api/telegram/channel` | Yes | List user's tracked Telegram channels |
| POST | `/api/telegram/channel` | Yes | Create Telegram channel. Body: `{ "type": "sender\|group", "identifier": "...", "name": "..." }` |
| PUT | `/api/telegram/channel/{id}` | Yes | Update user's Telegram channel |
| DELETE | `/api/telegram/channel/{id}` | Yes | Delete user's Telegram channel |
| GET | `/api/telegram/top-contacts` | Yes | Get top Telegram contacts for user |
| POST | `/api/telegram/sources/custom` | Yes | Add custom source by username |

### Google Calendar
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | `/api/gcal/status` | Yes | Connection status and scopes for current user |
| GET | `/api/gcal/calendars` | Yes | List user's available calendars |
| GET | `/api/gcal/events/today` | Yes | Today's calendar events from user's Google Calendar |
| POST | `/api/gcal/disconnect` | Yes | Disconnect user's Google Calendar |
| GET | `/api/gcal/settings` | Yes | Get user's sync settings |
| PUT | `/api/gcal/settings` | Yes | Update user's sync settings |

**Note:** OAuth is now handled via `/api/auth/google/add-scopes` with `scopes: ["calendar"]`, not dedicated GCal endpoints.

### Events
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | `/api/events` | Yes | List user's events. Query: `?status=pending\|confirmed\|synced\|rejected`, `?channel_id=...` |
| GET | `/api/events/today` | Yes | Today's merged events (Alfred + external calendars) for user |
| GET | `/api/events/{id}` | Yes | Get user's event with trigger message |
| PUT | `/api/events/{id}` | Yes | Update user's pending event |
| POST | `/api/events/{id}/confirm` | Yes | Confirm and sync event to user's Google Calendar |
| POST | `/api/events/{id}/reject` | Yes | Reject user's event |
| GET | `/api/events/channel/{channelId}/history` | Yes | Message context for user's channel |

### Reminders
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | `/api/reminders` | Yes | List user's reminders. Query: `?status=pending\|confirmed\|synced\|rejected\|completed\|dismissed`, `?channel_id=...` |
| GET | `/api/reminders/{id}` | Yes | Get user's reminder with trigger message |
| PUT | `/api/reminders/{id}` | Yes | Update user's pending reminder |
| POST | `/api/reminders/{id}/confirm` | Yes | Confirm and sync reminder to Google Calendar |
| POST | `/api/reminders/{id}/reject` | Yes | Reject user's reminder |
| POST | `/api/reminders/{id}/complete` | Yes | Mark user's reminder as completed |
| POST | `/api/reminders/{id}/dismiss` | Yes | Dismiss user's reminder without completing |

**Reminder Fields:**
- `title`, `description`: Text content
- `due_date`: ISO 8601 datetime (required)
- `reminder_time`: ISO 8601 datetime (optional, for notifications)
- `priority`: `low` \| `normal` \| `high`
- `status`: `pending` \| `confirmed` \| `synced` \| `rejected` \| `completed` \| `dismissed`

### Notifications
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | `/api/notifications/preferences` | Yes | Get user's notification settings |
| PUT | `/api/notifications/email` | Yes | Update user's email preferences |
| POST | `/api/notifications/push/register` | Yes | Register Expo push token for user |
| PUT | `/api/notifications/push` | Yes | Update user's push preferences |

### Gmail
| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | `/api/gmail/status` | Yes | Connection status and scopes for user |
| GET | `/api/gmail/sources` | Yes | List user's tracked email sources |
| POST | `/api/gmail/sources` | Yes | Create email source for user |
| GET | `/api/gmail/sources/{id}` | Yes | Get user's email source |
| PUT | `/api/gmail/sources/{id}` | Yes | Update user's email source |
| DELETE | `/api/gmail/sources/{id}` | Yes | Delete user's email source |
| GET | `/api/gmail/top-contacts` | Yes | Get user's top email contacts |
| POST | `/api/gmail/sources/custom` | Yes | Add custom email source for user |

**Note:** Gmail OAuth is now handled via `/api/auth/google/add-scopes` with `scopes: ["gmail"]`.

---

## Database Schema

### Tables (18 total)

**User & Authentication (5 tables):**
| Table | Purpose |
|-------|---------|
| `users` | User accounts (google_id, email, name, avatar_url, created_at, updated_at, last_login_at) |
| `user_sessions` | Active sessions (user_id, token_hash, expires_at, device_info, created_at) |
| `google_tokens` | Encrypted OAuth tokens per user (user_id, access_token_encrypted, refresh_token_encrypted, token_type, expiry, scopes, email) |
| `whatsapp_sessions` | WhatsApp connection tracking per user (user_id, phone_number, device_jid, connected, connected_at) |
| `telegram_sessions` | Telegram connection tracking per user (user_id, phone_number, connected, connected_at) |

**Data Tables (with user_id FK):**
| Table | Purpose |
|-------|---------|
| `channels` | Tracked WhatsApp/Telegram sources (user_id, source_type, type, identifier, name, enabled, total_message_count, last_message_at) |
| `message_history` | Last N messages per channel for Claude context (user_id, channel_id, sender_jid, sender_name, message_text, subject, timestamp) |
| `calendar_events` | Detected calendar events (user_id, channel_id, google_event_id, calendar_id, title, description, location, start_time, end_time, status, action_type, original_message_id, llm_reasoning, email_source_id) |
| `reminders` | Detected reminders/todos (user_id, channel_id, google_event_id, calendar_id, title, description, due_date, reminder_time, priority, status, action_type, original_message_id, llm_reasoning, email_source_id) |
| `event_attendees` | Event participants (event_id, email, display_name, optional) |
| `email_sources` | Tracked email sources for Gmail (user_id, type, identifier, name, enabled) |
| `processed_emails` | Processed email IDs to prevent duplicates (user_id, email_id, processed_at) |
| `gmail_top_contacts` | Cached top contacts for discovery UI (user_id, email, name, email_count, last_updated) |

**Settings Tables (per-user with user_id UNIQUE):**
| Table | Purpose |
|-------|---------|
| `user_notification_preferences` | Email/push notification settings per user (user_id, email_enabled, email_address, push_enabled, push_token, sms_enabled, sms_phone, webhook_enabled, webhook_url) |
| `gmail_settings` | Gmail integration settings per user (user_id, enabled, poll_interval_minutes, last_poll_at, top_contacts_computed_at) |
| `gcal_settings` | Google Calendar sync settings per user (user_id, sync_enabled, selected_calendar_id, selected_calendar_name) |
| `feature_settings` | App feature toggles per user (user_id, smart_calendar_enabled, smart_calendar_setup_complete, whatsapp_input_enabled, telegram_input_enabled, email_input_enabled, sms_input_enabled, alfred_calendar_enabled, google_calendar_enabled, outlook_calendar_enabled, onboarding_complete) |

**System:**
| Table | Purpose |
|-------|---------|
| `schema_migrations` | Database migration version tracking (version, applied_at) |

### Event Status Lifecycle
```
pending → confirmed → synced
    ↓
rejected
```

### Reminder Status Lifecycle
```
pending → confirmed → synced → completed
    ↓                              ↓
rejected                      dismissed
```

**Note:** Reminders have 2 additional terminal states (`completed`, `dismissed`) compared to events.

### Key Types

```go
// Users & Authentication (auth/)
type User struct {
    ID          int64
    GoogleID    string  // Unique Google account ID
    Email       string
    Name        string
    AvatarURL   string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    LastLoginAt *time.Time
}

type UserSession struct {
    ID         int64
    UserID     int64
    TokenHash  string    // SHA-256 hash of session token
    ExpiresAt  time.Time // 30 days from creation
    DeviceInfo string    // Optional device identifier
    CreatedAt  time.Time
}

type GoogleToken struct {
    ID                      int64
    UserID                  int64
    AccessTokenEncrypted    []byte // AES-256-GCM encrypted
    RefreshTokenEncrypted   []byte // AES-256-GCM encrypted
    TokenType               string // "Bearer"
    Expiry                  time.Time
    Scopes                  []string // JSON array: ["profile", "gmail", "calendar"]
    Email                   string
    CreatedAt               time.Time
    UpdatedAt               time.Time
}

// Unified Source Types (source/source.go)
type SourceType string  // "whatsapp" | "telegram" | "gmail"
type ChannelType string // "sender" | "group" | "domain" | "category"

type Message struct {
    SourceType SourceType
    SourceID   int64
    Identifier string
    SenderID   string
    SenderName string
    Text       string
    Subject    string // For emails
    Timestamp  time.Time
}

// CalendarEvent (database/events.go)
type CalendarEvent struct {
    ID            int64
    UserID        int64
    ChannelID     int64
    GoogleEventID *string         // Set after sync
    CalendarID    string          // "primary" or calendar ID
    Title         string
    Description   string
    Location      string
    StartTime     time.Time
    EndTime       *time.Time
    Status        EventStatus     // pending|confirmed|synced|rejected|deleted
    ActionType    EventActionType // create|update|delete
    OriginalMsgID *int64          // FK to message_history(id)
    LLMReasoning  string          // Why Claude detected this event
    EmailSourceID *int64          // FK to email_sources(id) for Gmail
    CreatedAt     time.Time
    UpdatedAt     time.Time
    Attendees     []Attendee
}

// Reminder (database/reminders.go)
type Reminder struct {
    ID            int64
    UserID        int64
    ChannelID     int64
    GoogleEventID *string
    CalendarID    string
    Title         string
    Description   string
    DueDate       time.Time
    ReminderTime  *time.Time       // Optional notification time
    Priority      ReminderPriority // low|normal|high
    Status        ReminderStatus   // pending|confirmed|synced|rejected|completed|dismissed
    ActionType    EventActionType  // create|update|delete
    OriginalMsgID *int64
    LLMReasoning  string
    EmailSourceID *int64
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type ReminderPriority string // "low" | "normal" | "high"
type ReminderStatus string   // "pending" | "confirmed" | "synced" | "rejected" | "completed" | "dismissed"

// Agent Types (agent/)
type EventAnalysis struct {
    HasEvent   bool
    Action     string   // create|update|delete|none
    Event      *EventData
    Reasoning  string
    Confidence float64
}

type ReminderAnalysis struct {
    HasReminder bool
    Action      string   // create|update|delete|none
    Reminder    *ReminderData
    Reasoning   string
    Confidence  float64
}

type Tool struct {
    Name        string
    Description string
    InputSchema map[string]interface{} // JSON schema for tool parameters
}

// FeatureSettings (database/features.go) - per-user settings
type FeatureSettings struct {
    UserID                      int64
    SmartCalendarEnabled        bool
    SmartCalendarSetupComplete  bool
    WhatsAppInputEnabled        bool
    TelegramInputEnabled        bool
    EmailInputEnabled           bool
    SMSInputEnabled             bool
    AlfredCalendarEnabled       bool
    GoogleCalendarEnabled       bool
    OutlookCalendarEnabled      bool
    OnboardingComplete          bool
}

// AppStatus (database/features.go) - derived from FeatureSettings for user
type AppStatus struct {
    OnboardingComplete bool
    WhatsAppEnabled    bool
    TelegramEnabled    bool
    GmailEnabled       bool
    GoogleCalEnabled   bool
}

// EventCreator (processor/event_creator.go) - shared event creation logic
type EventCreationParams struct {
    UserID        int64
    ChannelID     int64
    SourceType    SourceType
    EmailSourceID *int64
    MessageID     *int64
    Analysis      *EventAnalysis
    ExistingEvent *CalendarEvent  // For pending event updates
}

// Thread (gmail/client.go) - email thread with history
type Thread struct {
    ID       string
    Messages []ThreadMessage  // Up to 10 most recent messages
}
```

---

## Project Structure

### Backend (Go)
| Directory | Key Files | Purpose |
|-----------|-----------|---------|
| `internal/auth/` | `auth.go`, `middleware.go`, `encryption.go`, `context.go` | Authentication, OAuth, session management, token encryption (AES-256-GCM) |
| `internal/agent/` | `agent.go`, `analyzer.go`, `tool.go`, `types.go`, `api.go` | Tool-calling agent framework for event/reminder extraction |
| `internal/agent/tools/` | `calendar.go`, `datetime.go`, `location.go`, `attendees.go`, `reminder.go` | Tool implementations for Claude to use |
| `internal/agent/event/` | `event_analyzer.go` | Event detection analyzer |
| `internal/agent/reminder/` | `reminder_analyzer.go` | Reminder detection analyzer |
| `internal/clients/` | `manager.go`, `user_clients.go` | Per-user client lifecycle management (WhatsApp, Telegram, Gmail, GCal) |
| `internal/config/` | `env.go` | Environment configuration loading |
| `internal/database/` | `database.go`, `users.go`, `google_tokens.go`, `channels.go`, `events.go`, `reminders.go`, `features.go`, `messages.go`, `notifications.go`, `gmail.go`, `email_sources.go`, `attendees.go`, `whatsapp_sessions.go`, `telegram_sessions.go` | SQLite data layer with per-user scoping |
| `internal/database/migrations/` | `migrations.go`, `001_*.go`, `002_*.go`, `003_*.go`, `004_reminders.go`, `005_multi_user.go` | Database migrations |
| `internal/server/` | `server.go`, `handlers.go`, `auth_handlers.go`, `reminders_handlers.go`, `gmail_handlers.go`, `features_handlers.go`, `telegram_handlers.go`, `user_service_manager.go` | HTTP API with authentication middleware |
| `internal/source/` | `source.go` | Unified source types (WhatsApp, Telegram, Gmail) |
| `internal/claude/` | `client.go`, `prompt.go` | Claude AI client (legacy non-agent approach, still used) |
| `internal/processor/` | `processor.go`, `email_processor.go`, `event_creator.go`, `history.go` | Message processing pipeline with agent analyzers |
| `internal/whatsapp/` | `client.go`, `handler.go`, `groups.go`, `qr.go` | WhatsApp connection (per-user sessions) |
| `internal/telegram/` | `client.go`, `handler.go`, `groups.go`, `session.go` | Telegram connection (per-user sessions) |
| `internal/gcal/` | `client.go`, `auth.go`, `events.go`, `calendars.go` | Google Calendar integration (per-user clients) |
| `internal/gmail/` | `client.go`, `worker.go`, `scanner.go`, `discovery.go`, `parser.go` | Gmail integration (per-user workers) |
| `internal/notify/` | `service.go`, `notifier.go`, `resend.go`, `expo_push.go` | Notifications (email, push) |
| `internal/onboarding/` | `onboarding.go`, `clients.go` | Setup orchestration |
| `internal/sse/` | `state.go` | Onboarding SSE state |

### Mobile (React Native/Expo)
| Directory | Key Files | Purpose |
|-----------|-----------|---------|
| `mobile/src/api/` | `client.ts`, `auth.ts`, `whatsapp.ts`, `telegram.ts`, `gcal.ts`, `events.ts`, `reminders.ts`, `channels.ts`, `gmail.ts`, `notifications.ts`, `app.ts` | API clients with authentication |
| `mobile/src/hooks/` | `useAuth.ts`, `useAppStatus.ts`, `useEvents.ts`, `useReminders.ts`, `useChannels.ts`, `useTelegram.ts`, `usePushNotifications.ts`, `useOnboardingStatus.ts` | React Query hooks |
| `mobile/src/contexts/` | `AuthContext.tsx` | Authentication context provider (user state, session management) |
| `mobile/src/screens/` | `LoginScreen.tsx`, `HomeScreen.tsx`, `SettingsScreen.tsx`, `PreferencesScreen.tsx` | Main screens |
| `mobile/src/screens/smart-calendar/` | `WhatsAppPreferencesScreen.tsx`, `TelegramPreferencesScreen.tsx`, `GmailPreferencesScreen.tsx`, `GoogleCalendarPreferencesScreen.tsx` | Source preferences |
| `mobile/src/screens/onboarding/` | `WelcomeScreen.tsx`, `InputSelectionScreen.tsx`, `ConnectionScreen.tsx` | Onboarding flow |
| `mobile/src/components/` | `events/`, `reminders/`, `channels/`, `common/`, `home/`, `sources/` | UI components |
| `mobile/src/navigation/` | `RootNavigator.tsx`, `MainNavigator.tsx`, `OnboardingNavigator.tsx` | Navigation (includes login check) |
| `mobile/src/theme/` | `colors.ts`, `typography.ts` | Styling |
| `mobile/src/types/` | `user.ts`, `event.ts`, `reminder.ts`, `channel.ts`, `app.ts`, `features.ts` | TypeScript types |

---

## Environment Variables

### Required
| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Claude API key for event/reminder detection |
| `GOOGLE_CREDENTIALS_FILE` or `GOOGLE_CREDENTIALS_JSON` | OAuth credentials for Google login/Calendar/Gmail |

### Optional - Authentication & Security
| Variable | Default | Description |
|----------|---------|-------------|
| `ALFRED_DEV_MODE` | `false` | Bypass authentication (auto-injects user ID 1 for testing) |
| `ALFRED_BASE_URL` | - | Base URL for OAuth callbacks (e.g., `https://your-domain.com`) |
| `ALFRED_ENCRYPTION_KEY` | (auto-generated) | AES-256 key for token encryption (32 bytes hex). Auto-derived from ANTHROPIC_API_KEY if not set. |

### Optional - Server
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` / `ALFRED_HTTP_PORT` | `8080` | HTTP server port |
| `ALFRED_DB_PATH` | `./alfred.db` | SQLite database path |

### Optional - WhatsApp
| Variable | Default | Description |
|----------|---------|-------------|
| `ALFRED_WHATSAPP_DB_PATH` | `./whatsapp.db` | WhatsApp session DB (user 1 uses this, user N uses `whatsapp.db.user_N`) |
| `ALFRED_DEBUG_ALL_MESSAGES` | `false` | Log all WhatsApp messages (verbose debugging) |

### Optional - Telegram
| Variable | Default | Description |
|----------|---------|-------------|
| `ALFRED_TELEGRAM_API_ID` | - | Telegram API ID (from my.telegram.org) |
| `ALFRED_TELEGRAM_API_HASH` | - | Telegram API Hash (from my.telegram.org) |
| `ALFRED_TELEGRAM_DB_PATH` | `./telegram.db` | Telegram session database (user 1 uses this, user N uses `telegram.db.user_N`) |

### Optional - Claude
| Variable | Default | Description |
|----------|---------|-------------|
| `ALFRED_CLAUDE_MODEL` | `claude-sonnet-4-20250514` | Claude model ID |
| `ALFRED_CLAUDE_TEMPERATURE` | `0.1` | Model temperature (0-1, lower = more deterministic) |

### Optional - Gmail
| Variable | Default | Description |
|----------|---------|-------------|
| `ALFRED_GMAIL_MAX_EMAILS` | `10` | Max emails to process per poll |
| `ALFRED_GMAIL_POLL_INTERVAL` | `1` | Gmail polling interval in minutes |

### Optional - Notifications
| Variable | Default | Description |
|----------|---------|-------------|
| `ALFRED_RESEND_API_KEY` | - | Resend API key for email notifications |
| `ALFRED_EMAIL_FROM` | `Alfred <onboarding@resend.dev>` | Email sender address |

### Optional - Processing
| Variable | Default | Description |
|----------|---------|-------------|
| `ALFRED_MESSAGE_HISTORY_SIZE` | `25` | Messages per channel stored for Claude context |

### Deprecated
| Variable | Status | Notes |
|----------|--------|-------|
| `GOOGLE_TOKEN_FILE` | **Deprecated** | Tokens now stored encrypted in database per user. Not used in multi-user mode. |

---

## Development Patterns

### Authentication in Handlers
```go
// Get authenticated user from context
user := auth.GetUserFromContext(r.Context())
if user == nil {
    respondError(w, http.StatusUnauthorized, "authentication required")
    return
}
userID := user.ID

// Or use helper
userID := getUserID(r)
if userID == 0 {
    respondError(w, http.StatusUnauthorized, "authentication required")
    return
}
```

### Per-User Service Access
```go
// Get user's service manager
services, err := s.userServiceManager.GetServicesForUser(userID)
if err != nil {
    respondError(w, http.StatusServiceUnavailable, "services not available")
    return
}

// Access user's WhatsApp client
wa := services.WhatsApp
if wa == nil {
    respondError(w, http.StatusServiceUnavailable, "WhatsApp not connected")
    return
}

// Access user's GCal client
gcal, err := s.userServiceManager.GetOrCreateGCal(userID)
if err != nil {
    respondError(w, http.StatusServiceUnavailable, "Google Calendar not available")
    return
}
```

### Agent Tool Implementation
```go
// Define tool
tool := agent.Tool{
    Name: "get_current_time",
    Description: "Get current date and time in a specific timezone",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "timezone": map[string]interface{}{
                "type": "string",
                "description": "IANA timezone (e.g., America/New_York)",
            },
        },
        "required": []string{"timezone"},
    },
}

// Implement handler
handler := func(ctx context.Context, input map[string]interface{}) (string, error) {
    tz := input["timezone"].(string)
    location, err := time.LoadLocation(tz)
    if err != nil {
        return "", fmt.Errorf("invalid timezone: %w", err)
    }
    now := time.Now().In(location)
    return now.Format(time.RFC3339), nil
}

// Register with agent
agent.MustRegisterTool(tool, handler)
```

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

### Database Query (with user_id scoping)
```go
rows, err := d.Query(`
    SELECT id, title, status
    FROM calendar_events
    WHERE user_id = ? AND status = ?
`, userID, "pending")
defer rows.Close()
for rows.Next() {
    var id int64
    var title, status string
    if err := rows.Scan(&id, &title, &status); err != nil {
        return err
    }
    // Process row
}
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
railway variables set ALFRED_TELEGRAM_DB_PATH="/data/telegram.db"
railway variables set GOOGLE_CREDENTIALS_JSON='{"web":{...}}'
railway variables set ALFRED_BASE_URL="https://your-domain.com"
railway variables set ALFRED_ENCRYPTION_KEY="<32-byte-hex-key>"
```

**Note:** `GOOGLE_TOKEN_FILE` is deprecated. Tokens are now stored encrypted in the database per user.

### Google OAuth URIs
Add **BOTH** URIs to Google Cloud Console:
```
https://your-domain.com/api/auth/callback
alfred://oauth/callback
```

**Flow:** Google OAuth → `https://your-domain.com/api/auth/callback` (browser) → redirects to `alfred://oauth/callback` (mobile deep link) → mobile app exchanges code for session token

### Persistent Storage
Volume at `/data` stores:
- `alfred.db` - Main database with all user data
- `whatsapp.db` - User 1 WhatsApp session (user N uses `whatsapp.db.user_N`)
- `telegram.db` - User 1 Telegram session (user N uses `telegram.db.user_N`)

Session data persists in database (encrypted OAuth tokens in `google_tokens` table). No need to persist `token.json`.

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

### High (API, UI, auth changes)
| Task | Files |
|------|-------|
| API endpoints | [internal/server/handlers.go](internal/server/handlers.go), [internal/server/server.go](internal/server/server.go) |
| Authentication | [internal/auth/auth.go](internal/auth/auth.go), [internal/auth/middleware.go](internal/auth/middleware.go), [internal/server/auth_handlers.go](internal/server/auth_handlers.go) |
| Agent analyzers | [internal/agent/event/event_analyzer.go](internal/agent/event/event_analyzer.go), [internal/agent/reminder/reminder_analyzer.go](internal/agent/reminder/reminder_analyzer.go) |
| Agent tools | [internal/agent/tools/*.go](internal/agent/tools/) |
| Event/reminder detection | [internal/agent/analyzer.go](internal/agent/analyzer.go), [internal/claude/prompt.go](internal/claude/prompt.go) |
| Mobile screens | `mobile/src/screens/**`, `mobile/src/components/**` |
| Navigation | [mobile/src/navigation/RootNavigator.tsx](mobile/src/navigation/RootNavigator.tsx) |
| Auth context | [mobile/src/contexts/AuthContext.tsx](mobile/src/contexts/AuthContext.tsx) |
| Login flow | [mobile/src/screens/LoginScreen.tsx](mobile/src/screens/LoginScreen.tsx) |

### Medium (Feature changes)
| Task | Files |
|------|-------|
| Database schema | [internal/database/migrations/](internal/database/migrations/) |
| Per-user client management | [internal/clients/manager.go](internal/clients/manager.go), [internal/server/user_service_manager.go](internal/server/user_service_manager.go) |
| WhatsApp processing | [internal/processor/processor.go](internal/processor/processor.go), [internal/whatsapp/handler.go](internal/whatsapp/handler.go) |
| Telegram processing | [internal/processor/processor.go](internal/processor/processor.go), [internal/telegram/handler.go](internal/telegram/handler.go) |
| Gmail processing | [internal/processor/email_processor.go](internal/processor/email_processor.go), [internal/gmail/worker.go](internal/gmail/worker.go) |
| Event creation logic | [internal/processor/event_creator.go](internal/processor/event_creator.go) |
| Reminder management | [internal/database/reminders.go](internal/database/reminders.go), [internal/server/reminders_handlers.go](internal/server/reminders_handlers.go) |
| Google Calendar | [internal/gcal/events.go](internal/gcal/events.go) |
| Notifications | [internal/notify/service.go](internal/notify/service.go) |
| Mobile API hooks | `mobile/src/hooks/**` |
| Mobile API clients | [mobile/src/api/auth.ts](mobile/src/api/auth.ts), [mobile/src/api/reminders.ts](mobile/src/api/reminders.ts) |

### Low (Configuration, deployment)
| Task | Files |
|------|-------|
| Configuration | [internal/config/env.go](internal/config/env.go) |
| Deployment | `Dockerfile`, `railway.toml` |

---

## Testing Philosophy

When fixing bugs, follow test-driven development:
1. Write a test that reproduces the bug first
2. Have the fix prove itself with a passing test
3. Never start by trying to fix without a reproducing test
