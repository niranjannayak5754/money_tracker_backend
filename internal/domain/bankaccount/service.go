package bankaccount

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Service interface {
	Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error)
	List(ctx context.Context, userID common.UserID) ([]Model, error)
	Update(ctx context.Context, userID common.UserID, id common.BankAccountID, in UpdateInput) error
	Delete(ctx context.Context, userID common.UserID, id common.BankAccountID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type CreateInput struct {
	Name         string
	Balance      float64
	InterestRate *float64
}

type UpdateInput struct {
	Name         *string
	Balance      *float64
	InterestRate *float64
}

func (s *service) Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error) {
	if in.Name == "" {
		return Model{}, apperr.ValidationErr("name is required")
	}
	if in.Balance < 0 {
		return Model{}, apperr.ValidationErr("balance cannot be negative")
	}

	now := time.Now().UTC()
	acc := Model{
		UserID:       userID,
		Name:         in.Name,
		Balance:      in.Balance,
		InterestRate: in.InterestRate,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	id, err := s.repo.Create(ctx, acc)
	if err != nil {
		return Model{}, apperr.InternalErr("failed to create bank account", err)
	}
	acc.ID = id

	return acc, nil
}

func (s *service) List(ctx context.Context, userID common.UserID) ([]Model, error) {
	items, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, apperr.InternalErr("failed to list bank accounts", err)
	}
	if items == nil {
		return []Model{}, nil
	}
	return items, nil
}

func (s *service) Update(ctx context.Context, userID common.UserID, id common.BankAccountID, in UpdateInput) error {
	set := map[string]any{"updated_at": time.Now().UTC()}

	if in.Name != nil {
		if *in.Name == "" {
			return apperr.ValidationErr("name cannot be empty")
		}
		set["name"] = *in.Name
	}
	if in.Balance != nil {
		if *in.Balance < 0 {
			return apperr.ValidationErr("balance cannot be negative")
		}
		set["balance"] = *in.Balance
	}
	if in.InterestRate != nil {
		set["interest_rate"] = *in.InterestRate
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return apperr.InternalErr("failed to update bank account", err)
	}
	if !updated {
		return apperr.NotFoundErr("bank account not found")
	}
	return nil
}

func (s *service) Delete(ctx context.Context, userID common.UserID, id common.BankAccountID) error {
	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to delete bank account", err)
	}
	if !deleted {
		return apperr.NotFoundErr("bank account not found")
	}
	return nil
}
