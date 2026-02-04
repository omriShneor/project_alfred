# Phase 13: Production Deployment Checklist

## Multi-User Architecture Deployment to Railway

This deployment migrates Project Alfred to a **full multi-user architecture** where each user has isolated WhatsApp/Telegram sessions.

## ⚠️ BREAKING CHANGES

**All users will need to re-authenticate WhatsApp and Telegram after this deployment.**

- Old session files (`whatsapp.db`, `telegram.db`) will be deleted automatically
- New per-user sessions: `whatsapp.db.user_1`, `telegram.db.user_2`, etc.
- This is a **one-time migration** with no backward compatibility
- User data (events, channels, reminders) will be preserved

## Pre-Deployment Checklist

### 1. Code Review ✅
- [x] All tests passing (273 tests: unit + E2E + mobile)
- [x] Build successful (`go build`)
- [x] No compilation errors
- [x] Multi-user implementation complete (Phases 1-11)
- [x] Test updates complete (Phase 12)

### 2. Git Commit
```bash
# Stage all changes
git add -A

# Create commit with multi-user migration
git commit -m "$(cat <<'EOF'
Multi-user architecture: Per-user WhatsApp/Telegram sessions

BREAKING CHANGE: Users must re-authenticate WhatsApp/Telegram

Changes:
- Add ClientManager for per-user WhatsApp/Telegram clients
- Session files now per-user: whatsapp.db.user_{userID}
- Update all database queries to filter by user_id
- Fix StoreSourceMessage to populate user_id from channel
- Update test infrastructure with MockClientManager
- Delete legacy onboarding package (replaced by ClientManager)

Migration:
- CleanupLegacySessions() auto-deletes old session files on startup
- All user data preserved (events, channels, messages)
- Sessions must be recreated (users re-pair services)

Test Results:
- Unit tests: PASS (all database, agent, processor)
- E2E tests: PASS (16 suites, 257 tests)
- Mobile tests: PASS
- Build: SUCCESS

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"

# Verify commit
git log -1 --stat
```

### 3. Environment Variables Check

#### Required (Must be set in Railway)
- [x] `ANTHROPIC_API_KEY` - Claude API key
- [x] `GOOGLE_CREDENTIALS_JSON` - OAuth credentials (JSON string)

#### Recommended for Production
```bash
# Database paths (persistent volume)
ALFRED_DB_PATH=/data/alfred.db
ALFRED_WHATSAPP_DB_PATH=/data/whatsapp.db
ALFRED_TELEGRAM_DB_PATH=/data/telegram.db

# Base URL for OAuth callbacks
ALFRED_BASE_URL=https://alfred-production-d2c9.up.railway.app

# Optional: Telegram API credentials
ALFRED_TELEGRAM_API_ID=your_api_id
ALFRED_TELEGRAM_API_HASH=your_api_hash

# Optional: Email notifications
ALFRED_RESEND_API_KEY=re_...
```

#### Development Mode (Disable for Production!)
```bash
# DO NOT SET THIS IN PRODUCTION
# ALFRED_DEV_MODE=false  (default is false, no need to set)
```

### 4. Railway Volume Configuration

Verify persistent volume is mounted at `/data`:
```bash
railway volume list
# Should show volume mounted at /data
```

If not created:
```bash
railway volume add --mount-path /data
```

### 5. Pre-Deployment Commands

```bash
# Verify all tests pass
ALFRED_ENCRYPTION_KEY=test-key make test-all

# Verify build works
go build -o /tmp/alfred .

# Check for uncommitted changes
git status

# Push to main branch
git push origin main
```

## Deployment Steps

### Step 1: Deploy to Railway

```bash
# Deploy current commit
railway up

# Or trigger deployment via Railway dashboard
# Railway will automatically build and deploy when you push to main
```

### Step 2: Monitor Deployment

```bash
# Watch deployment logs
railway logs

# Look for key messages:
# ✅ "Deleted legacy session file: /data/whatsapp.db"
# ✅ "Deleted legacy session file: /data/telegram.db"
# ✅ "Starting HTTP server on http://localhost:8080"
# ✅ "ClientManager: Starting session restoration"
```

### Step 3: Verify Health Check

```bash
# Check health endpoint
curl https://alfred-production-d2c9.up.railway.app/health

# Expected response:
# {
#   "status": "healthy",
#   "database": "ok",
#   "whatsapp": "disconnected",  # Expected - no sessions yet
#   "google_calendar": "ok"
# }
```

### Step 4: Verify Legacy Cleanup

Check logs for confirmation that old session files were deleted:
```bash
railway logs | grep "Deleted legacy session file"
```

Expected output:
```
Deleted legacy session file: /data/whatsapp.db
Deleted legacy session file: /data/telegram.db
```

### Step 5: Test User Authentication Flow

1. **Login via mobile app**
   - User should be able to log in with Google
   - Auth token should work correctly

2. **Re-pair WhatsApp**
   - Navigate to WhatsApp setup
   - Enter phone number
   - Enter pairing code from WhatsApp app
   - Verify connection shows "Connected"
   - **New session file created**: `/data/whatsapp.db.user_{userID}`

3. **Re-pair Telegram**
   - Navigate to Telegram setup
   - Enter phone number
   - Enter verification code from Telegram
   - Verify connection shows "Connected"
   - **New session file created**: `/data/telegram.db.user_{userID}`

4. **Verify existing data**
   - Check that past events are still visible
   - Verify channels are still configured
   - Confirm reminders are intact

### Step 6: Test Multi-User Isolation

If you have test accounts:

1. **User 1 pairs WhatsApp**
   ```bash
   railway ssh
   ls -la /data/whatsapp.db.user_*
   # Should show: whatsapp.db.user_1
   ```

2. **User 2 pairs WhatsApp**
   ```bash
   ls -la /data/whatsapp.db.user_*
   # Should show: whatsapp.db.user_1, whatsapp.db.user_2
   ```

3. **Send messages to both users**
   - Verify User 1 only sees their events
   - Verify User 2 only sees their events
   - No cross-user data leakage

## Post-Deployment Verification

### Checklist
- [ ] Health check returns 200 OK
- [ ] Legacy session files deleted (check logs)
- [ ] New per-user session files created on re-pairing
- [ ] User login/authentication works
- [ ] WhatsApp re-pairing works
- [ ] Telegram re-pairing works
- [ ] Existing events/channels/data preserved
- [ ] No errors in Railway logs
- [ ] Memory usage normal (check Railway metrics)
- [ ] No cross-user data leakage (if testing with multiple accounts)

### Expected Behavior

**Normal:**
- ✅ Users must re-pair WhatsApp/Telegram
- ✅ Session files follow new naming: `{service}.db.user_{userID}`
- ✅ Old session files deleted automatically
- ✅ All existing user data intact

**Issues to Watch For:**
- ❌ WhatsApp/Telegram pairing fails → Check Telegram API credentials
- ❌ "Channel does not exist" errors → Check StoreSourceMessage fix deployed
- ❌ Cross-user data visible → Check database query filters
- ❌ Old session files not deleted → Check CleanupLegacySessions() called

## Rollback Plan

If critical issues occur:

### Quick Rollback
```bash
# Revert to previous Railway deployment
railway rollback

# Or redeploy specific commit
git log --oneline  # Find previous commit
railway up --detach <commit-hash>
```

### Data Recovery
- User data (events, channels, messages) is preserved in `/data/alfred.db`
- Session files can be recreated by re-pairing
- No data loss expected from this migration

## Communication to Users

**Recommended notification:**

> **Important Update: WhatsApp & Telegram Re-authentication Required**
>
> We've upgraded Alfred to support multiple users with improved data isolation.
>
> **Action Required:**
> - Re-pair your WhatsApp connection (Settings → WhatsApp)
> - Re-pair your Telegram connection (Settings → Telegram)
>
> Your existing events, channels, and settings are preserved.
>
> This is a one-time update. Thank you for your patience!

## Monitoring

### First 24 Hours
```bash
# Monitor logs continuously
railway logs --follow

# Watch for patterns:
# - Successful re-pairings: "ClientManager: Creating WhatsApp client for user X"
# - Session file creation: "whatsapp.db.user_X"
# - Any errors or panics
```

### Metrics to Monitor
- Memory usage (should be stable, ~50-100MB per active user)
- Error rate (should be minimal)
- Response times (should remain fast)
- Active user sessions (check `/data/whatsapp.db.user_*` file count)

## Phase 14: Post-Deployment Cleanup

**After 1-2 weeks of stable operation:**

1. **Remove CleanupLegacySessions()**
   - This function is only needed for the one-time migration
   - Should be removed after all users have upgraded

2. **Create cleanup commit:**
   ```bash
   # Edit internal/clients/manager.go
   # Remove CleanupLegacySessions() method

   # Edit main.go
   # Remove call to clientManager.CleanupLegacySessions()

   git commit -m "Remove legacy session cleanup (migration complete)"
   railway up
   ```

## Success Criteria

✅ All users can log in successfully
✅ WhatsApp re-pairing works smoothly
✅ Telegram re-pairing works smoothly
✅ Existing data fully accessible
✅ No cross-user data leakage
✅ No critical errors in logs
✅ Memory usage stable
✅ Session files correctly named (`*.user_{userID}`)

## Timeline

- **Day 0**: Deploy to Railway (this checklist)
- **Day 1-7**: Monitor logs, assist users with re-pairing
- **Day 7-14**: Verify stability, no rollbacks needed
- **Day 14+**: Phase 14 cleanup (remove CleanupLegacySessions)

## Notes

- This is a **clean migration** - no backward compatibility
- Session files are the only breaking change
- All database data is preserved
- Per-user session isolation prevents future multi-user conflicts
- CleanupLegacySessions() is temporary (remove after migration)
