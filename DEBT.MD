# Tech Debt
1. Session management is done in memory without any eviction/expiration is this scalable?
2. Whatsapp and telegram sessions are using files to manage the session this is probably not scalable and we should consider using S3. 
3. reset onboarding endpoint should only reset my onboarding status.
4. dev user - omri shneor - created automatically id ALFRED_DEV_MODE=true