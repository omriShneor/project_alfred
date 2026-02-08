# Tech Debt
1. Session management is done in memory without any eviction/expiration is this scalable?
2. Whatsapp and telegram sessions are using files to manage the session this is probably not scalable and we should consider using S3. 