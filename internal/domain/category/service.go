package category

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrInvalidName = errors.New("category name is required")
	ErrInvalidType = errors.New("invalid category type")
)

type Model struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Name      string             `bson:"name" json:"name"`
	Type      string             `bson:"type" json:"type"` // expense|income
	Archived  bool               `bson:"archived" json:"archived"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type Repository interface {
	Create(ctx any, m Model) error
	Update(ctx any, userID, id primitive.ObjectID, set map[string]any) (bool, error)
	ListActive(ctx any, userID primitive.ObjectID) ([]Model, error)
	ExistsForUser(rctx any, uid, categoryID primitive.ObjectID) (bool, error)
}

// SERVICE INTERFACE (business logic only)
type Service interface {
	List(ctx context.Context, userID primitive.ObjectID) ([]Model, error)
	Create(ctx context.Context, userID primitive.ObjectID, input CreateInput) (Model, error)
	Update(ctx context.Context, userID, categoryID primitive.ObjectID, input UpdateInput) (bool, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

//
// INPUT STRUCTS
//

type CreateInput struct {
	Name string
	Type string // optional, default "expense"
}

type UpdateInput struct {
	Name     *string
	Archived *bool
}

//
// BUSINESS LOGIC
//

func (s *service) List(ctx context.Context, userID primitive.ObjectID) ([]Model, error) {
	return s.repo.ListActive(ctx, userID)
}

func (s *service) Create(ctx context.Context, userID primitive.ObjectID, in CreateInput) (Model, error) {
	if in.Name == "" {
		return Model{}, ErrInvalidName
	}

	kind := in.Type
	if kind == "" {
		kind = "expense"
	}

	if kind != "expense" && kind != "income" {
		return Model{}, ErrInvalidType
	}

	cat := Model{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Name:      in.Name,
		Type:      kind,
		Archived:  false,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, cat); err != nil {
		return Model{}, err
	}

	return cat, nil
}

func (s *service) Update(ctx context.Context, userID, id primitive.ObjectID, in UpdateInput) (bool, error) {
	set := map[string]any{}

	if in.Name != nil {
		if *in.Name == "" {
			return false, ErrInvalidName
		}
		set["name"] = *in.Name
	}

	if in.Archived != nil {
		set["archived"] = *in.Archived
	}

	return s.repo.Update(ctx, userID, id, set)
}
