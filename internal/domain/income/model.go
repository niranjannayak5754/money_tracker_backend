package income

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Model struct {
	ID         common.IncomeID   `json:"id"`
	UserID     common.UserID     `json:"user_id"`
	Amount     float64           `json:"amount"`
	Date       time.Time         `json:"date"`
	CategoryID common.CategoryID `json:"category_id"`
	Source     string            `json:"source,omitempty"`
	Notes      string            `json:"notes,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	DeletedAt  *time.Time        `json:"deleted_at,omitempty"`
	DeletedBy  *common.UserID    `json:"deleted_by,omitempty"`
}
