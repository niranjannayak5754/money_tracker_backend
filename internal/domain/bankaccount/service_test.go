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
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{accounts: map[common.BankAccountID]*Model{}}
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
	return nil, nil
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
