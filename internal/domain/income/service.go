package income

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

var (
	ErrInvalidAmount = errors.New("amount must be greater than zero")
	ErrDecodeAmount  = errors.New("invalid amount format")
)

type Model struct {
	ID        primitive.ObjectID   `bson:"_id" json:"id"`
	UserID    primitive.ObjectID   `bson:"user_id" json:"user_id"`
	Amount    primitive.Decimal128 `bson:"amount" json:"-"`
	AmountF   float64              `bson:"-" json:"amount"`
	Date      time.Time            `bson:"date" json:"date"`
	Source    string               `bson:"source,omitempty" json:"source,omitempty"`
	Notes     string               `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time            `bson:"updated_at" json:"updated_at"`
}

//
// REPOSITORY
//

type Repository interface {
	Create(ctx any, m Model) error
	ListMonth(ctx any, uid primitive.ObjectID, start, end time.Time) ([]Model, error)
	Update(ctx any, uid, id primitive.ObjectID, set map[string]any) (bool, error)
	Delete(ctx any, uid, id primitive.ObjectID) (bool, error)
}

//
// SERVICE INTERFACE (domain only)
//

type Service interface {
	Create(ctx context.Context, userID primitive.ObjectID, input CreateInput) (Model, error)
	List(ctx context.Context, userID primitive.ObjectID, month string) ([]Model, error)
	Update(ctx context.Context, userID, id primitive.ObjectID, input UpdateInput) (bool, error)
	Delete(ctx context.Context, userID, id primitive.ObjectID) (bool, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

//
// INPUT DTOs
//

type CreateInput struct {
	Amount float64
	Date   time.Time
	Source string
	Notes  string
}

type UpdateInput struct {
	Amount *float64
	Date   *time.Time
	Source *string
	Notes  *string
}

//
// BUSINESS LOGIC
//

func (s *service) Create(ctx context.Context, uid primitive.ObjectID, in CreateInput) (Model, error) {
	if in.Amount <= 0 {
		return Model{}, ErrInvalidAmount
	}

	dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", in.Amount))
	if err != nil {
		return Model{}, ErrDecodeAmount
	}

	rec := Model{
		ID:        primitive.NewObjectID(),
		UserID:    uid,
		Amount:    dec,
		AmountF:   in.Amount,
		Date:      shared.ChooseDate(in.Date),
		Source:    in.Source,
		Notes:     in.Notes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, rec); err != nil {
		return Model{}, err
	}

	return rec, nil
}

func (s *service) List(ctx context.Context, uid primitive.ObjectID, month string) ([]Model, error) {
	start, end := shared.MonthRange(month)

	items, err := s.repo.ListMonth(ctx, uid, start, end)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].AmountF = shared.Decimal128ToFloat(items[i].Amount)
	}

	return items, nil
}

func (s *service) Update(ctx context.Context, uid, id primitive.ObjectID, in UpdateInput) (bool, error) {
	set := map[string]any{
		"updated_at": time.Now(),
	}

	if in.Amount != nil {
		dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", *in.Amount))
		if err != nil {
			return false, ErrDecodeAmount
		}
		set["amount"] = dec
	}

	if in.Date != nil {
		set["date"] = *in.Date
	}

	if in.Source != nil {
		set["source"] = *in.Source
	}

	if in.Notes != nil {
		set["notes"] = *in.Notes
	}

	return s.repo.Update(ctx, uid, id, set)
}

func (s *service) Delete(ctx context.Context, uid, id primitive.ObjectID) (bool, error) {
	return s.repo.Delete(ctx, uid, id)
}
