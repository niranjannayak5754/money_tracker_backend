package category

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/ptr"
)

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

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
	repo     Repository
	expenses ExpenseRepository
}

func NewService(repo Repository, expenses ExpenseRepository) Service {
	return &service{repo: repo, expenses: expenses}
}

// INPUT STRUCTS
type CreateInput struct {
	Name  string
	Type  string // expense | income (optional, default expense)
	Color string // optional, hex like #RRGGBB
	Icon  string // optional, freeform icon key/emoji
}

type UpdateInput struct {
	Name     *string
	Color    *string
	Icon     *string
	Archived *bool

	// ReassignTo, if set alongside Archived=true, moves any expenses
	// referencing this category to ReassignTo before archiving instead of
	// blocking the archive.
	ReassignTo *common.CategoryID
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

	if in.Color != "" && !hexColorPattern.MatchString(in.Color) {
		return Model{}, apperr.ValidationErr("color must be a hex value like #RRGGBB")
	}

	cat := Model{
		ID:        "", // ID will be set by the repository
		UserID:    userID,
		Name:      in.Name,
		Type:      kind,
		Color:     in.Color,
		Icon:      in.Icon,
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

	if in.Color != nil {
		if *in.Color != "" && !hexColorPattern.MatchString(*in.Color) {
			return apperr.ValidationErr("color must be a hex value like #RRGGBB")
		}
		set["color"] = *in.Color
	}

	if in.Icon != nil {
		set["icon"] = *in.Icon
	}

	if in.Archived != nil {
		if *in.Archived {
			if err := s.prepareArchive(ctx, userID, categoryID, in.ReassignTo); err != nil {
				return err
			}
		}
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

// prepareArchive blocks archiving a category that still has expenses
// pointing at it, unless reassignTo is given — in which case those
// expenses are moved to reassignTo first.
func (s *service) prepareArchive(
	ctx context.Context,
	userID common.UserID,
	categoryID common.CategoryID,
	reassignTo *common.CategoryID,
) error {
	if reassignTo != nil {
		if *reassignTo == categoryID {
			return apperr.ValidationErr("cannot reassign a category's expenses to itself")
		}

		ok, err := s.repo.ExistsForUser(ctx, userID, *reassignTo)
		if err != nil {
			return apperr.InternalErr("failed to validate reassign target category", err)
		}
		if !ok {
			return apperr.ValidationErr("invalid reassign_to category")
		}

		if _, err := s.expenses.ReassignCategory(ctx, userID, categoryID, *reassignTo); err != nil {
			return apperr.InternalErr("failed to reassign expenses", err)
		}

		return nil
	}

	count, err := s.expenses.CountByCategory(ctx, userID, categoryID)
	if err != nil {
		return apperr.InternalErr("failed to check category usage", err)
	}
	if count > 0 {
		return apperr.ValidationErr(fmt.Sprintf(
			"cannot archive: %d expense(s) still reference this category; pass reassign_to to move them first",
			count,
		))
	}

	return nil
}
