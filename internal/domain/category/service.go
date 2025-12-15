package category

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
)

// SERVICE INTERFACE
type Service interface {
	List(
		ctx context.Context,
		userID primitive.ObjectID,
	) ([]Model, error)

	Create(
		ctx context.Context,
		userID primitive.ObjectID,
		in CreateInput,
	) (Model, error)

	Update(
		ctx context.Context,
		userID, categoryID primitive.ObjectID,
		in UpdateInput,
	) (bool, error)
}

// SERVICE IMPLEMENTATION (PRIVATE)
type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// INPUT STRUCTS
type CreateInput struct {
	Name string
	Type string // expense | income (optional, default expense)
}

type UpdateInput struct {
	Name     *string
	Archived *bool
}

// BUSINESS LOGIC

// List returns all active categories for a user
func (s *service) List(
	ctx context.Context,
	userID primitive.ObjectID,
) ([]Model, error) {

	return s.repo.ListActive(ctx, userID)
}

// Create creates a new category with validation
func (s *service) Create(
	ctx context.Context,
	userID primitive.ObjectID,
	in CreateInput,
) (Model, error) {

	if in.Name == "" {
		return Model{}, apperr.ValidationErr("category name is required")
	}

	kind := in.Type
	if kind == "" {
		kind = "expense"
	}

	if kind != "expense" && kind != "income" {
		return Model{}, apperr.ValidationErr("invalid category type")
	}

	cat := Model{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Name:      in.Name,
		Type:      kind,
		Archived:  false,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, cat); err != nil {
		return Model{}, apperr.InternalErr(
			"failed to create category",
			err,
		)
	}

	return cat, nil
}

// Update updates name and/or archived flag for a category
func (s *service) Update(
	ctx context.Context,
	userID, categoryID primitive.ObjectID,
	in UpdateInput,
) (bool, error) {

	set := map[string]any{}

	if in.Name != nil {
		if *in.Name == "" {
			return false, apperr.ValidationErr("category name cannot be empty")
		}
		set["name"] = *in.Name
	}

	if in.Archived != nil {
		set["archived"] = *in.Archived
	}

	if len(set) == 0 {
		return false, apperr.ValidationErr("no fields to update")
	}

	ok, err := s.repo.Update(ctx, userID, categoryID, set)
	if err != nil {
		return false, apperr.InternalErr(
			"failed to update category",
			err,
		)
	}

	return ok, nil
}
