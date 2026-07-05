package bankaccount

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

// Model represents a savings account. Balance only ever changes via the
// initial Create balance or an Adjust (add/withdraw) call — never a direct
// edit — so every change is captured as a LedgerEntry.
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

type LedgerEntryType string

const (
	LedgerOpening  LedgerEntryType = "opening"
	LedgerAdd      LedgerEntryType = "add"
	LedgerWithdraw LedgerEntryType = "withdraw"
)

// LedgerEntry is one dated, immutable balance change for a bank account —
// the account's initial balance (LedgerOpening) plus every subsequent
// Adjust call. Amount is always positive; direction comes from Type.
type LedgerEntry struct {
	ID            common.LedgerEntryID `json:"id"`
	UserID        common.UserID        `json:"user_id"`
	BankAccountID common.BankAccountID `json:"bank_account_id"`
	Type          LedgerEntryType      `json:"type"`
	Amount        float64              `json:"amount"`
	Note          string               `json:"note,omitempty"`
	BalanceAfter  float64              `json:"balance_after"`
	CreatedAt     time.Time            `json:"created_at"`
}
