package notification

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Type string

const (
	TypeBillDue        Type = "bill_due"
	TypeFDMaturity     Type = "fd_maturity"
	TypeBudgetExceeded Type = "budget_exceeded"
)

// Model is an in-app notification — surfaced only through this API, never
// emailed/pushed/texted. RelatedEntityType/RelatedEntityID identify what
// triggered it (e.g. "category"/"<id>:<month>", "investment"/"<id>") and
// double as the dedup key so the same situation doesn't re-notify on every
// tick or every subsequent expense.
type Model struct {
	ID                common.NotificationID `json:"id"`
	UserID            common.UserID         `json:"user_id"`
	Type              Type                  `json:"type"`
	Title             string                `json:"title"`
	Message           string                `json:"message"`
	RelatedEntityType string                `json:"related_entity_type,omitempty"`
	RelatedEntityID   string                `json:"related_entity_id,omitempty"`
	CreatedAt         time.Time             `json:"created_at"`
	ReadAt            *time.Time            `json:"read_at,omitempty"`
	DismissedAt       *time.Time            `json:"dismissed_at,omitempty"`
}
