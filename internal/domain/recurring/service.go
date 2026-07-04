package recurring

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

type Service interface {
	Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error)
	List(ctx context.Context, userID common.UserID) ([]Model, error)
	Update(ctx context.Context, userID common.UserID, id common.RecurringID, in UpdateInput) error
	Delete(ctx context.Context, userID common.UserID, id common.RecurringID) error

	// ListDue returns templates due to run, filtering out (and
	// deactivating) any whose next occurrence would fall after EndDate.
	ListDue(ctx context.Context, asOf time.Time) ([]Model, error)

	// MarkRun records that a template was materialized at ranAt and
	// advances it to its next occurrence, deactivating it if that next
	// occurrence would fall after EndDate.
	MarkRun(ctx context.Context, userID common.UserID, id common.RecurringID, ranAt time.Time) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type CreateInput struct {
	EntityType EntityType
	Frequency  Frequency
	DayOfMonth int
	Payload    map[string]any
	StartDate  time.Time
	EndDate    *time.Time
}

type UpdateInput struct {
	DayOfMonth *int
	Payload    map[string]any
	EndDate    *time.Time
	Active     *bool
}

func (s *service) Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error) {
	if in.EntityType != EntityExpense && in.EntityType != EntityIncome && in.EntityType != EntityInvestment {
		return Model{}, apperr.ValidationErr("invalid entity_type")
	}
	if in.Frequency != FrequencyMonthly && in.Frequency != FrequencyYearly {
		return Model{}, apperr.ValidationErr("invalid frequency")
	}
	if in.DayOfMonth < 1 || in.DayOfMonth > 31 {
		return Model{}, apperr.ValidationErr("day_of_month must be between 1 and 31")
	}
	if err := validatePayload(in.EntityType, in.Payload); err != nil {
		return Model{}, err
	}

	start := shared.ChooseDate(in.StartDate)
	if in.EndDate != nil && !in.EndDate.After(start) {
		return Model{}, apperr.ValidationErr("end_date must be after start_date")
	}

	now := time.Now().UTC()
	m := Model{
		UserID:      userID,
		EntityType:  in.EntityType,
		Frequency:   in.Frequency,
		DayOfMonth:  in.DayOfMonth,
		Payload:     in.Payload,
		StartDate:   start,
		EndDate:     in.EndDate,
		NextRunDate: start,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	id, err := s.repo.Create(ctx, m)
	if err != nil {
		return Model{}, apperr.InternalErr("failed to create recurring template", err)
	}
	m.ID = id

	return m, nil
}

func (s *service) List(ctx context.Context, userID common.UserID) ([]Model, error) {
	items, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, apperr.InternalErr("failed to list recurring templates", err)
	}
	if items == nil {
		return []Model{}, nil
	}
	return items, nil
}

func (s *service) Update(ctx context.Context, userID common.UserID, id common.RecurringID, in UpdateInput) error {
	set := map[string]any{"updated_at": time.Now().UTC()}

	if in.DayOfMonth != nil {
		if *in.DayOfMonth < 1 || *in.DayOfMonth > 31 {
			return apperr.ValidationErr("day_of_month must be between 1 and 31")
		}
		set["day_of_month"] = *in.DayOfMonth
	}
	if in.Payload != nil {
		current, err := s.repo.GetByID(ctx, userID, id)
		if err != nil {
			if repository.DataNotFoundErr(err) {
				return apperr.NotFoundErr("recurring template not found")
			}
			return apperr.InternalErr("failed to load recurring template", err)
		}
		if err := validatePayload(current.EntityType, in.Payload); err != nil {
			return err
		}
		set["payload"] = in.Payload
	}
	if in.EndDate != nil {
		set["end_date"] = *in.EndDate
	}
	if in.Active != nil {
		set["active"] = *in.Active
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return apperr.InternalErr("failed to update recurring template", err)
	}
	if !updated {
		return apperr.NotFoundErr("recurring template not found")
	}
	return nil
}

func (s *service) Delete(ctx context.Context, userID common.UserID, id common.RecurringID) error {
	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to delete recurring template", err)
	}
	if !deleted {
		return apperr.NotFoundErr("recurring template not found")
	}
	return nil
}

func (s *service) ListDue(ctx context.Context, asOf time.Time) ([]Model, error) {
	candidates, err := s.repo.ListDue(ctx, asOf)
	if err != nil {
		return nil, apperr.InternalErr("failed to list due recurring templates", err)
	}

	due := make([]Model, 0, len(candidates))
	for _, tmpl := range candidates {
		if tmpl.EndDate != nil && tmpl.NextRunDate.After(*tmpl.EndDate) {
			_, _ = s.repo.Update(ctx, tmpl.UserID, tmpl.ID, map[string]any{
				"active":     false,
				"updated_at": time.Now().UTC(),
			})
			continue
		}
		due = append(due, tmpl)
	}

	return due, nil
}

func (s *service) MarkRun(ctx context.Context, userID common.UserID, id common.RecurringID, ranAt time.Time) error {
	tmpl, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		if repository.DataNotFoundErr(err) {
			return apperr.NotFoundErr("recurring template not found")
		}
		return apperr.InternalErr("failed to load recurring template", err)
	}

	next := nextRunDate(tmpl.NextRunDate, tmpl.Frequency, tmpl.DayOfMonth, tmpl.StartDate.Month())

	active := true
	if tmpl.EndDate != nil && next.After(*tmpl.EndDate) {
		active = false
	}

	set := map[string]any{
		"last_run_date": ranAt,
		"next_run_date": next,
		"active":        active,
		"updated_at":    time.Now().UTC(),
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return apperr.InternalErr("failed to update recurring template after run", err)
	}
	if !updated {
		return apperr.NotFoundErr("recurring template not found")
	}
	return nil
}

func validatePayload(entityType EntityType, payload map[string]any) error {
	amount, ok := toFloat64(payload["amount"])
	if !ok || amount <= 0 {
		return apperr.ValidationErr("payload.amount is required and must be a positive number")
	}

	switch entityType {
	case EntityExpense:
		if s, ok := payload["category_id"].(string); !ok || s == "" {
			return apperr.ValidationErr("payload.category_id is required for expense templates")
		}
	case EntityInvestment:
		if s, ok := payload["type"].(string); !ok || s == "" {
			return apperr.ValidationErr("payload.type is required for investment templates")
		}
	}

	return nil
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
