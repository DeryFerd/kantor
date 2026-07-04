package operational

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	operationaldto "github.com/kana-consultant/kantor/backend/internal/dto/operational"
	"github.com/kana-consultant/kantor/backend/internal/model"
	operationalrepo "github.com/kana-consultant/kantor/backend/internal/repository/operational"
)

var (
	ErrConsentRequired        = errors.New("tracking consent is required before activity can be recorded")
	ErrTrackerSessionNotFound = errors.New("activity session not found")
	ErrDomainCategoryNotFound = errors.New("domain category not found")
)

type trackerRepository interface {
	GetConsent(ctx context.Context, userID string) (model.ActivityConsent, error)
	UpsertConsent(ctx context.Context, userID string, consented bool, ipAddress string, now time.Time) (model.ActivityConsent, error)
	StartSession(ctx context.Context, userID string, params operationalrepo.TrackerStartSessionParams) (model.ActivitySession, error)
	EndSession(ctx context.Context, userID string, sessionID string, endedAt time.Time) (model.ActivitySession, error)
	RecordHeartbeat(ctx context.Context, params operationalrepo.TrackerHeartbeatParams) (model.ActivityEntry, model.ActivitySession, error)
	GetActivityOverview(ctx context.Context, userID string, activityRange operationalrepo.TrackerActivityRange) (model.TrackerActivityOverview, error)
	GetTeamActivity(ctx context.Context, activityRange operationalrepo.TrackerActivityRange, userID *string) (model.TrackerTeamOverview, error)
	GetDailySummary(ctx context.Context, date time.Time) (model.TrackerDailySummary, error)
	ListDomainCategories(ctx context.Context) ([]model.DomainCategory, error)
	ListObservedDomains(ctx context.Context) ([]model.TrackerObservedDomain, error)
	ListConsentAudit(ctx context.Context) ([]model.TrackerConsentAudit, error)
	CreateDomainCategory(ctx context.Context, params operationalrepo.UpsertDomainCategoryParams) (model.DomainCategory, error)
	UpdateDomainCategory(ctx context.Context, domainID string, params operationalrepo.UpsertDomainCategoryParams) (model.DomainCategory, error)
	BulkClassifyObservedDomains(ctx context.Context, domains []string, isProductive bool, category *string) (model.BulkClassifyDomainsResult, error)
	DeleteDomainCategory(ctx context.Context, domainID string) error
	PurgeOldSessions(ctx context.Context, cutoff time.Time) (int64, error)
	EndActiveSessions(ctx context.Context, userID string) (int64, error)
	EndStaleSessions(ctx context.Context, cutoff time.Time) (int64, error)
	GetUserExtensionVersion(ctx context.Context, userID string) (string, error)
}

type TrackerService struct {
	repo          trackerRepository
	retentionDays int
}

type TrackerBatchResult struct {
	Processed int `json:"processed"`
	Skipped   int `json:"skipped"`
}

func NewTrackerService(repo trackerRepository, retentionDays int) *TrackerService {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	return &TrackerService{repo: repo, retentionDays: retentionDays}
}

func (s *TrackerService) GetConsent(ctx context.Context, userID string) (model.ActivityConsent, error) {
	consent, err := s.repo.GetConsent(ctx, userID)
	if errors.Is(err, operationalrepo.ErrTrackerConsentNotFound) {
		return model.ActivityConsent{
			UserID:    userID,
			Consented: false,
		}, nil
	}
	return consent, err
}

func (s *TrackerService) GiveConsent(ctx context.Context, userID string, ipAddress string, now time.Time) (model.ActivityConsent, error) {
	return s.repo.UpsertConsent(ctx, userID, true, ipAddress, now)
}

func (s *TrackerService) RevokeConsent(ctx context.Context, userID string, ipAddress string, now time.Time) (model.ActivityConsent, error) {
	consent, err := s.repo.UpsertConsent(ctx, userID, false, ipAddress, now)
	if err != nil {
		return model.ActivityConsent{}, err
	}
	// Close any open sessions so no further time is credited after consent is
	// withdrawn and a later re-grant cannot resume a stale (back-fillable) one.
	if _, err := s.repo.EndActiveSessions(ctx, userID); err != nil {
		return model.ActivityConsent{}, err
	}
	return consent, nil
}

func (s *TrackerService) StartSession(ctx context.Context, userID string, request operationaldto.TrackerStartSessionRequest, now time.Time) (model.ActivitySession, error) {
	if err := s.requireConsent(ctx, userID); err != nil {
		return model.ActivitySession{}, err
	}

	startedAt := now
	if request.Timestamp != nil {
		startedAt = *request.Timestamp
	}
	// Never let a client start a session dated in the future (skewed clock or
	// abuse) — it would bucket activity into a future date.
	if startedAt.After(now) {
		startedAt = now
	}

	return s.repo.StartSession(ctx, userID, operationalrepo.TrackerStartSessionParams{
		StartedAt:             startedAt,
		TimezoneOffsetMinutes: request.TimezoneOffsetMinutes,
		TimezoneName:          request.TimezoneName,
		ExtensionVersion:      request.ExtensionVersion,
	})
}

func (s *TrackerService) EndSession(ctx context.Context, userID string, sessionID string, now time.Time) (model.ActivitySession, error) {
	session, err := s.repo.EndSession(ctx, userID, sessionID, now)
	if errors.Is(err, operationalrepo.ErrTrackerSessionNotFound) {
		return model.ActivitySession{}, ErrTrackerSessionNotFound
	}
	return session, err
}

func (s *TrackerService) RecordHeartbeat(ctx context.Context, userID string, request operationaldto.TrackerHeartbeatRequest, now time.Time) (model.ActivityEntry, model.ActivitySession, error) {
	if err := s.requireConsent(ctx, userID); err != nil {
		return model.ActivityEntry{}, model.ActivitySession{}, err
	}

	entry, session, err := s.repo.RecordHeartbeat(ctx, operationalrepo.TrackerHeartbeatParams{
		SessionID:             request.SessionID,
		UserID:                userID,
		URL:                   strings.TrimSpace(request.URL),
		Domain:                strings.ToLower(strings.TrimSpace(request.Domain)),
		PageTitle:             request.PageTitle,
		IsIdle:                request.IsIdle,
		Timestamp:             request.Timestamp,
		Now:                   now,
		TimezoneOffsetMinutes: request.TimezoneOffsetMinutes,
		TimezoneName:          request.TimezoneName,
		ExtensionVersion:      request.ExtensionVersion,
	})
	if errors.Is(err, operationalrepo.ErrTrackerSessionNotFound) {
		return model.ActivityEntry{}, model.ActivitySession{}, ErrTrackerSessionNotFound
	}
	return entry, session, err
}

func (s *TrackerService) RecordBatch(ctx context.Context, userID string, request operationaldto.TrackerBatchEntriesRequest, now time.Time) (TrackerBatchResult, error) {
	if err := s.requireConsent(ctx, userID); err != nil {
		return TrackerBatchResult{}, err
	}

	entries := append([]operationaldto.TrackerHeartbeatRequest(nil), request.Entries...)
	sortHeartbeats(entries)

	result := TrackerBatchResult{}
	var currentSessionID string
	for _, entry := range entries {
		// If an earlier entry's session was rotated/recreated server-side (e.g. a
		// stale-gap close), route the remaining replayed beats to the new session
		// instead of the now-closed original so post-gap activity is not dropped.
		if currentSessionID != "" {
			entry.SessionID = currentSessionID
		}
		_, session, err := s.RecordHeartbeat(ctx, userID, entry, now)
		if err != nil {
			result.Skipped++
			continue
		}
		if session.ID != "" {
			currentSessionID = session.ID
		}
		result.Processed++
	}
	return result, nil
}

func (s *TrackerService) GetMyActivity(ctx context.Context, userID string, dateFrom time.Time, dateTo time.Time) (model.TrackerActivityOverview, error) {
	return s.repo.GetActivityOverview(ctx, userID, normalizedRange(dateFrom, dateTo))
}

func (s *TrackerService) GetUserActivity(ctx context.Context, userID string, dateFrom time.Time, dateTo time.Time) (model.TrackerActivityOverview, error) {
	return s.repo.GetActivityOverview(ctx, userID, normalizedRange(dateFrom, dateTo))
}

func (s *TrackerService) GetTeamActivity(ctx context.Context, dateFrom time.Time, dateTo time.Time, userID *string) (model.TrackerTeamOverview, error) {
	return s.repo.GetTeamActivity(ctx, normalizedRange(dateFrom, dateTo), userID)
}

func (s *TrackerService) GetDailySummary(ctx context.Context, date time.Time) (model.TrackerDailySummary, error) {
	normalized := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	return s.repo.GetDailySummary(ctx, normalized)
}

func (s *TrackerService) ListDomainCategories(ctx context.Context) ([]model.DomainCategory, error) {
	return s.repo.ListDomainCategories(ctx)
}

func (s *TrackerService) ListObservedDomains(ctx context.Context) ([]model.TrackerObservedDomain, error) {
	return s.repo.ListObservedDomains(ctx)
}

func (s *TrackerService) ListConsentAudit(ctx context.Context) ([]model.TrackerConsentAudit, error) {
	return s.repo.ListConsentAudit(ctx)
}

func (s *TrackerService) CreateDomainCategory(ctx context.Context, request operationaldto.DomainCategoryRequest) (model.DomainCategory, error) {
	return s.repo.CreateDomainCategory(ctx, operationalrepo.UpsertDomainCategoryParams{
		DomainPattern: request.DomainPattern,
		Category:      request.Category,
		IsProductive:  request.IsProductive,
	})
}

func (s *TrackerService) UpdateDomainCategory(ctx context.Context, domainID string, request operationaldto.DomainCategoryRequest) (model.DomainCategory, error) {
	item, err := s.repo.UpdateDomainCategory(ctx, domainID, operationalrepo.UpsertDomainCategoryParams{
		DomainPattern: request.DomainPattern,
		Category:      request.Category,
		IsProductive:  request.IsProductive,
	})
	if errors.Is(err, operationalrepo.ErrDomainCategoryNotFound) {
		return model.DomainCategory{}, ErrDomainCategoryNotFound
	}
	return item, err
}

func (s *TrackerService) BulkClassifyObservedDomains(ctx context.Context, request operationaldto.TrackerBulkClassifyDomainsRequest) (model.BulkClassifyDomainsResult, error) {
	return s.repo.BulkClassifyObservedDomains(ctx, request.Domains, request.IsProductive, request.Category)
}

func (s *TrackerService) DeleteDomainCategory(ctx context.Context, domainID string) error {
	err := s.repo.DeleteDomainCategory(ctx, domainID)
	if errors.Is(err, operationalrepo.ErrDomainCategoryNotFound) {
		return ErrDomainCategoryNotFound
	}
	return err
}

func (s *TrackerService) PurgeOldData(ctx context.Context, now time.Time) (int64, error) {
	// Sessions store a LOCAL date but the cutoff derives from UTC now; keep one
	// extra day of slack so a timezone offset can never purge data still inside
	// the retention window.
	cutoff := now.AddDate(0, 0, -(s.retentionDays + 1))
	return s.repo.PurgeOldSessions(ctx, cutoff)
}

// trackerStaleSessionThreshold is how long a session may go without a heartbeat
// before the sweeper closes it as abandoned (matches the in-request stale gap).
const trackerStaleSessionThreshold = 15 * time.Minute

// EndStaleSessions closes sessions orphaned by crashed/disconnected clients so a
// later heartbeat or StartSession cannot resurrect them and back-fill the gap.
func (s *TrackerService) EndStaleSessions(ctx context.Context, now time.Time) (int64, error) {
	return s.repo.EndStaleSessions(ctx, now.Add(-trackerStaleSessionThreshold))
}

func (s *TrackerService) requireConsent(ctx context.Context, userID string) error {
	consent, err := s.GetConsent(ctx, userID)
	if err != nil {
		return err
	}
	if !consent.Consented {
		return ErrConsentRequired
	}
	return nil
}

func normalizedRange(dateFrom time.Time, dateTo time.Time) operationalrepo.TrackerActivityRange {
	start := time.Date(dateFrom.Year(), dateFrom.Month(), dateFrom.Day(), 0, 0, 0, 0, dateFrom.Location())
	end := time.Date(dateTo.Year(), dateTo.Month(), dateTo.Day(), 0, 0, 0, 0, dateTo.Location())
	if end.Before(start) {
		end = start
	}
	return operationalrepo.TrackerActivityRange{DateFrom: start, DateTo: end}
}

func sortHeartbeats(entries []operationaldto.TrackerHeartbeatRequest) {
	if len(entries) < 2 {
		return
	}
	sort.Slice(entries, func(i int, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
}
