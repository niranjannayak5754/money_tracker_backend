package bankaccount

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

// Model represents a cash/bank holding. Balance is entered/updated manually
// by the user — there is no transaction-linked ledger in v1.
type Model struct {
	ID           common.BankAccountID `json:"id"`
	UserID       common.UserID        `json:"user_id"`
	Name         string               `json:"name"`
	Balance      float64              `json:"balance"`
	InterestRate *float64             `json:"interest_rate,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
	DeletedAt    *time.Time           `json:"deleted_at,omitempty"`
	DeletedBy    *common.UserID       `json:"deleted_by,omitempty"`
}
