package model

import "time"

type CompensationPolicy struct {
	TenantID          string    `json:"tenant_id"`
	MonthlyBaseSalary int64     `json:"monthly_base_salary"`
	MinHoursPerDay    float64   `json:"min_hours_per_day"`
	MinHoursPerMonth  float64   `json:"min_hours_per_month"`
	Timezone          string    `json:"timezone"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type SalarySafetyStatus string

const (
	SalarySafetyStatusSafe   SalarySafetyStatus = "safe"
	SalarySafetyStatusAtRisk SalarySafetyStatus = "at_risk"
	SalarySafetyStatusNoData SalarySafetyStatus = "no_data"
)

type DailyHoursViolation struct {
	Date        time.Time `json:"date"`
	ActiveHours float64   `json:"active_hours"`
}

type SalarySafetyEvaluation struct {
	EmployeeID         string                `json:"employee_id"`
	UserID             *string               `json:"user_id,omitempty"`
	FullName           string                `json:"full_name"`
	PeriodYear         int                   `json:"period_year"`
	PeriodMonth        int                   `json:"period_month"`
	MonthlyActiveHours float64               `json:"monthly_active_hours"`
	MinHoursPerMonth   float64               `json:"min_hours_per_month"`
	MinHoursPerDay     float64               `json:"min_hours_per_day"`
	BaseSalary         int64                 `json:"base_salary"`
	DailyViolations    []DailyHoursViolation `json:"daily_violations"`
	Status             SalarySafetyStatus    `json:"status"`
}
