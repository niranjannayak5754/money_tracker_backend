package expense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

var (
	ErrInvalidAmount   = errors.New("amount must be greater than zero")
	ErrInvalidCategory = errors.New("invalid category")
	ErrDecodeAmount    = errors.New("invalid amount format")
	ErrInvalidInput    = errors.New("invalid input")
)

type Model struct {
	ID         primitive.ObjectID   `bson:"_id" json:"id"`
	UserID     primitive.ObjectID   `bson:"user_id" json:"user_id"`
	Amount     primitive.Decimal128 `bson:"amount" json:"-"`
	AmountF    float64              `bson:"-" json:"amount"`
	Date       time.Time            `bson:"date" json:"date"`
	CategoryID primitive.ObjectID   `bson:"category_id" json:"category_id"`
	Merchant   string               `bson:"merchant,omitempty" json:"merchant,omitempty"`
	Notes      string               `bson:"notes,omitempty" json:"notes,omitempty"`
	Tags       []string             `bson:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt  time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time            `bson:"updated_at" json:"updated_at"`
}

//
// REPOSITORY INTERFACES
//

// Provided by category module
type CategoryRepo interface {
	ExistsForUser(ctx any, userID, categoryID primitive.ObjectID) (bool, error)
}

// Expense repository
type Repository interface {
	Create(ctx any, m Model) error
	ListMonth(ctx any, userID primitive.ObjectID, start, end time.Time, category *primitive.ObjectID) ([]Model, error)
	Update(ctx any, userID, id primitive.ObjectID, set map[string]any) (bool, error)
	Delete(ctx any, userID, id primitive.ObjectID) (bool, error)
}

//
// SERVICE INTERFACE (domain-only)
//

type Service interface {
	Create(ctx context.Context, uid primitive.ObjectID, input CreateInput) (Model, error)
	List(ctx context.Context, uid primitive.ObjectID, month string, category string) ([]Model, error)
	Update(ctx context.Context, uid, id primitive.ObjectID, input UpdateInput) (bool, error)
	Delete(ctx context.Context, uid, id primitive.ObjectID) (bool, error)
}

type service struct {
	repo Repository
	cats CategoryRepo
}

func NewService(repo Repository, cats CategoryRepo) Service {
	return &service{repo: repo, cats: cats}
}

//
// INPUT DTOs
//

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

//
// BUSINESS LOGIC
//

func (s *service) Create(ctx context.Context, uid primitive.ObjectID, in CreateInput) (Model, error) {
	if in.Amount <= 0 {
		return Model{}, ErrInvalidAmount
	}

	// Validate category belongs to user
	ok, err := s.cats.ExistsForUser(ctx, uid, in.CategoryID)
	if err != nil {
		return Model{}, err
	}
	if !ok {
		return Model{}, ErrInvalidCategory
	}

	amountDec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", in.Amount))
	if err != nil {
		return Model{}, ErrDecodeAmount
	}

	exp := Model{
		ID:         primitive.NewObjectID(),
		UserID:     uid,
		Amount:     amountDec,
		AmountF:    in.Amount,
		Date:       shared.ChooseDate(in.Date),
		CategoryID: in.CategoryID,
		Merchant:   in.Merchant,
		Notes:      in.Notes,
		Tags:       in.Tags,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, exp); err != nil {
		return Model{}, err
	}

	return exp, nil
}

func (s *service) List(ctx context.Context, uid primitive.ObjectID, month string, category string) ([]Model, error) {
	start, end := shared.MonthRange(month)

	var catID *primitive.ObjectID
	if category != "" {
		id, err := primitive.ObjectIDFromHex(category)
		if err == nil {
			catID = &id
		}
	}

	items, err := s.repo.ListMonth(ctx, uid, start, end, catID)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].AmountF = shared.Decimal128ToFloat(items[i].Amount)
	}

	return items, nil
}

func (s *service) Update(ctx context.Context, uid, id primitive.ObjectID, in UpdateInput) (bool, error) {
	set := map[string]any{"updated_at": time.Now()}

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

	if in.CategoryID != nil {
		set["category_id"] = *in.CategoryID
	}

	if in.Merchant != nil {
		set["merchant"] = *in.Merchant
	}

	if in.Notes != nil {
		set["notes"] = *in.Notes
	}

	if in.Tags != nil {
		set["tags"] = *in.Tags
	}

	return s.repo.Update(ctx, uid, id, set)
}

func (s *service) Delete(ctx context.Context, uid, id primitive.ObjectID) (bool, error) {
	return s.repo.Delete(ctx, uid, id)
}
