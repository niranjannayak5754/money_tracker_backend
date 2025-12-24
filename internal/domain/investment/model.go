package investment

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Type string

const (
	TypeSIP          Type = "sip"
	TypeLiquidFund   Type = "liquid_fund"
	TypeFixedDeposit Type = "fixed_deposit"
	TypeGold         Type = "gold"
	TypeSilver       Type = "silver"
	TypeStock        Type = "stock"
)

type Model struct {
	ID         common.InvestmentID `json:"id"`
	UserID     common.UserID       `json:"user_id"`
	Type       Type                `json:"type"`
	Instrument string              `json:"instrument,omitempty"`
	Amount     float64             `json:"amount"`
	Date       time.Time           `json:"date"`
	Notes      string              `json:"notes,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
	DeletedAt  *time.Time          `json:"deleted_at,omitempty"`
	DeletedBy  *common.UserID      `json:"deleted_by,omitempty"`
}
