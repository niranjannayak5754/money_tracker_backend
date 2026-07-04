package investment

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

// --- fake repository (in-memory, no DB dependency) ---

type fakeInvestmentRepo struct {
	mu     sync.Mutex
	items  map[common.InvestmentID]*Model
	types  []TypeDoc
	nextID int
}

func newFakeInvestmentRepo() *fakeInvestmentRepo {
	return &fakeInvestmentRepo{
		items: map[common.InvestmentID]*Model{},
		types: []TypeDoc{
			{Key: "gold", Name: "Gold", Active: true},
			{Key: "fixed_deposit", Name: "Fixed Deposit (FD)", Active: true},
		},
	}
}

func (f *fakeInvestmentRepo) Create(ctx context.Context, m Model) (common.InvestmentID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := common.InvestmentID(fmt.Sprintf("inv-%d", f.nextID))
	m.ID = id
	cp := m
	f.items[id] = &cp
	return id, nil
}

func (f *fakeInvestmentRepo) ListByMonth(ctx context.Context, userID common.UserID, start, end time.Time) ([]Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Model
	for _, m := range f.items {
		if m.UserID == userID && m.DeletedAt == nil {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (f *fakeInvestmentRepo) GetByID(ctx context.Context, userID common.UserID, id common.InvestmentID) (*Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID || m.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	cp := *m
	return &cp, nil
}

func (f *fakeInvestmentRepo) Update(ctx context.Context, userID common.UserID, id common.InvestmentID, set map[string]any) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID || m.DeletedAt != nil {
		return false, nil
	}
	for k, v := range set {
		switch k {
		case "amount":
			m.Amount = v.(float64)
		case "retrieved_amount":
			m.RetrievedAmount = v.(float64)
		case "realized_pnl":
			m.RealizedPnl = v.(float64)
		case "status":
			m.Status = Status(v.(string))
		case "notes":
			m.Notes = v.(string)
		case "type":
			m.Type = v.(string)
		case "display_type":
			m.DisplayType = v.(string)
		case "instrument":
			m.Instrument = v.(string)
		case "date":
			m.Date = v.(time.Time)
		case "updated_at":
			m.UpdatedAt = v.(time.Time)
		}
	}
	return true, nil
}

func (f *fakeInvestmentRepo) Delete(ctx context.Context, userID common.UserID, id common.InvestmentID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return false, nil
	}
	now := time.Now().UTC()
	m.DeletedAt = &now
	return true, nil
}

func (f *fakeInvestmentRepo) GetTypes(ctx context.Context) ([]TypeDoc, error) {
	return f.types, nil
}

// --- tests ---

const testUID = common.UserID("user-1")

func mustCreateInvestment(t *testing.T, svc Service, amount float64) Model {
	t.Helper()
	m, err := svc.Create(context.Background(), testUID, CreateInput{
		Type:   "fixed_deposit",
		Amount: amount,
		Date:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create investment failed: %v", err)
	}
	return m
}

func TestClose_FullClose_ComputesRealizedPnl(t *testing.T) {
	repo := newFakeInvestmentRepo()
	svc := NewService(repo)
	ctx := context.Background()

	inv := mustCreateInvestment(t, svc, 10000)

	out, err := svc.Close(ctx, testUID, inv.ID, CloseInput{
		WithdrawnAmount:   11000,
		CostBasisConsumed: 10000,
		Date:              time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if out.RealizedPnl != 1000 {
		t.Fatalf("expected realized pnl 1000, got %v", out.RealizedPnl)
	}
	if out.RetrievedAmount != 11000 {
		t.Fatalf("expected retrieved amount 11000, got %v", out.RetrievedAmount)
	}
	if out.Amount != 0 {
		t.Fatalf("expected remaining amount 0, got %v", out.Amount)
	}
	if out.Status != StatusClosed {
		t.Fatalf("expected status CLOSED, got %v", out.Status)
	}
}

func TestClose_PartialWithdrawal_LeavesPartiallyRetrieved(t *testing.T) {
	repo := newFakeInvestmentRepo()
	svc := NewService(repo)
	ctx := context.Background()

	inv := mustCreateInvestment(t, svc, 10000)

	out, err := svc.Close(ctx, testUID, inv.ID, CloseInput{
		WithdrawnAmount:   4000,
		CostBasisConsumed: 3000,
		Date:              time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if out.Amount != 7000 {
		t.Fatalf("expected remaining amount 7000, got %v", out.Amount)
	}
	if out.RetrievedAmount != 4000 {
		t.Fatalf("expected retrieved amount 4000, got %v", out.RetrievedAmount)
	}
	if out.RealizedPnl != 1000 {
		t.Fatalf("expected realized pnl 1000, got %v", out.RealizedPnl)
	}
	if out.Status != StatusPartiallyRetrieved {
		t.Fatalf("expected status PARTIALLY_RETRIEVED, got %v", out.Status)
	}
}

func TestClose_OverWithdraw_Rejected(t *testing.T) {
	repo := newFakeInvestmentRepo()
	svc := NewService(repo)
	ctx := context.Background()

	inv := mustCreateInvestment(t, svc, 5000)

	_, err := svc.Close(ctx, testUID, inv.ID, CloseInput{
		WithdrawnAmount:   7000,
		CostBasisConsumed: 6000,
		Date:              time.Now().UTC(),
	})
	if err == nil {
		t.Fatalf("expected error when cost basis consumed exceeds remaining amount")
	}
	if !apperr.IsKind(err, apperr.Validation) {
		t.Fatalf("expected validation error kind, got %v", err)
	}
}

func TestClose_SequentialPartialWithdrawals_SumToFullClose(t *testing.T) {
	repo := newFakeInvestmentRepo()
	svc := NewService(repo)
	ctx := context.Background()

	inv := mustCreateInvestment(t, svc, 10000)

	first, err := svc.Close(ctx, testUID, inv.ID, CloseInput{
		WithdrawnAmount:   4000,
		CostBasisConsumed: 3000,
		Date:              time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if first.Status != StatusPartiallyRetrieved {
		t.Fatalf("expected PARTIALLY_RETRIEVED after first withdrawal, got %v", first.Status)
	}

	second, err := svc.Close(ctx, testUID, inv.ID, CloseInput{
		WithdrawnAmount:   8000,
		CostBasisConsumed: 7000, // exactly the remaining basis
		Date:              time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("second close failed: %v", err)
	}

	if second.Amount != 0 {
		t.Fatalf("expected remaining amount 0 after full withdrawal, got %v", second.Amount)
	}
	if second.Status != StatusClosed {
		t.Fatalf("expected status CLOSED after final withdrawal, got %v", second.Status)
	}
	if second.RetrievedAmount != 12000 {
		t.Fatalf("expected cumulative retrieved amount 12000, got %v", second.RetrievedAmount)
	}
	if second.RealizedPnl != 2000 {
		t.Fatalf("expected cumulative realized pnl 2000, got %v", second.RealizedPnl)
	}
}

func TestClose_AlreadyClosed_Rejected(t *testing.T) {
	repo := newFakeInvestmentRepo()
	svc := NewService(repo)
	ctx := context.Background()

	inv := mustCreateInvestment(t, svc, 5000)

	if _, err := svc.Close(ctx, testUID, inv.ID, CloseInput{
		WithdrawnAmount:   5500,
		CostBasisConsumed: 5000,
		Date:              time.Now().UTC(),
	}); err != nil {
		t.Fatalf("first close failed: %v", err)
	}

	if _, err := svc.Close(ctx, testUID, inv.ID, CloseInput{
		WithdrawnAmount:   100,
		CostBasisConsumed: 100,
		Date:              time.Now().UTC(),
	}); err == nil {
		t.Fatalf("expected error when closing an already-closed investment")
	}
}
