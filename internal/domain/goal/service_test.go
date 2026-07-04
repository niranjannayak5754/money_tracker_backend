package goal

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/bankaccount"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/investment"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

// --- fakes (in-memory, no DB dependency) ---

type fakeGoalRepo struct {
	mu     sync.Mutex
	items  map[common.GoalID]*Model
	nextID int
}

func newFakeGoalRepo() *fakeGoalRepo {
	return &fakeGoalRepo{items: map[common.GoalID]*Model{}}
}

func (f *fakeGoalRepo) Create(ctx context.Context, m Model) (common.GoalID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := common.GoalID(fmt.Sprintf("goal-%d", f.nextID))
	m.ID = id
	cp := m
	f.items[id] = &cp
	return id, nil
}

func (f *fakeGoalRepo) List(ctx context.Context, userID common.UserID) ([]Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Model
	for _, m := range f.items {
		if m.UserID == userID {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (f *fakeGoalRepo) Update(ctx context.Context, userID common.UserID, id common.GoalID, set map[string]any) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return false, nil
	}
	for k, v := range set {
		switch k {
		case "name":
			m.Name = v.(string)
		case "target_amount":
			m.TargetAmount = v.(float64)
		case "status":
			m.Status = Status(v.(string))
		}
	}
	return true, nil
}

func (f *fakeGoalRepo) Delete(ctx context.Context, userID common.UserID, id common.GoalID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return false, nil
	}
	delete(f.items, id)
	return true, nil
}

type fakeBankAccountRepo struct {
	accounts map[common.BankAccountID]*bankaccount.Model
}

func (f *fakeBankAccountRepo) GetByID(ctx context.Context, userID common.UserID, id common.BankAccountID) (*bankaccount.Model, error) {
	acc, ok := f.accounts[id]
	if !ok || acc.UserID != userID {
		return nil, repository.ErrNotFound
	}
	return acc, nil
}

type fakeInvestmentRepo struct {
	investments map[common.InvestmentID]*investment.Model
}

func (f *fakeInvestmentRepo) GetByID(ctx context.Context, userID common.UserID, id common.InvestmentID) (*investment.Model, error) {
	inv, ok := f.investments[id]
	if !ok || inv.UserID != userID {
		return nil, repository.ErrNotFound
	}
	return inv, nil
}

// --- tests ---

const testUID = common.UserID("user-1")

func TestResolveProgress_BankAccountLinked_ComputesFromBalance(t *testing.T) {
	goalRepo := newFakeGoalRepo()
	bankRepo := &fakeBankAccountRepo{accounts: map[common.BankAccountID]*bankaccount.Model{
		"acc-1": {ID: "acc-1", UserID: testUID, Balance: 4000},
	}}
	svc := NewService(goalRepo, bankRepo, &fakeInvestmentRepo{})
	ctx := context.Background()

	g, err := svc.Create(ctx, testUID, CreateInput{
		Name:         "Emergency Fund",
		TargetAmount: 10000,
		LinkedType:   LinkedBankAccount,
		LinkedID:     "acc-1",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if g.CurrentValue != 4000 {
		t.Fatalf("expected current value 4000, got %v", g.CurrentValue)
	}
	if g.PercentComplete != 40 {
		t.Fatalf("expected 40%% complete, got %v", g.PercentComplete)
	}
	if g.LinkedEntityMissing {
		t.Fatalf("expected linked entity to be found")
	}
}

func TestResolveProgress_InvestmentLinked_ComputesFromAmountPlusRetrieved(t *testing.T) {
	goalRepo := newFakeGoalRepo()
	invRepo := &fakeInvestmentRepo{investments: map[common.InvestmentID]*investment.Model{
		"inv-1": {ID: "inv-1", UserID: testUID, Amount: 3000, RetrievedAmount: 2000},
	}}
	svc := NewService(goalRepo, &fakeBankAccountRepo{}, invRepo)
	ctx := context.Background()

	g, err := svc.Create(ctx, testUID, CreateInput{
		Name:         "House Downpayment",
		TargetAmount: 10000,
		LinkedType:   LinkedInvestment,
		LinkedID:     "inv-1",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// total contribution counts remaining basis + whatever was withdrawn,
	// not just the current remaining amount.
	if g.CurrentValue != 5000 {
		t.Fatalf("expected current value 5000 (3000+2000), got %v", g.CurrentValue)
	}
	if g.PercentComplete != 50 {
		t.Fatalf("expected 50%% complete, got %v", g.PercentComplete)
	}
}

func TestPercentComplete_CapsAt100WhenOverfunded(t *testing.T) {
	goalRepo := newFakeGoalRepo()
	bankRepo := &fakeBankAccountRepo{accounts: map[common.BankAccountID]*bankaccount.Model{
		"acc-1": {ID: "acc-1", UserID: testUID, Balance: 15000},
	}}
	svc := NewService(goalRepo, bankRepo, &fakeInvestmentRepo{})
	ctx := context.Background()

	g, err := svc.Create(ctx, testUID, CreateInput{
		Name:         "Overfunded Goal",
		TargetAmount: 10000,
		LinkedType:   LinkedBankAccount,
		LinkedID:     "acc-1",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if g.PercentComplete != 100 {
		t.Fatalf("expected percent complete capped at 100, got %v", g.PercentComplete)
	}
	if g.CurrentValue != 15000 {
		t.Fatalf("expected current value to still report the true balance 15000 (only the percent is capped), got %v", g.CurrentValue)
	}
}

func TestResolveProgress_MissingLinkedBankAccount_DegradesGracefully(t *testing.T) {
	goalRepo := newFakeGoalRepo()
	svc := NewService(goalRepo, &fakeBankAccountRepo{accounts: map[common.BankAccountID]*bankaccount.Model{}}, &fakeInvestmentRepo{})
	ctx := context.Background()

	g, err := svc.Create(ctx, testUID, CreateInput{
		Name:         "Deleted Account Goal",
		TargetAmount: 5000,
		LinkedType:   LinkedBankAccount,
		LinkedID:     "acc-does-not-exist",
	})
	if err != nil {
		t.Fatalf("expected create to succeed even if the linked account can't be resolved yet, got error: %v", err)
	}

	if !g.LinkedEntityMissing {
		t.Fatalf("expected LinkedEntityMissing to be true for a nonexistent linked account")
	}
	if g.CurrentValue != 0 {
		t.Fatalf("expected current value 0 when linked entity is missing, got %v", g.CurrentValue)
	}
	if g.PercentComplete != 0 {
		t.Fatalf("expected percent complete 0 when linked entity is missing, got %v", g.PercentComplete)
	}
}

func TestResolveProgress_MissingLinkedInvestment_DegradesGracefully(t *testing.T) {
	goalRepo := newFakeGoalRepo()
	svc := NewService(goalRepo, &fakeBankAccountRepo{}, &fakeInvestmentRepo{investments: map[common.InvestmentID]*investment.Model{}})
	ctx := context.Background()

	g, err := svc.Create(ctx, testUID, CreateInput{
		Name:         "Closed Investment Goal",
		TargetAmount: 5000,
		LinkedType:   LinkedInvestment,
		LinkedID:     "inv-does-not-exist",
	})
	if err != nil {
		t.Fatalf("expected create to succeed, got error: %v", err)
	}
	if !g.LinkedEntityMissing {
		t.Fatalf("expected LinkedEntityMissing to be true for a nonexistent linked investment")
	}
}

func TestList_ResolvesProgressForEachGoal(t *testing.T) {
	goalRepo := newFakeGoalRepo()
	bankRepo := &fakeBankAccountRepo{accounts: map[common.BankAccountID]*bankaccount.Model{
		"acc-1": {ID: "acc-1", UserID: testUID, Balance: 2500},
	}}
	svc := NewService(goalRepo, bankRepo, &fakeInvestmentRepo{})
	ctx := context.Background()

	if _, err := svc.Create(ctx, testUID, CreateInput{Name: "G1", TargetAmount: 5000, LinkedType: LinkedBankAccount, LinkedID: "acc-1"}); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	items, err := svc.List(ctx, testUID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || items[0].CurrentValue != 2500 {
		t.Fatalf("expected 1 goal with resolved progress 2500, got %+v", items)
	}
}

func TestCreate_ValidatesLinkedType(t *testing.T) {
	svc := NewService(newFakeGoalRepo(), &fakeBankAccountRepo{}, &fakeInvestmentRepo{})
	ctx := context.Background()

	_, err := svc.Create(ctx, testUID, CreateInput{
		Name:         "Bad Goal",
		TargetAmount: 1000,
		LinkedType:   LinkedType("something_else"),
		LinkedID:     "x",
	})
	if err == nil {
		t.Fatalf("expected invalid linked_type to be rejected")
	}
}
