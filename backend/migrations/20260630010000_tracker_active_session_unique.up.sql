-- Enforce one active activity_session per user. Existing duplicate active
-- sessions (left by the pre-fix StartSession race or never-ended sessions) are
-- closed at their last-seen time, keeping the most recent per user, so the
-- unique index can be created.
--
-- activity_sessions has FORCE ROW LEVEL SECURITY, and migrations run without an
-- app.current_tenant set (and in production as a non-superuser owner), so a
-- plain UPDATE would be filtered to zero rows by the tenant_isolation policy and
-- the dedup would silently do nothing — making CREATE UNIQUE INDEX abort on any
-- instance that actually has duplicates. Loop per tenant and set the GUC, as
-- 20260413093000_operational_kanban_column_type.up.sql does.
DO $$
DECLARE
  current_tenant_id UUID;
BEGIN
  FOR current_tenant_id IN SELECT id FROM tenants LOOP
    PERFORM set_config('app.current_tenant', current_tenant_id::text, true);
    UPDATE activity_sessions
    SET is_active = FALSE, end_time = updated_at
    WHERE is_active
      AND tenant_id = current_tenant_id
      AND id NOT IN (
        SELECT DISTINCT ON (user_id) id
        FROM activity_sessions
        WHERE is_active AND tenant_id = current_tenant_id
        ORDER BY user_id, start_time DESC
      );
  END LOOP;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_activity_sessions_one_active
  ON activity_sessions (tenant_id, user_id)
  WHERE is_active;
