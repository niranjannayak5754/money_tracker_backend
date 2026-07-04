package budget

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

type Service interface {
	Set(ctx context.Context, userID common.UserID, in SetInput) (Model, error)
	List(ctx context.Context, userID common.UserID) ([]Model, error)
	Delete(ctx context.Context, userID common.UserID, id common.BudgetID) error

	// ResolveForMonth returns the effective budget per category (and the
	// overall cap, if any) as of the given month.
	ResolveForMonth(ctx context.Context, userID common.UserID, month string) (Resolved, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type SetInput struct {
	CategoryID    *common.CategoryID
	Amount        float64
	EffectiveFrom string
}

// Resolved is the effective budget as of a given month.
type Resolved struct {
	Overall    *float64
	ByCategory map[common.CategoryID]float64
}

func (s *service) Set(ctx context.Context, userID common.UserID, in SetInput) (Model, error) {
	if in.Amount <= 0 {
		return Model{}, apperr.ValidationErr("amount must be greater than zero")
	}
	if _, _, err := shared.MonthBounds(in.EffectiveFrom); err != nil {
		return Model{}, err
	}

	now := time.Now().UTC()
	m := Model{
		UserID:        userID,
		CategoryID:    in.CategoryID,
		Amount:        in.Amount,
		EffectiveFrom: in.EffectiveFrom,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	id, err := s.repo.Set(ctx, m)
	if err != nil {
		return Model{}, apperr.InternalErr("failed to set budget", err)
	}
	m.ID = id

	return m, nil
}

func (s *service) List(ctx context.Context, userID common.UserID) ([]Model, error) {
	items, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, apperr.InternalErr("failed to list budgets", err)
	}
	if items == nil {
		return []Model{}, nil
	}
	return items, nil
}

func (s *service) Delete(ctx context.Context, userID common.UserID, id common.BudgetID) error {
	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to delete budget", err)
	}
	if !deleted {
		return apperr.NotFoundErr("budget not found")
	}
	return nil
}

func (s *service) ResolveForMonth(ctx context.Context, userID common.UserID, month string) (Resolved, error) {
	if _, _, err := shared.MonthBounds(month); err != nil {
		return Resolved{}, err
	}

	rows, err := s.repo.List(ctx, userID)
	if err != nil {
		return Resolved{}, apperr.InternalErr("failed to load budgets", err)
	}

	return resolve(rows, month), nil
}

// resolve picks, per category (and separately for the overall cap keyed by
// a nil CategoryID), the row with the latest EffectiveFrom that is still
// <= month — not simply the most recently created row. "YYYY-MM" strings
// compare correctly with plain string comparison.
func resolve(rows []Model, month string) Resolved {
	out := Resolved{ByCategory: map[common.CategoryID]float64{}}

	var overallEffFrom string
	categoryEffFrom := map[common.CategoryID]string{}

	for _, r := range rows {
		if r.EffectiveFrom > month {
			continue
		}

		if r.CategoryID == nil {
			if out.Overall == nil || r.EffectiveFrom > overallEffFrom {
				amt := r.Amount
				out.Overall = &amt
				overallEffFrom = r.EffectiveFrom
			}
			continue
		}

		cid := *r.CategoryID
		if existing, ok := categoryEffFrom[cid]; !ok || r.EffectiveFrom > existing {
			out.ByCategory[cid] = r.Amount
			categoryEffFrom[cid] = r.EffectiveFrom
		}
	}

	return out
}
