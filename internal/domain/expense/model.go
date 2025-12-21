package expense

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Model struct {
	ID         common.ExpenseID  `json:"id"`
	UserID     common.UserID     `json:"user_id"`
	Amount     float64           `json:"amount"`
	Date       time.Time         `json:"date"`
	CategoryID common.CategoryID `json:"category_id"`
	Merchant   string            `json:"merchant,omitempty"`
	Notes      string            `json:"notes,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}
