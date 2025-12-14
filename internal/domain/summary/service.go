package summary

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

type Repository interface {
	IncomeTotal(ctx context.Context, uid primitive.ObjectID, start, end time.Time) (float64, error)
	ExpenseTotals(ctx context.Context, uid primitive.ObjectID, start, end time.Time) (float64, []map[string]any, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, uid primitive.ObjectID, month string) (map[string]any, error) {
	start, end := shared.MonthRange(month)

	incomeTotal, err := s.repo.IncomeTotal(ctx, uid, start, end)
	if err != nil {
		return nil, err
	}

	expTotal, catBreakdown, err := s.repo.ExpenseTotals(ctx, uid, start, end)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"income_total":       incomeTotal,
		"expense_total":      expTotal,
		"savings":            incomeTotal - expTotal,
		"category_breakdown": catBreakdown,
	}, nil
}
