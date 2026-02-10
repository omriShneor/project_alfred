## DEPLOY TO PROD!!! ##


# Tech Debt
1. Session management is done in memory without any eviction/expiration is this scalable?
2. Whatsapp and telegram sessions are using local file system to manage the session this is probably not scalable and we should consider using S3. 
3. Agent events description, context, location, etc..  are bad and need to be fixed - how can you reliably make this consistently good? is there a way to test this reliably?

# New Feature Ideas
2. Agent-to-Agent communication in coordinating and scheduling time and place to meet. 
3. Voice agent!
4. Prep work for tomorrow?