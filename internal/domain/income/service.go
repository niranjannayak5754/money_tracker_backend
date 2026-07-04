package income

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
	Update(ctx context.Context, userID common.UserID, id common.IncomeID, in UpdateInput) error
	Delete(ctx context.Context, userID common.UserID, id common.IncomeID) error
}

type service struct {
	repo Repository
	cats CategoryRepository
}

func NewService(repo Repository, cats CategoryRepository) Service {
	return &service{repo: repo, cats: cats}
}

type CreateInput struct {
	Amount     float64
	Date       time.Time
	CategoryID common.CategoryID
	Source     string
	Notes      string
}

type UpdateInput struct {
	Amount     *float64
	Date       *time.Time
	CategoryID *common.CategoryID
	Source     *string
	Notes      *string
}

func (s *service) Create(
	ctx context.Context,
	userID common.UserID,
	in CreateInput,
) (Model, error) {

	if in.Amount <= 0 {
		return Model{}, apperr.ValidationErr("amount must be greater than zero")
	}

	if in.CategoryID == "" {
		return Model{}, apperr.ValidationErr("category_id is required")
	}

	ok, err := s.cats.ExistsForUserWithType(ctx, userID, in.CategoryID, "income")
	if err != nil {
		return Model{}, apperr.InternalErr("category validation failed", err)
	}
	if !ok {
		return Model{}, apperr.ValidationErr("invalid category")
	}

	now := time.Now().UTC()

	rec := Model{
		ID:         "",
		UserID:     userID,
		Amount:     in.Amount,
		Date:       shared.ChooseDate(in.Date),
		CategoryID: in.CategoryID,
		Source:     in.Source,
		Notes:      in.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	id, err := s.repo.Create(ctx, rec)
	if err != nil {
		return Model{}, apperr.InternalErr("failed to create income", err)
	}
	rec.ID = id

	return rec, nil
}

func (s *service) List(
	ctx context.Context,
	userID common.UserID,
	month string,
) ([]Model, error) {

	start, end, err := shared.MonthRange(month)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.ListByMonth(ctx, userID, start, end)
	if err != nil {
		return nil, apperr.InternalErr("failed to list income", err)
	}

	if items == nil {
		return []Model{}, nil
	}

	// repo returns domain-friendly float amounts

	return items, nil
}

func (s *service) Update(
	ctx context.Context,
	userID common.UserID,
	id common.IncomeID,
	in UpdateInput,
) error {

	set := map[string]any{
		"updated_at": time.Now().UTC(),
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
	if in.CategoryID != nil {
		ok, err := s.cats.ExistsForUserWithType(ctx, userID, *in.CategoryID, "income")
		if err != nil {
			return apperr.InternalErr("category validation failed", err)
		}
		if !ok {
			return apperr.ValidationErr("invalid category")
		}
		set["category_id"] = *in.CategoryID
	}
	if in.Source != nil {
		set["source"] = *in.Source
	}
	if in.Notes != nil {
		set["notes"] = *in.Notes
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return apperr.InternalErr("failed to update income", err)
	}
	if !updated {
		return apperr.NotFoundErr("income not found")
	}

	return nil
}

func (s *service) Delete(
	ctx context.Context,
	userID common.UserID,
	id common.IncomeID,
) error {

	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to delete income", err)
	}
	if !deleted {
		return apperr.NotFoundErr("income not found")
	}

	return nil
}
