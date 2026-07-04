package investment

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Model struct {
	ID              common.InvestmentID `json:"id"`
	UserID          common.UserID       `json:"user_id"`
	Type            string              `json:"type"`
	DisplayType     string              `json:"display_type"`
	Instrument      string              `json:"instrument,omitempty"`
	Amount          float64             `json:"amount"`
	Status          Status              `json:"status"`
	RetrievedAmount float64             `json:"retrieved_amount"`
	RealizedPnl     float64             `json:"realized_pnl"`
	Date            time.Time           `json:"date"`
	MaturityDate    *time.Time          `json:"maturity_date,omitempty"`
	Notes           string              `json:"notes,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	DeletedAt       *time.Time          `json:"deleted_at,omitempty"`
	DeletedBy       *common.UserID      `json:"deleted_by,omitempty"`
}

type TypeDoc struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// Status is an enum-like typed string for investment status values.
type Status string

const (
	StatusActive             Status = "ACTIVE"
	StatusClosed             Status = "CLOSED"
	StatusPartiallyRetrieved Status = "PARTIALLY_RETRIEVED"
)
