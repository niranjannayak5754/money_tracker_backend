package bankaccount

import (
	"context"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

type fakeRepo struct {
	accounts map[common.BankAccountID]*Model
	ledgers  map[common.BankAccountID][]LedgerEntry
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{accounts: map[common.BankAccountID]*Model{}, ledgers: map[common.BankAccountID][]LedgerEntry{}}
}

func (f *fakeRepo) Create(ctx context.Context, m Model) (common.BankAccountID, error) {
	return "", nil
}
func (f *fakeRepo) List(ctx context.Context, userID common.UserID) ([]Model, error) {
	return nil, nil
}
func (f *fakeRepo) GetByID(ctx context.Context, userID common.UserID, id common.BankAccountID) (*Model, error) {
	acc, ok := f.accounts[id]
	if !ok || acc.UserID != userID {
		return nil, repository.ErrNotFound
	}
	cp := *acc
	return &cp, nil
}
func (f *fakeRepo) Delete(ctx context.Context, userID common.UserID, id common.BankAccountID) (bool, error) {
	return false, nil
}
func (f *fakeRepo) BalanceTotal(ctx context.Context, userID common.UserID) (float64, error) {
	return 0, nil
}
func (f *fakeRepo) ListLedger(ctx context.Context, userID common.UserID, id common.BankAccountID, start, end time.Time) ([]LedgerEntry, error) {
	all := f.ledgers[id]
	if start.IsZero() && end.IsZero() {
		return all, nil
	}
	var out []LedgerEntry
	for _, e := range all {
		if !e.CreatedAt.Before(start) && e.CreatedAt.Before(end) {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeRepo) Adjust(ctx context.Context, userID common.UserID, id common.BankAccountID, delta float64, note string) (float64, error) {
	acc, ok := f.accounts[id]
	if !ok || acc.UserID != userID {
		return 0, repository.ErrNotFound
	}
	if acc.Balance+delta < 0 {
		return 0, repository.ErrNotFound
	}
	acc.Balance += delta
	return acc.Balance, nil
}

const testUID = common.UserID("user-1")

func TestAdjust_Add_IncreasesBalance(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts["acc-1"] = &Model{ID: "acc-1", UserID: testUID, Balance: 1000, UpdatedAt: time.Now().UTC()}
	svc := NewService(repo)

	got, err := svc.Adjust(context.Background(), testUID, "acc-1", AdjustInput{Direction: AdjustAdd, Amount: 500})
	if err != nil {
		t.Fatalf("adjust failed: %v", err)
	}
	if got.Balance != 1500 {
		t.Fatalf("expected balance 1500, got %v", got.Balance)
	}
}

func TestAdjust_Withdraw_DecreasesBalance(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts["acc-1"] = &Model{ID: "acc-1", UserID: testUID, Balance: 1000}
	svc := NewService(repo)

	got, err := svc.Adjust(context.Background(), testUID, "acc-1", AdjustInput{Direction: AdjustWithdraw, Amount: 400})
	if err != nil {
		t.Fatalf("adjust failed: %v", err)
	}
	if got.Balance != 600 {
		t.Fatalf("expected balance 600, got %v", got.Balance)
	}
}

func TestAdjust_Withdraw_InsufficientBalance_Rejected(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts["acc-1"] = &Model{ID: "acc-1", UserID: testUID, Balance: 100}
	svc := NewService(repo)

	_, err := svc.Adjust(context.Background(), testUID, "acc-1", AdjustInput{Direction: AdjustWithdraw, Amount: 200})
	if err == nil {
		t.Fatalf("expected withdrawal beyond balance to be rejected")
	}
	if !apperr.IsKind(err, apperr.Validation) {
		t.Fatalf("expected validation error kind, got %v", err)
	}
	if repo.accounts["acc-1"].Balance != 100 {
		t.Fatalf("expected balance to be untouched after a rejected withdrawal, got %v", repo.accounts["acc-1"].Balance)
	}
}

func TestAdjust_ZeroOrNegativeAmount_Rejected(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts["acc-1"] = &Model{ID: "acc-1", UserID: testUID, Balance: 100}
	svc := NewService(repo)

	if _, err := svc.Adjust(context.Background(), testUID, "acc-1", AdjustInput{Direction: AdjustAdd, Amount: 0}); err == nil {
		t.Fatalf("expected zero amount to be rejected")
	}
	if _, err := svc.Adjust(context.Background(), testUID, "acc-1", AdjustInput{Direction: AdjustAdd, Amount: -50}); err == nil {
		t.Fatalf("expected negative amount to be rejected")
	}
}

func TestAdjust_InvalidDirection_Rejected(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts["acc-1"] = &Model{ID: "acc-1", UserID: testUID, Balance: 100}
	svc := NewService(repo)

	_, err := svc.Adjust(context.Background(), testUID, "acc-1", AdjustInput{Direction: AdjustDirection("transfer"), Amount: 50})
	if err == nil {
		t.Fatalf("expected invalid direction to be rejected")
	}
}

func TestAdjust_UnknownAccount_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	_, err := svc.Adjust(context.Background(), testUID, "acc-missing", AdjustInput{Direction: AdjustAdd, Amount: 50})
	if err == nil {
		t.Fatalf("expected unknown account to be rejected")
	}
	if !apperr.IsKind(err, apperr.NotFound) {
		t.Fatalf("expected not-found error kind, got %v", err)
	}
}

func TestBalanceHistory_ReturnsClosingBalancePerMonth(t *testing.T) {
	repo := newFakeRepo()
	now := time.Now().UTC()
	curStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	twoMonthsAgoStart := curStart.AddDate(0, -2, 0)
	oneMonthAgoStart := curStart.AddDate(0, -1, 0)

	repo.ledgers["acc-1"] = []LedgerEntry{
		{ID: "l1", BankAccountID: "acc-1", Type: LedgerOpening, Amount: 1000, BalanceAfter: 1000, CreatedAt: twoMonthsAgoStart.AddDate(0, 0, 2)},
		{ID: "l2", BankAccountID: "acc-1", Type: LedgerAdd, Amount: 500, BalanceAfter: 1500, CreatedAt: oneMonthAgoStart.AddDate(0, 0, 5)},
	}
	svc := NewService(repo)

	got, err := svc.BalanceHistory(context.Background(), testUID, "acc-1", 3)
	if err != nil {
		t.Fatalf("balance history failed: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 months, got %d", len(got))
	}
	if got[0].Balance != 1000 {
		t.Fatalf("expected month -2 closing balance 1000, got %v", got[0].Balance)
	}
	if got[1].Balance != 1500 {
		t.Fatalf("expected month -1 closing balance 1500, got %v", got[1].Balance)
	}
	if got[2].Balance != 1500 {
		t.Fatalf("expected current month to carry forward balance 1500, got %v", got[2].Balance)
	}
}

func TestBalanceHistory_NoLedgerEntries_AllZero(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	got, err := svc.BalanceHistory(context.Background(), testUID, "acc-empty", 3)
	if err != nil {
		t.Fatalf("balance history failed: %v", err)
	}
	for _, m := range got {
		if m.Balance != 0 {
			t.Fatalf("expected zero balance for account with no ledger, got %v for %s", m.Balance, m.Month)
		}
	}
}

func TestBalanceHistory_MonthsAboveTwelve_Rejected(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	_, err := svc.BalanceHistory(context.Background(), testUID, "acc-1", 13)
	if err == nil {
		t.Fatalf("expected months > 12 to be rejected")
	}
	if !apperr.IsKind(err, apperr.Validation) {
		t.Fatalf("expected validation error kind, got %v", err)
	}
}
