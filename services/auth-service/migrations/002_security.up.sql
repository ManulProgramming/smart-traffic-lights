CREATE INDEX IF NOT EXISTS users_lower_email_idx ON users (lower(email));
CREATE INDEX IF NOT EXISTS sessions_active_idx ON sessions (user_id, expires_at) WHERE revoked_at IS NULL;
