package category

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/ptr"
)

// SERVICE INTERFACE
type Service interface {
	List(ctx context.Context, userID common.UserID) ([]Model, error)
	ListActive(ctx context.Context, userID common.UserID) ([]Model, error)
	ListArchived(ctx context.Context, userID common.UserID) ([]Model, error)

	Create(
		ctx context.Context,
		userID common.UserID,
		in CreateInput,
	) (Model, error)

	Update(
		ctx context.Context,
		userID common.UserID,
		categoryID common.CategoryID,
		in UpdateInput,
	) error
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

// List returns all categories for a user
func (s *service) List(
	ctx context.Context,
	userID common.UserID,
) ([]Model, error) {

	userCategories, err := s.repo.List(ctx, userID, nil)
	if err != nil {
		return nil, apperr.InternalErr("failed to list categories", err)
	}

	return userCategories, nil
}

func (s *service) ListActive(
	ctx context.Context,
	userID common.UserID,
) ([]Model, error) {

	userCategories, err := s.repo.List(ctx, userID, ptr.Bool(false))
	if err != nil {
		return nil, apperr.InternalErr("failed to list active categories", err)
	}

	return userCategories, nil
}

func (s *service) ListArchived(
	ctx context.Context,
	userID common.UserID,
) ([]Model, error) {

	userCategories, err := s.repo.List(ctx, userID, ptr.Bool(true))
	if err != nil {
		return nil, apperr.InternalErr("failed to list archived categories", err)
	}

	return userCategories, nil
}

// Create creates a new category with validation
func (s *service) Create(
	ctx context.Context,
	userID common.UserID,
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
		ID:        "", // ID will be set by the repository
		UserID:    userID,
		Name:      in.Name,
		Type:      kind,
		Archived:  false,
		CreatedAt: time.Now().UTC(),
	}

	id, err := s.repo.Create(ctx, cat)
	if err != nil {
		return Model{}, apperr.InternalErr(
			"failed to create category",
			err,
		)
	}
	cat.ID = id

	return cat, nil
}

// Update updates name and/or archived flag for a category
func (s *service) Update(
	ctx context.Context,
	userID common.UserID,
	categoryID common.CategoryID,
	in UpdateInput,
) error {

	set := map[string]any{}

	if in.Name != nil {
		if *in.Name == "" {
			return apperr.ValidationErr("category name cannot be empty")
		}
		set["name"] = *in.Name
	}

	if in.Archived != nil {
		set["archived"] = *in.Archived
	}

	if len(set) == 0 {
		return apperr.ValidationErr("no fields to update")
	}

	ok, err := s.repo.Update(ctx, userID, categoryID, set)
	if err != nil {
		return apperr.InternalErr("failed to update category", err)
	}

	if !ok {
		return apperr.NotFoundErr("category not found")
	}

	return nil
}
