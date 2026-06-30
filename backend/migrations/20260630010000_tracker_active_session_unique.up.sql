-- Enforce one active activity_session per user. Existing duplicate active
-- sessions (left by the pre-fix StartSession race or never-ended sessions) are
-- closed at their last-seen time, keeping the most recent per user, so the
-- unique index can be created.
UPDATE activity_sessions
SET is_active = FALSE, end_time = updated_at
WHERE is_active
  AND id NOT IN (
    SELECT DISTINCT ON (tenant_id, user_id) id
    FROM activity_sessions
    WHERE is_active
    ORDER BY tenant_id, user_id, start_time DESC
  );

CREATE UNIQUE INDEX IF NOT EXISTS uq_activity_sessions_one_active
  ON activity_sessions (tenant_id, user_id)
  WHERE is_active;
