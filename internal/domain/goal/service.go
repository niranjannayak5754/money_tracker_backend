package goal

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Service interface {
	Create(ctx context.Context, userID common.UserID, in CreateInput) (WithProgress, error)
	List(ctx context.Context, userID common.UserID) ([]WithProgress, error)
	Update(ctx context.Context, userID common.UserID, id common.GoalID, in UpdateInput) error
	Delete(ctx context.Context, userID common.UserID, id common.GoalID) error
}

type service struct {
	repo         Repository
	bankAccounts BankAccountRepository
	investments  InvestmentRepository
}

func NewService(repo Repository, bankAccounts BankAccountRepository, investments InvestmentRepository) Service {
	return &service{repo: repo, bankAccounts: bankAccounts, investments: investments}
}

type CreateInput struct {
	Name         string
	TargetAmount float64
	TargetDate   *time.Time
	LinkedType   LinkedType
	LinkedID     string
}

type UpdateInput struct {
	Name         *string
	TargetAmount *float64
	TargetDate   *time.Time
	Status       *Status
}

func (s *service) Create(ctx context.Context, userID common.UserID, in CreateInput) (WithProgress, error) {
	if in.Name == "" {
		return WithProgress{}, apperr.ValidationErr("name is required")
	}
	if in.TargetAmount <= 0 {
		return WithProgress{}, apperr.ValidationErr("target_amount must be greater than zero")
	}
	if in.LinkedType != LinkedBankAccount && in.LinkedType != LinkedInvestment {
		return WithProgress{}, apperr.ValidationErr("linked_type must be bank_account or investment")
	}
	if in.LinkedID == "" {
		return WithProgress{}, apperr.ValidationErr("linked_id is required")
	}

	now := time.Now().UTC()
	g := Model{
		UserID:       userID,
		Name:         in.Name,
		TargetAmount: in.TargetAmount,
		TargetDate:   in.TargetDate,
		LinkedType:   in.LinkedType,
		LinkedID:     in.LinkedID,
		Status:       StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	id, err := s.repo.Create(ctx, g)
	if err != nil {
		return WithProgress{}, apperr.InternalErr("failed to create goal", err)
	}
	g.ID = id

	return s.resolveProgress(ctx, g), nil
}

func (s *service) List(ctx context.Context, userID common.UserID) ([]WithProgress, error) {
	items, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, apperr.InternalErr("failed to list goals", err)
	}

	out := make([]WithProgress, 0, len(items))
	for _, g := range items {
		out = append(out, s.resolveProgress(ctx, g))
	}
	return out, nil
}

func (s *service) Update(ctx context.Context, userID common.UserID, id common.GoalID, in UpdateInput) error {
	set := map[string]any{"updated_at": time.Now().UTC()}

	if in.Name != nil {
		if *in.Name == "" {
			return apperr.ValidationErr("name cannot be empty")
		}
		set["name"] = *in.Name
	}
	if in.TargetAmount != nil {
		if *in.TargetAmount <= 0 {
			return apperr.ValidationErr("target_amount must be greater than zero")
		}
		set["target_amount"] = *in.TargetAmount
	}
	if in.TargetDate != nil {
		set["target_date"] = *in.TargetDate
	}
	if in.Status != nil {
		if *in.Status != StatusActive && *in.Status != StatusCompleted && *in.Status != StatusAbandoned {
			return apperr.ValidationErr("invalid status")
		}
		set["status"] = string(*in.Status)
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return apperr.InternalErr("failed to update goal", err)
	}
	if !updated {
		return apperr.NotFoundErr("goal not found")
	}
	return nil
}

func (s *service) Delete(ctx context.Context, userID common.UserID, id common.GoalID) error {
	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to delete goal", err)
	}
	if !deleted {
		return apperr.NotFoundErr("goal not found")
	}
	return nil
}

// resolveProgress reads the linked entity's current value at read time —
// there's no separate progress ledger to keep in sync. A missing/deleted
// linked entity degrades to zero progress with LinkedEntityMissing set,
// rather than erroring the whole list/create call.
func (s *service) resolveProgress(ctx context.Context, g Model) WithProgress {
	wp := WithProgress{Model: g}

	switch g.LinkedType {
	case LinkedBankAccount:
		acc, err := s.bankAccounts.GetByID(ctx, g.UserID, common.BankAccountID(g.LinkedID))
		if err != nil {
			wp.LinkedEntityMissing = true
			return wp
		}
		wp.CurrentValue = acc.Balance
	case LinkedInvestment:
		inv, err := s.investments.GetByID(ctx, g.UserID, common.InvestmentID(g.LinkedID))
		if err != nil {
			wp.LinkedEntityMissing = true
			return wp
		}
		wp.CurrentValue = inv.Amount + inv.RetrievedAmount
	}

	if g.TargetAmount > 0 {
		pct := wp.CurrentValue / g.TargetAmount * 100
		if pct > 100 {
			pct = 100
		}
		if pct < 0 {
			pct = 0
		}
		wp.PercentComplete = pct
	}

	return wp
}
