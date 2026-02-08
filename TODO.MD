## DEPLOY TO PROD!!! ##


# Tech Debt
1. Session management is done in memory without any eviction/expiration is this scalable?
2. unbounded analyzer goroutine fan-out under load (plus lossy queue behavior).

Where it happens

Incoming messages from all users share one buffered channel: manager.go (line 54).
A single loop consumes that channel: processor.go (line 82).
For each message, the processor spawns up to two fire-and-forget goroutines:
event analyzer: processor.go (line 154)
reminder analyzer: processor.go (line 175)
Gmail path does the same for each email:
event analyzer: email_processor.go (line 68)
reminder analyzer: email_processor.go (line 89)
and Gmail worker can feed many emails in a loop: worker.go (line 223), worker.go (line 247).
Why this is dangerous

There is no concurrency cap (no semaphore/worker pool/rate limit) around analysis calls.
If input rate > analysis completion rate, goroutines accumulate unbounded.
Each goroutine may do expensive LLM work and DB writes, increasing CPU, memory, and DB contention.
When ingress backs up, producers start dropping messages:
WhatsApp drop path: handler.go (line 201), handler.go (line 213)
Telegram drop path: handler.go (line 157), handler.go (line 169)
Result is not just slowness; it is correctness loss (missed reminders/events).
Why severity is High (not Critical)

It is primarily an availability/reliability risk, not a direct auth/data-isolation breach.
Under realistic load spikes, impact is major:
dropped user messages
missed detections
latency spikes
potential OOM / instability
Blast radius is system-wide because queue and processor are shared across users.
Concrete failure mode example
If analysis takes ~2s and tracked ingress is 20 msg/s, you can create ~40 analyzer goroutines per second (2 per message). That backlog grows rapidly, queue fills, then new messages are dropped.

Operational signals you’d see

Frequent "message channel full, dropping message" logs.
Rising goroutine count and memory footprint.
Increased DB busy/lock pressure and slower handler response.
User-visible “missed” events/reminders.
If you want, next I can implement a bounded worker-pool fix for this path (with metrics and safe shutdown), then we can re-evaluate severity and residual risk.
3. Whatsapp and telegram sessions are using local file system to manage the session this is probably not scalable and we should consider using S3. 

# New Feature Ideas
1. Agent events description, context, location, etc..  are bad and need to be fixed - how can you reliably make this consistently good? is there a way to test this reliably?
2. Agent-to-Agent communication in coordinating and scheduling time and place to meet. 
3. Voice agent!
4. Prep work for tomorrow?