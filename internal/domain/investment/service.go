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
	ListTypes(ctx context.Context) ([]TypeDoc, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type CreateInput struct {
	Type       string
	Instrument string
	Amount     float64
	Date       time.Time
	Notes      string
}

type UpdateInput struct {
	Type       *string
	Instrument *string
	Amount     *float64
	Date       *time.Time
	Notes      *string
}

func (s *service) Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error) {
	if in.Amount <= 0 {
		return Model{}, apperr.ValidationErr("amount must be greater than zero")
	}

	displayType, err := s.validateType(ctx, in.Type)
	if err != nil {
		return Model{}, err
	}

	now := time.Now().UTC()

	rec := Model{
		ID:              "",
		UserID:          userID,
		Type:            in.Type,
		DisplayType:     displayType,
		Instrument:      in.Instrument,
		Amount:          in.Amount,
		RetrievedAmount: 0,
		RealizedPnl:     0,
		Status:          StatusActive,
		Date:            shared.ChooseDate(in.Date),
		Notes:           in.Notes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, rec); err != nil {
		return Model{}, apperr.InternalErr("failed to create investment", err)
	}

	return rec, nil
}

func (s *service) List(ctx context.Context, userID common.UserID, month string) ([]Model, error) {
	start, end, err := shared.MonthRange(month)
	if err != nil {
		return nil, err
	}

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
		displayType, err := s.validateType(ctx, *in.Type)
		if err != nil {
			return err
		}
		set["type"] = *in.Type
		set["display_type"] = displayType
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

func (s *service) ListTypes(ctx context.Context) ([]TypeDoc, error) {
	types, err := s.repo.GetTypes(ctx)
	if err != nil {
		return nil, apperr.InternalErr("failed to fetch investment types", err)
	}

	if types == nil {
		return []TypeDoc{}, nil
	}

	return types, nil
}

func (s *service) validateType(ctx context.Context, typ string) (string, error) {
	if typ == "" {
		return "", apperr.ValidationErr("type is required")
	}
	types, err := s.repo.GetTypes(ctx)
	if err != nil {
		return "", apperr.InternalErr("failed to validate investment type", err)
	}
	if len(types) == 0 {
		return "", apperr.ValidationErr("no investment types available")
	}
	for _, t := range types {
		if t.Key == typ && t.Active {
			return t.Name, nil
		}
	}
	return "", apperr.ValidationErr("invalid investment type")
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
