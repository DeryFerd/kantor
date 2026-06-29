CREATE TABLE compensation_policies (
    tenant_id UUID PRIMARY KEY DEFAULT current_setting('app.current_tenant')::uuid REFERENCES tenants(id) ON DELETE CASCADE,
    monthly_base_salary BIGINT NOT NULL DEFAULT 30000000 CHECK (monthly_base_salary >= 0),
    min_hours_per_day NUMERIC(5,2) NOT NULL DEFAULT 4 CHECK (min_hours_per_day >= 0 AND min_hours_per_day <= 24),
    min_hours_per_month NUMERIC(6,2) NOT NULL DEFAULT 160 CHECK (min_hours_per_month >= 0),
    timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE compensation_policies ENABLE ROW LEVEL SECURITY;
ALTER TABLE compensation_policies FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON compensation_policies
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::uuid);

DO $$ BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'kantor_app') THEN
        EXECUTE 'GRANT SELECT, INSERT, UPDATE, DELETE ON compensation_policies TO kantor_app';
    END IF;
END $$;
