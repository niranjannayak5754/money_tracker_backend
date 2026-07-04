package expense

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

type Service interface {
	Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error)
	List(ctx context.Context, userID common.UserID, month, category string) ([]Model, error)
	Update(ctx context.Context, userID common.UserID, id common.ExpenseID, in UpdateInput) error
	Delete(ctx context.Context, userID common.UserID, id common.ExpenseID) error
}

type service struct {
	repo Repository
	cats CategoryRepository
}

func NewService(
	repo Repository,
	cats CategoryRepository,
) Service {
	return &service{
		repo: repo,
		cats: cats,
	}
}

type CreateInput struct {
	Amount     float64
	Date       time.Time
	CategoryID common.CategoryID
	Merchant   string
	Notes      string
	Tags       []string
}

type UpdateInput struct {
	Amount     *float64
	Date       *time.Time
	CategoryID *common.CategoryID
	Merchant   *string
	Notes      *string
	Tags       *[]string
}

func (s *service) Create(
	ctx context.Context,
	userID common.UserID,
	in CreateInput,
) (Model, error) {

	if in.Amount <= 0 {
		return Model{}, apperr.ValidationErr("amount must be greater than zero")
	}

	ok, err := s.cats.ExistsForUser(ctx, userID, in.CategoryID)
	if err != nil {
		return Model{}, apperr.InternalErr("category validation failed", err)
	}
	if !ok {
		return Model{}, apperr.ValidationErr("invalid category")
	}

	now := time.Now().UTC()

	exp := Model{
		ID:         "", // repository will set underlying mongo id
		UserID:     userID,
		Amount:     in.Amount,
		Date:       shared.ChooseDate(in.Date),
		CategoryID: in.CategoryID,
		Merchant:   in.Merchant,
		Notes:      in.Notes,
		Tags:       in.Tags,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.Create(ctx, exp); err != nil {
		return Model{}, apperr.InternalErr("failed to create expense", err)
	}

	return exp, nil
}

func (s *service) List(
	ctx context.Context,
	userID common.UserID,
	month, category string,
) ([]Model, error) {

	start, end, err := shared.MonthRange(month)
	if err != nil {
		return nil, err
	}

	var catID *common.CategoryID
	if category != "" {
		cid := common.CategoryID(category)
		catID = &cid
	}

	items, err := s.repo.ListByMonth(ctx, userID, start, end, catID)
	if err != nil {
		return nil, apperr.InternalErr("failed to list expenses", err)
	}

	if items == nil {
		return []Model{}, nil
	}

	return items, nil
}

func (s *service) Update(
	ctx context.Context,
	userID common.UserID,
	id common.ExpenseID,
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
		date := shared.ChooseDate(*in.Date)
		set["date"] = &date
	}

	if in.CategoryID != nil {
		ok, err := s.cats.ExistsForUser(ctx, userID, *in.CategoryID)
		if err != nil {
			return apperr.InternalErr("category validation failed", err)
		}
		if !ok {
			return apperr.ValidationErr("invalid category")
		}
		set["category_id"] = *in.CategoryID
	}

	if in.Merchant != nil {
		set["merchant"] = in.Merchant
	}

	if in.Notes != nil {
		set["notes"] = in.Notes
	}

	if in.Tags != nil {
		set["tags"] = in.Tags
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return apperr.InternalErr("failed to update expense", err)
	}

	if !updated {
		return apperr.NotFoundErr("expense not found")
	}

	return nil
}

func (s *service) Delete(
	ctx context.Context,
	userID common.UserID,
	id common.ExpenseID,
) error {

	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to delete expense", err)
	}

	if !deleted {
		return apperr.NotFoundErr("expense not found")
	}

	return nil
}
