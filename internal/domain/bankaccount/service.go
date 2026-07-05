package bankaccount

import (
	"context"
	"errors"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

type Service interface {
	Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error)
	List(ctx context.Context, userID common.UserID) ([]Model, error)
	Delete(ctx context.Context, userID common.UserID, id common.BankAccountID) error
	Adjust(ctx context.Context, userID common.UserID, id common.BankAccountID, in AdjustInput) (Model, error)
	ListLedger(ctx context.Context, userID common.UserID, id common.BankAccountID, month string) ([]LedgerEntry, error)
	BalanceHistory(ctx context.Context, userID common.UserID, id common.BankAccountID, months int) ([]MonthlyBalance, error)
}

// MonthlyBalance is an account's closing balance at the end of one month —
// the balance_after of its last ledger entry on or before that month, or
// the previous month's closing balance carried forward if nothing moved
// that month. Zero for any month before the account existed.
type MonthlyBalance struct {
	Month   string  `json:"month"`
	Balance float64 `json:"balance"`
}

// AdjustDirection is which way an Adjust call moves an account's balance.
type AdjustDirection string

const (
	AdjustAdd      AdjustDirection = "add"
	AdjustWithdraw AdjustDirection = "withdraw"
)

type AdjustInput struct {
	Direction AdjustDirection
	Amount    float64
	Note      string
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type CreateInput struct {
	Name         string
	Balance      float64
	InterestRate *float64
}

func (s *service) Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error) {
	if in.Name == "" {
		return Model{}, apperr.ValidationErr("name is required")
	}
	if in.Balance < 0 {
		return Model{}, apperr.ValidationErr("balance cannot be negative")
	}

	now := time.Now().UTC()
	acc := Model{
		UserID:       userID,
		Name:         in.Name,
		Balance:      in.Balance,
		InterestRate: in.InterestRate,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	id, err := s.repo.Create(ctx, acc)
	if err != nil {
		return Model{}, apperr.InternalErr("failed to create bank account", err)
	}
	acc.ID = id

	return acc, nil
}

func (s *service) List(ctx context.Context, userID common.UserID) ([]Model, error) {
	items, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, apperr.InternalErr("failed to list bank accounts", err)
	}
	if items == nil {
		return []Model{}, nil
	}
	return items, nil
}

func (s *service) Delete(ctx context.Context, userID common.UserID, id common.BankAccountID) error {
	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to delete bank account", err)
	}
	if !deleted {
		return apperr.NotFoundErr("bank account not found")
	}
	return nil
}

// Adjust moves money in or out of a savings account balance — a manual
// add/withdraw, not a transaction-linked ledger. The balance is updated via
// an atomic increment (see Repository.Adjust) rather than a read-then-write,
// so two concurrent withdrawals can't both pass a balance check and
// overdraw the account.
func (s *service) Adjust(ctx context.Context, userID common.UserID, id common.BankAccountID, in AdjustInput) (Model, error) {
	if in.Amount <= 0 {
		return Model{}, apperr.ValidationErr("amount must be greater than zero")
	}

	var delta float64
	switch in.Direction {
	case AdjustAdd:
		delta = in.Amount
	case AdjustWithdraw:
		delta = -in.Amount
	default:
		return Model{}, apperr.ValidationErr("direction must be add or withdraw")
	}

	acc, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Model{}, apperr.NotFoundErr("bank account not found")
		}
		return Model{}, apperr.InternalErr("failed to load bank account", err)
	}
	if delta < 0 && acc.Balance+delta < 0 {
		return Model{}, apperr.ValidationErr("insufficient balance for withdrawal")
	}

	newBalance, err := s.repo.Adjust(ctx, userID, id, delta, in.Note)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Model{}, apperr.ValidationErr("insufficient balance for withdrawal")
		}
		return Model{}, apperr.InternalErr("failed to adjust bank account balance", err)
	}

	acc.Balance = newBalance
	acc.UpdatedAt = time.Now().UTC()
	return *acc, nil
}

// ListLedger returns an account's ledger entries, optionally scoped to a
// single month — the opening entry (from Create) plus every Adjust since.
func (s *service) ListLedger(ctx context.Context, userID common.UserID, id common.BankAccountID, month string) ([]LedgerEntry, error) {
	start, end, err := shared.MonthRange(month)
	if err != nil {
		return nil, err
	}

	entries, err := s.repo.ListLedger(ctx, userID, id, start, end)
	if err != nil {
		return nil, apperr.InternalErr("failed to list bank account ledger", err)
	}
	if entries == nil {
		return []LedgerEntry{}, nil
	}
	return entries, nil
}

// BalanceHistory returns the account's closing balance for each of the
// last `months` calendar months (oldest first), derived entirely from its
// own ledger — independent of expenses or any other account.
func (s *service) BalanceHistory(ctx context.Context, userID common.UserID, id common.BankAccountID, months int) ([]MonthlyBalance, error) {
	if months <= 0 {
		months = 3
	}
	if months > config.TWELVE {
		return nil, apperr.ValidationErr("months cannot be greater than 12")
	}

	entries, err := s.repo.ListLedger(ctx, userID, id, time.Time{}, time.Time{})
	if err != nil {
		return nil, apperr.InternalErr("failed to list bank account ledger", err)
	}

	now := time.Now().UTC()
	out := make([]MonthlyBalance, 0, months)
	entryIdx := 0
	var closingBalance float64

	for i := months - 1; i >= 0; i-- {
		t := now.AddDate(0, -i, 0)
		monthStart := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		monthEnd := monthStart.AddDate(0, 1, 0)

		for entryIdx < len(entries) && entries[entryIdx].CreatedAt.Before(monthEnd) {
			closingBalance = entries[entryIdx].BalanceAfter
			entryIdx++
		}

		out = append(out, MonthlyBalance{
			Month:   monthStart.Format(config.STANDARD_YEAR_MONTH),
			Balance: closingBalance,
		})
	}

	return out, nil
}
