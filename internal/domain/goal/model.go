package goal

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type LinkedType string

const (
	LinkedBankAccount LinkedType = "bank_account"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusAbandoned Status = "abandoned"
)

// Model is a savings goal linked to an existing bank account.
// Progress isn't stored — it's resolved at read time from the linked
// entity's current value, so there's no separate ledger to keep in sync.
type Model struct {
	ID           common.GoalID  `json:"id"`
	UserID       common.UserID  `json:"user_id"`
	Name         string         `json:"name"`
	TargetAmount float64        `json:"target_amount"`
	TargetDate   *time.Time     `json:"target_date,omitempty"`
	LinkedType   LinkedType     `json:"linked_type"`
	LinkedID     string         `json:"linked_id"`
	Status       Status         `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    *time.Time     `json:"deleted_at,omitempty"`
	DeletedBy    *common.UserID `json:"deleted_by,omitempty"`
}

// WithProgress is a goal plus its resolved-at-read-time progress.
type WithProgress struct {
	Model
	CurrentValue float64 `json:"current_value"`
	// PercentComplete is capped to [0, 100] — a goal can be over-funded,
	// but "percent complete" reads oddly past full.
	PercentComplete float64 `json:"percent_complete"`
	// LinkedEntityMissing is true when the linked bank account no longer
	// exists (deleted) — progress reads as 0 rather than erroring.
	LinkedEntityMissing bool `json:"linked_entity_missing,omitempty"`
}
