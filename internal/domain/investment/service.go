package investment

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

type Service interface {
	Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error)
	List(ctx context.Context, userID common.UserID, month string) ([]Model, error)
	Update(ctx context.Context, userID common.UserID, id common.InvestmentID, in UpdateInput) error
	Delete(ctx context.Context, userID common.UserID, id common.InvestmentID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type CreateInput struct {
	Type       Type
	Instrument string
	Amount     float64
	Date       time.Time
	Notes      string
}

type UpdateInput struct {
	Type       *Type
	Instrument *string
	Amount     *float64
	Date       *time.Time
	Notes      *string
}

func (s *service) Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error) {
	if in.Amount <= 0 {
		return Model{}, apperr.ValidationErr("amount must be greater than zero")
	}

	now := time.Now().UTC()

	rec := Model{
		ID:         "",
		UserID:     userID,
		Type:       in.Type,
		Instrument: in.Instrument,
		Amount:     in.Amount,
		Date:       shared.ChooseDate(in.Date),
		Notes:      in.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.Create(ctx, rec); err != nil {
		return Model{}, apperr.InternalErr("failed to create investment", err)
	}

	return rec, nil
}

func (s *service) List(ctx context.Context, userID common.UserID, month string) ([]Model, error) {
	start, end := shared.MonthRange(month)

	items, err := s.repo.ListByMonth(ctx, userID, start, end)
	if err != nil {
		return nil, apperr.InternalErr("failed to list investments", err)
	}

	if items == nil {
		return []Model{}, nil
	}
	return items, nil
}

func (s *service) Update(ctx context.Context, userID common.UserID, id common.InvestmentID, in UpdateInput) error {
	set := map[string]any{"updated_at": time.Now().UTC()}

	if in.Type != nil {
		set["type"] = *in.Type
	}
	if in.Instrument != nil {
		set["instrument"] = *in.Instrument
	}
	if in.Amount != nil {
		if *in.Amount <= 0 {
			return apperr.ValidationErr("amount must be greater than zero")
		}
		set["amount"] = *in.Amount
	}
	if in.Date != nil {
		set["date"] = shared.ChooseDate(*in.Date)
	}
	if in.Notes != nil {
		set["notes"] = *in.Notes
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return apperr.InternalErr("failed to update investment", err)
	}
	if !updated {
		return apperr.NotFoundErr("investment not found")
	}
	return nil
}

func (s *service) Delete(ctx context.Context, userID common.UserID, id common.InvestmentID) error {
	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to delete investment", err)
	}
	if !deleted {
		return apperr.NotFoundErr("investment not found")
	}
	return nil
}
