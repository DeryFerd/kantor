package hris

import (
	"context"
	"database/sql"
	"time"

	"github.com/kana-consultant/kantor/backend/internal/model"
	repository "github.com/kana-consultant/kantor/backend/internal/repository"
)

type CompensationPolicyRepository struct {
	db repository.DBTX
}

func NewCompensationPolicyRepository(db repository.DBTX) *CompensationPolicyRepository {
	return &CompensationPolicyRepository{db: db}
}

type UpdateCompensationPolicyParams struct {
	MonthlyBaseSalary int64
	MinHoursPerDay    float64
	MinHoursPerMonth  float64
	Timezone          string
	UpdatedBy         string
}

type EmployeeMonthlyHoursRow struct {
	EmployeeID    string
	UserID        *string
	FullName      string
	ActiveSeconds int64
}

type EmployeeDailyHoursRow struct {
	EmployeeID    string
	Date          time.Time
	ActiveSeconds int64
}

func (r *CompensationPolicyRepository) EnsureRow(ctx context.Context) error {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	_, err := repository.DB(ctx, r.db).Exec(ctx, `
		INSERT INTO compensation_policies (tenant_id)
		VALUES (current_setting('app.current_tenant')::uuid)
		ON CONFLICT (tenant_id) DO NOTHING
	`)
	return err
}

func (r *CompensationPolicyRepository) Get(ctx context.Context) (model.CompensationPolicy, error) {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	var policy model.CompensationPolicy
	err := repository.DB(ctx, r.db).QueryRow(ctx, `
		SELECT tenant_id::text, monthly_base_salary,
		       min_hours_per_day::double precision, min_hours_per_month::double precision,
		       timezone, updated_at
		FROM compensation_policies
		LIMIT 1
	`).Scan(
		&policy.TenantID,
		&policy.MonthlyBaseSalary,
		&policy.MinHoursPerDay,
		&policy.MinHoursPerMonth,
		&policy.Timezone,
		&policy.UpdatedAt,
	)
	return policy, err
}

func (r *CompensationPolicyRepository) Update(ctx context.Context, params UpdateCompensationPolicyParams) (model.CompensationPolicy, error) {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	var policy model.CompensationPolicy
	err := repository.DB(ctx, r.db).QueryRow(ctx, `
		UPDATE compensation_policies
		SET monthly_base_salary = $1,
		    min_hours_per_day = $2,
		    min_hours_per_month = $3,
		    timezone = $4,
		    updated_by = NULLIF($5, '')::uuid,
		    updated_at = NOW()
		RETURNING tenant_id::text, monthly_base_salary,
		          min_hours_per_day::double precision, min_hours_per_month::double precision,
		          timezone, updated_at
	`, params.MonthlyBaseSalary, params.MinHoursPerDay, params.MinHoursPerMonth, params.Timezone, params.UpdatedBy).Scan(
		&policy.TenantID,
		&policy.MonthlyBaseSalary,
		&policy.MinHoursPerDay,
		&policy.MinHoursPerMonth,
		&policy.Timezone,
		&policy.UpdatedAt,
	)
	return policy, err
}

func (r *CompensationPolicyRepository) ListMonthlyActiveSeconds(ctx context.Context, from time.Time, to time.Time, employeeID *string) ([]EmployeeMonthlyHoursRow, error) {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	rows, err := repository.DB(ctx, r.db).Query(ctx, `
		SELECT e.id::text, e.user_id::text, e.full_name,
		       COALESCE(SUM(s.total_active_seconds), 0)::bigint AS active_seconds
		FROM employees e
		LEFT JOIN activity_sessions s
		  ON s.user_id = e.user_id
		 AND s.date BETWEEN $1::date AND $2::date
		WHERE e.employment_status IN ('active', 'probation')
		  AND ($3::uuid IS NULL OR e.id = $3::uuid)
		GROUP BY e.id, e.user_id, e.full_name
		ORDER BY e.full_name
	`, from, to, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]EmployeeMonthlyHoursRow, 0)
	for rows.Next() {
		var row EmployeeMonthlyHoursRow
		var userID sql.NullString
		if err := rows.Scan(&row.EmployeeID, &userID, &row.FullName, &row.ActiveSeconds); err != nil {
			return nil, err
		}
		if userID.Valid {
			row.UserID = &userID.String
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *CompensationPolicyRepository) ListDailyHoursViolations(ctx context.Context, from time.Time, to time.Time, minSeconds int64, employeeID *string) ([]EmployeeDailyHoursRow, error) {
	ctx, cancel := repository.QueryContext(ctx)
	defer cancel()

	rows, err := repository.DB(ctx, r.db).Query(ctx, `
		SELECT e.id::text, s.date, SUM(s.total_active_seconds)::bigint AS active_seconds
		FROM employees e
		JOIN activity_sessions s ON s.user_id = e.user_id
		WHERE s.date BETWEEN $1::date AND $2::date
		  AND e.employment_status IN ('active', 'probation')
		  AND ($4::uuid IS NULL OR e.id = $4::uuid)
		GROUP BY e.id, s.date
		HAVING SUM(s.total_active_seconds) > 0 AND SUM(s.total_active_seconds) < $3::bigint
		ORDER BY e.id, s.date
	`, from, to, minSeconds, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]EmployeeDailyHoursRow, 0)
	for rows.Next() {
		var row EmployeeDailyHoursRow
		if err := rows.Scan(&row.EmployeeID, &row.Date, &row.ActiveSeconds); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
