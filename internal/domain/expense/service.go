package expense

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

type Service interface {
	Create(ctx context.Context, userID primitive.ObjectID, in CreateInput) (Model, error)
	List(ctx context.Context, userID primitive.ObjectID, month, category string) ([]Model, error)
	Update(ctx context.Context, userID, id primitive.ObjectID, in UpdateInput) error
	Delete(ctx context.Context, userID, id primitive.ObjectID) error
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
	CategoryID primitive.ObjectID
	Merchant   string
	Notes      string
	Tags       []string
}

type UpdateInput struct {
	Amount     *float64
	Date       *time.Time
	CategoryID *primitive.ObjectID
	Merchant   *string
	Notes      *string
	Tags       *[]string
}

func (s *service) Create(
	ctx context.Context,
	userID primitive.ObjectID,
	in CreateInput,
) (Model, error) {

	if in.Amount <= 0 {
		return Model{}, apperr.ValidationErr("amount must be greater than zero")
	}

	dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", in.Amount))
	if err != nil {
		return Model{}, apperr.InternalErr("invalid amount format", err)
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
		ID:         primitive.NewObjectID(),
		UserID:     userID,
		Amount:     dec,
		AmountF:    in.Amount,
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
	userID primitive.ObjectID,
	month, category string,
) ([]Model, error) {

	start, end := shared.MonthRange(month)

	var catID *primitive.ObjectID
	if category != "" {
		id, err := primitive.ObjectIDFromHex(category)
		if err != nil {
			return nil, apperr.ValidationErr("invalid category id")
		}
		catID = &id
	}

	items, err := s.repo.ListByMonth(ctx, userID, start, end, catID)
	if err != nil {
		return nil, apperr.InternalErr("failed to list expenses", err)
	}

	if items == nil {
		return []Model{}, nil
	}

	for i := range items {
		items[i].AmountF = shared.Decimal128ToFloat(items[i].Amount)
	}

	return items, nil
}

func (s *service) Update(
	ctx context.Context,
	userID, id primitive.ObjectID,
	in UpdateInput,
) error {

	set := map[string]any{
		"updated_at": time.Now().UTC(),
	}

	if in.Amount != nil {
		if *in.Amount <= 0 {
			return apperr.ValidationErr("amount must be greater than zero")
		}

		dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", *in.Amount))
		if err != nil {
			return apperr.InternalErr("invalid amount format", err)
		}
		set["amount"] = dec
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
		set["category_id"] = in.CategoryID
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
	userID, id primitive.ObjectID,
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
