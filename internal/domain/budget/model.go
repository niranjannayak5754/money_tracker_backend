package budget

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

// Model is one effective-dated budget row. CategoryID nil means an overall
// spend cap rather than a per-category one. Setting a new EffectiveFrom
// creates a new row rather than overwriting the old one, so budget history
// is preserved and past months keep resolving against what was in effect
// at the time.
type Model struct {
	ID            common.BudgetID    `json:"id"`
	UserID        common.UserID      `json:"user_id"`
	CategoryID    *common.CategoryID `json:"category_id,omitempty"`
	Amount        float64            `json:"amount"`
	EffectiveFrom string             `json:"effective_from"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}
