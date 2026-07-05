package recurring

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type EntityType string

const (
	EntityExpense EntityType = "expense"
)

type Frequency string

const (
	FrequencyMonthly Frequency = "monthly"
	FrequencyYearly  Frequency = "yearly"
)

// Model is a recurring-transaction template. Payload carries whatever
// fields the target entity's Create needs (e.g. amount/category_id for an
// expense) — the scheduler materializes it through the normal, validated
// expense Create path, it never writes that collection directly.
type Model struct {
	ID          common.RecurringID `json:"id"`
	UserID      common.UserID      `json:"user_id"`
	EntityType  EntityType         `json:"entity_type"`
	Frequency   Frequency          `json:"frequency"`
	DayOfMonth  int                `json:"day_of_month"`
	Payload     map[string]any     `json:"payload"`
	StartDate   time.Time          `json:"start_date"`
	EndDate     *time.Time         `json:"end_date,omitempty"`
	NextRunDate time.Time          `json:"next_run_date"`
	LastRunDate *time.Time         `json:"last_run_date,omitempty"`
	Active      bool               `json:"active"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	DeletedAt   *time.Time         `json:"deleted_at,omitempty"`
	DeletedBy   *common.UserID     `json:"deleted_by,omitempty"`
}
