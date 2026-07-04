package debt

import (
	"context"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, m Model) (common.DebtID, error)
	List(ctx context.Context, userID common.UserID) ([]Model, error)
	GetByID(ctx context.Context, userID common.UserID, id common.DebtID) (*Model, error)
	Update(ctx context.Context, userID common.UserID, id common.DebtID, set map[string]any) (bool, error)
	Delete(ctx context.Context, userID common.UserID, id common.DebtID) (bool, error)
}
