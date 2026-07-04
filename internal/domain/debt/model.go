package debt

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Status string

const (
	StatusActive Status = "ACTIVE"
	StatusClosed Status = "CLOSED"
)

// Model represents a loan/debt liability. OutstandingBalance and
// TotalInterestPaid are updated via RecordPayment as EMIs/prepayments come
// in — the caller supplies each payment's principal/interest split, since
// the app doesn't have the lender's amortization schedule.
type Model struct {
	ID                 common.DebtID  `json:"id"`
	UserID             common.UserID  `json:"user_id"`
	Name               string         `json:"name"`
	Principal          float64        `json:"principal"`
	InterestRate       float64        `json:"interest_rate"`
	EMIAmount          float64        `json:"emi_amount,omitempty"`
	TenureMonths       int            `json:"tenure_months,omitempty"`
	StartDate          time.Time      `json:"start_date"`
	OutstandingBalance float64        `json:"outstanding_balance"`
	TotalInterestPaid  float64        `json:"total_interest_paid"`
	Status             Status         `json:"status"`
	Notes              string         `json:"notes,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          *time.Time     `json:"deleted_at,omitempty"`
	DeletedBy          *common.UserID `json:"deleted_by,omitempty"`
}
