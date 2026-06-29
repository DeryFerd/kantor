package hris

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kana-consultant/kantor/backend/internal/model"
	hrisrepo "github.com/kana-consultant/kantor/backend/internal/repository/hris"
)

var ErrCompensationPolicyInvalid = errors.New("compensation policy is invalid")

type compensationPolicyRepository interface {
	EnsureRow(ctx context.Context) error
	Get(ctx context.Context) (model.CompensationPolicy, error)
	Update(ctx context.Context, params hrisrepo.UpdateCompensationPolicyParams) (model.CompensationPolicy, error)
	ListMonthlyActiveSeconds(ctx context.Context, from time.Time, to time.Time, employeeID *string) ([]hrisrepo.EmployeeMonthlyHoursRow, error)
	ListDailyHoursViolations(ctx context.Context, from time.Time, to time.Time, minSeconds int64, employeeID *string) ([]hrisrepo.EmployeeDailyHoursRow, error)
}

type CompensationPolicyService struct {
	repo compensationPolicyRepository
}

func NewCompensationPolicyService(repo compensationPolicyRepository) *CompensationPolicyService {
	return &CompensationPolicyService{repo: repo}
}

func (s *CompensationPolicyService) GetPolicy(ctx context.Context) (model.CompensationPolicy, error) {
	if err := s.repo.EnsureRow(ctx); err != nil {
		return model.CompensationPolicy{}, err
	}
	return s.repo.Get(ctx)
}

func (s *CompensationPolicyService) UpdatePolicy(ctx context.Context, params hrisrepo.UpdateCompensationPolicyParams) (model.CompensationPolicy, error) {
	if err := validateCompensationPolicy(params); err != nil {
		return model.CompensationPolicy{}, err
	}
	if err := s.repo.EnsureRow(ctx); err != nil {
		return model.CompensationPolicy{}, err
	}
	return s.repo.Update(ctx, params)
}

func (s *CompensationPolicyService) EvaluateSalarySafety(ctx context.Context, year int, month int, employeeID *string) ([]model.SalarySafetyEvaluation, error) {
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("%w: month must be between 1 and 12", ErrCompensationPolicyInvalid)
	}
	if year < 2000 || year > 9999 {
		return nil, fmt.Errorf("%w: year is out of range", ErrCompensationPolicyInvalid)
	}

	policy, err := s.GetPolicy(ctx)
	if err != nil {
		return nil, err
	}

	location, err := time.LoadLocation(policy.Timezone)
	if err != nil {
		location = time.UTC
	}
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, location)
	to := from.AddDate(0, 1, -1)

	monthly, err := s.repo.ListMonthlyActiveSeconds(ctx, from, to, employeeID)
	if err != nil {
		return nil, err
	}

	minDaySeconds := int64(policy.MinHoursPerDay * 3600)
	violationRows, err := s.repo.ListDailyHoursViolations(ctx, from, to, minDaySeconds, employeeID)
	if err != nil {
		return nil, err
	}

	violationsByEmployee := make(map[string][]model.DailyHoursViolation)
	for _, row := range violationRows {
		violationsByEmployee[row.EmployeeID] = append(violationsByEmployee[row.EmployeeID], model.DailyHoursViolation{
			Date:        row.Date,
			ActiveHours: secondsToHours(row.ActiveSeconds),
		})
	}

	evaluations := make([]model.SalarySafetyEvaluation, 0, len(monthly))
	for _, row := range monthly {
		monthlyHours := secondsToHours(row.ActiveSeconds)
		violations := violationsByEmployee[row.EmployeeID]
		if violations == nil {
			violations = []model.DailyHoursViolation{}
		}
		evaluations = append(evaluations, model.SalarySafetyEvaluation{
			EmployeeID:         row.EmployeeID,
			UserID:             row.UserID,
			FullName:           row.FullName,
			PeriodYear:         year,
			PeriodMonth:        month,
			MonthlyActiveHours: monthlyHours,
			MinHoursPerMonth:   policy.MinHoursPerMonth,
			MinHoursPerDay:     policy.MinHoursPerDay,
			BaseSalary:         policy.MonthlyBaseSalary,
			DailyViolations:    violations,
			Status:             evaluateStatus(row.UserID, monthlyHours, policy.MinHoursPerMonth, violations),
		})
	}
	return evaluations, nil
}

func evaluateStatus(userID *string, monthlyHours float64, minMonthly float64, violations []model.DailyHoursViolation) model.SalarySafetyStatus {
	if userID == nil {
		return model.SalarySafetyStatusNoData
	}
	if monthlyHours >= minMonthly && len(violations) == 0 {
		return model.SalarySafetyStatusSafe
	}
	return model.SalarySafetyStatusAtRisk
}

func validateCompensationPolicy(params hrisrepo.UpdateCompensationPolicyParams) error {
	if params.MonthlyBaseSalary < 0 {
		return fmt.Errorf("%w: base salary must not be negative", ErrCompensationPolicyInvalid)
	}
	if params.MinHoursPerDay < 0 || params.MinHoursPerDay > 24 {
		return fmt.Errorf("%w: min hours per day must be between 0 and 24", ErrCompensationPolicyInvalid)
	}
	if params.MinHoursPerMonth < 0 {
		return fmt.Errorf("%w: min hours per month must not be negative", ErrCompensationPolicyInvalid)
	}
	if strings.TrimSpace(params.Timezone) == "" {
		return fmt.Errorf("%w: timezone is required", ErrCompensationPolicyInvalid)
	}
	if _, err := time.LoadLocation(params.Timezone); err != nil {
		return fmt.Errorf("%w: timezone is not recognized", ErrCompensationPolicyInvalid)
	}
	return nil
}

func secondsToHours(seconds int64) float64 {
	return math.Round(float64(seconds)/3600*100) / 100
}
