# Project Alfred

WhatsApp integration service that monitors messages from tracked contacts and groups, with Google Calendar integration, designed as a foundation for an AI assistant.

## Tech Stack

- **Language**: Go 1.25.5
- **Database**: SQLite (via `github.com/mattn/go-sqlite3`)
- **WhatsApp**: `go.mau.fi/whatsmeow` (WhatsApp Web protocol)
- **Google Calendar**: `google.golang.org/api/calendar/v3` with OAuth2
- **QR Codes**: `github.com/skip2/go-qrcode`

## Project Structure

```
project_alfred/
├── main.go                 # Entry point - startup, signal handling, graceful shutdown
├── go.mod                  # Go module dependencies
└── internal/
    ├── config/
    │   └── env.go          # Environment variable configuration
    ├── database/
    │   ├── database.go     # SQLite initialization and migrations
    │   └── channels.go     # Channel CRUD operations
    ├── gcal/
    │   ├── client.go       # Google Calendar client wrapper
    │   ├── auth.go         # OAuth2 flow and token management
    │   └── calendars.go    # Calendar listing operations
    ├── onboarding/
    │   └── onboarding.go   # WhatsApp + Google Calendar connection setup
    ├── server/
    │   ├── server.go       # HTTP server setup and routing
    │   ├── handlers.go     # HTTP request handlers
    │   └── static/
    │       └── admin.html  # Admin panel web UI
    └── whatsapp/
        ├── client.go       # WhatsApp client wrapper
        ├── handler.go      # Message event handling and filtering
        ├── groups.go       # Group/contact discovery
        └── qr.go           # QR code generation and display
```

## Running the Application

```bash
# Set required environment variables
export ANTHROPIC_API_KEY="your-api-key"

# For Google Calendar (optional), place credentials.json in project root

# Run directly
go run main.go

# Or build and run
go build -o alfred
./alfred
```

The application will:
1. Display a QR code for WhatsApp linking (first run only)
2. Open browser for Google Calendar OAuth (if credentials.json exists)
3. Start HTTP server on port 8080
4. Open the admin UI at `http://localhost:8080/admin`

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | Yes | - | Claude API key |
| `GOOGLE_CREDENTIALS_FILE` | No | `./credentials.json` | Google Calendar OAuth credentials |
| `GOOGLE_TOKEN_FILE` | No | `./token.json` | Google Calendar OAuth token storage |
| `ALFRED_DB_PATH` | No | `./alfred.db` | SQLite database path |
| `ALFRED_HTTP_PORT` | No | `8080` | HTTP server port |
| `ALFRED_DEBUG_ALL_MESSAGES` | No | `false` | Log all WhatsApp messages |
| `ALFRED_CLAUDE_MODEL` | No | `claude-sonnet-4-20250514` | Claude model to use |

## Key Concepts

### Channels
A "channel" represents a WhatsApp contact or group that can be tracked. Channels have:
- **type**: `"sender"` (individual contact) or `"group"`
- **identifier**: Phone number (e.g., `1234567890@s.whatsapp.net`) or group JID
- **name**: Display name
- **calendar_id**: Associated Google Calendar (default: `"primary"`)
- **enabled**: Whether to process messages from this channel

### Google Calendar Integration
- OAuth2 authentication with browser-based flow
- Token persistence in `token.json` for automatic reconnection
- Each tracked channel can be assigned a specific Google Calendar
- Calendars are managed via the Admin UI

### Message Flow
1. WhatsApp event handler receives incoming messages
2. Handler extracts text content (supports text, images, videos, documents with captions)
3. Checks if sender/group is in tracked channels
4. If tracked, sends message to internal channel for processing
5. (Phase 3 - TODO) AI assistant processes the message

## Architecture

### Package Responsibilities

- **config**: Loads configuration from environment variables with defaults
- **database**: SQLite connection, migrations, and channel CRUD operations
- **gcal**: Google Calendar OAuth2 authentication and calendar operations
- **onboarding**: Orchestrates WhatsApp and Google Calendar connection setup
- **server**: HTTP API and embedded web UI for channel management
- **whatsapp**: WhatsApp client, message handling, and discovery

### Data Flow

```
WhatsApp → handler.go (filter) → message channel → (Phase 3: assistant)
                ↓
            database (check if tracked)
                ↓
            server/handlers.go (API management)
                ↓
            gcal/client.go (calendar operations)
```

### Databases and Token Storage

- `whatsapp.db`: WhatsApp session data (managed by whatsmeow)
- `alfred.db`: Application data (channels table)
- `token.json`: Google Calendar OAuth tokens

## HTTP API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Dashboard with stats |
| GET | `/admin` | Admin panel web UI |
| GET | `/api/discovery/channels` | List all discoverable WhatsApp channels |
| GET | `/api/channel` | List tracked channels (optional `?type=sender\|group`) |
| POST | `/api/channel` | Create tracked channel |
| PUT | `/api/channel/{id}` | Update channel (name, calendar_id, enabled) |
| DELETE | `/api/channel/{id}` | Delete tracked channel |
| GET | `/api/gcal/status` | Google Calendar connection status |
| GET | `/api/gcal/calendars` | List available Google Calendars |
| POST | `/api/gcal/connect` | Get OAuth URL for Google Calendar |

### Channel API Request/Response

```json
// POST /api/channel
{
  "type": "sender",
  "identifier": "1234567890@s.whatsapp.net",
  "name": "John Doe",
  "calendar_id": "primary"
}

// Response
{
  "id": 1,
  "type": "sender",
  "identifier": "1234567890@s.whatsapp.net",
  "name": "John Doe",
  "calendar_id": "primary",
  "enabled": true,
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Google Calendar API

```json
// GET /api/gcal/status
{
  "connected": true,
  "message": "Connected"
}

// GET /api/gcal/calendars
[
  {
    "id": "primary",
    "summary": "My Calendar",
    "primary": true,
    "access_role": "owner"
  }
]

// POST /api/gcal/connect
{
  "auth_url": "https://accounts.google.com/o/oauth2/auth?...",
  "message": "Open this URL to authorize Google Calendar access"
}
```

## Database Schema

```sql
CREATE TABLE channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL CHECK(type IN ('sender', 'group')),
    identifier TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    calendar_id TEXT NOT NULL DEFAULT 'primary',
    enabled BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_channels_identifier ON channels(identifier);
CREATE INDEX idx_channels_type ON channels(type);
```

## Development Patterns

### Error Handling
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Return errors up the call stack, handle at entry points
- Critical errors go to stderr: `fmt.Fprintf(os.Stderr, ...)`

### Logging
- Use `fmt.Println`/`fmt.Printf` for standard output
- No external logging library - keep it simple

### Database Access
- Standard `database/sql` with prepared statements
- Helper functions: `scanChannel()`, `scanChannelRows()`
- All CRUD operations in `channels.go`

### HTTP Handlers
- Use Go 1.22+ routing: `"GET /path"`, `"POST /path/{id}"`
- Return JSON with `json.NewEncoder(w).Encode()`
- Set `Content-Type: application/json` header

### OAuth2 Flow (Google Calendar)
- Browser-based authorization with localhost callback on port 8089
- Token refresh handled automatically
- Token stored in `token.json` with 0600 permissions

### Adding New API Endpoints
1. Add route in `server/server.go` `registerRoutes()`
2. Implement handler in `server/handlers.go`
3. Add database operations in `internal/database/` if needed

### Adding New Configuration
1. Add field to `Config` struct in `config/env.go`
2. Load in `LoadFromEnv()` using helper functions
3. Use `getEnvOrDefault()`, `getEnvAsIntOrDefault()`, or `getEnvAsBoolOrDefault()`

## Project Status

- **Phase 1**: WhatsApp connection and message filtering (complete)
- **Phase 2**: Channel discovery, management UI, and Google Calendar integration (complete)
- **Phase 3**: AI assistant integration (TODO - see `main.go:59`)

## Common Files to Modify

| Task | Files |
|------|-------|
| Add API endpoint | `server/server.go`, `server/handlers.go` |
| Add database table/field | `database/database.go` (migrations), new file for CRUD |
| Add configuration | `config/env.go` |
| Modify message handling | `whatsapp/handler.go` |
| Update admin UI | `server/static/admin.html` |
| Add WhatsApp features | `whatsapp/client.go`, `whatsapp/groups.go` |
| Add Google Calendar features | `gcal/client.go`, `gcal/calendars.go` |
