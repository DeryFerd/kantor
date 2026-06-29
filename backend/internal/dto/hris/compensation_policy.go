package hris

type UpdateCompensationPolicyRequest struct {
	MonthlyBaseSalary int64   `json:"monthly_base_salary" validate:"min=0"`
	MinHoursPerDay    float64 `json:"min_hours_per_day" validate:"min=0,max=24"`
	MinHoursPerMonth  float64 `json:"min_hours_per_month" validate:"min=0,max=744"`
	Timezone          string  `json:"timezone" validate:"required,max=64"`
}

type CompensationPolicyResponse struct {
	MonthlyBaseSalary int64   `json:"monthly_base_salary"`
	MinHoursPerDay    float64 `json:"min_hours_per_day"`
	MinHoursPerMonth  float64 `json:"min_hours_per_month"`
	Timezone          string  `json:"timezone"`
	UpdatedAt         string  `json:"updated_at"`
}

type DailyHoursViolationResponse struct {
	Date        string  `json:"date"`
	ActiveHours float64 `json:"active_hours"`
}

type SalarySafetyResponse struct {
	EmployeeID         string                        `json:"employee_id"`
	UserID             *string                       `json:"user_id,omitempty"`
	FullName           string                        `json:"full_name"`
	PeriodYear         int                           `json:"period_year"`
	PeriodMonth        int                           `json:"period_month"`
	MonthlyActiveHours float64                       `json:"monthly_active_hours"`
	MinHoursPerMonth   float64                       `json:"min_hours_per_month"`
	MinHoursPerDay     float64                       `json:"min_hours_per_day"`
	BaseSalary         int64                         `json:"base_salary"`
	DailyViolations    []DailyHoursViolationResponse `json:"daily_violations"`
	Status             string                        `json:"status"`
}
