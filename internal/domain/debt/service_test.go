package debt

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

type fakeDebtRepo struct {
	mu     sync.Mutex
	items  map[common.DebtID]*Model
	nextID int
}

func newFakeDebtRepo() *fakeDebtRepo {
	return &fakeDebtRepo{items: map[common.DebtID]*Model{}}
}

func (f *fakeDebtRepo) Create(ctx context.Context, m Model) (common.DebtID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := common.DebtID(fmt.Sprintf("debt-%d", f.nextID))
	m.ID = id
	cp := m
	f.items[id] = &cp
	return id, nil
}

func (f *fakeDebtRepo) List(ctx context.Context, userID common.UserID) ([]Model, error) {
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

func (f *fakeDebtRepo) GetByID(ctx context.Context, userID common.UserID, id common.DebtID) (*Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return nil, repository.ErrNotFound
	}
	cp := *m
	return &cp, nil
}

func (f *fakeDebtRepo) Update(ctx context.Context, userID common.UserID, id common.DebtID, set map[string]any) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return false, nil
	}
	for k, v := range set {
		switch k {
		case "outstanding_balance":
			m.OutstandingBalance = v.(float64)
		case "total_interest_paid":
			m.TotalInterestPaid = v.(float64)
		case "status":
			m.Status = Status(v.(string))
		case "name":
			m.Name = v.(string)
		case "interest_rate":
			m.InterestRate = v.(float64)
		case "emi_amount":
			m.EMIAmount = v.(float64)
		case "tenure_months":
			m.TenureMonths = v.(int)
		case "notes":
			m.Notes = v.(string)
		}
	}
	return true, nil
}

func (f *fakeDebtRepo) Delete(ctx context.Context, userID common.UserID, id common.DebtID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.items[id]
	if !ok {
		return false, nil
	}
	delete(f.items, id)
	return true, nil
}

// --- tests ---

const testUID = common.UserID("user-1")

func mustCreateDebt(t *testing.T, svc Service, principal float64) Model {
	t.Helper()
	d, err := svc.Create(context.Background(), testUID, CreateInput{
		Name:      "Home Loan",
		Principal: principal,
		StartDate: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create debt failed: %v", err)
	}
	return d
}

func TestRecordPayment_ReducesOutstandingBalance(t *testing.T) {
	repo := newFakeDebtRepo()
	svc := NewService(repo)
	ctx := context.Background()

	d := mustCreateDebt(t, svc, 100000)

	out, err := svc.RecordPayment(ctx, testUID, d.ID, PaymentInput{
		PrincipalComponent: 5000,
		InterestComponent:  800,
		Date:               time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("record payment failed: %v", err)
	}

	if out.OutstandingBalance != 95000 {
		t.Fatalf("expected outstanding balance 95000, got %v", out.OutstandingBalance)
	}
	if out.TotalInterestPaid != 800 {
		t.Fatalf("expected total interest paid 800, got %v", out.TotalInterestPaid)
	}
	if out.Status != StatusActive {
		t.Fatalf("expected status ACTIVE, got %v", out.Status)
	}
}

func TestRecordPayment_AutoClosesAtZeroBalance(t *testing.T) {
	repo := newFakeDebtRepo()
	svc := NewService(repo)
	ctx := context.Background()

	d := mustCreateDebt(t, svc, 5000)

	out, err := svc.RecordPayment(ctx, testUID, d.ID, PaymentInput{
		PrincipalComponent: 5000,
		InterestComponent:  100,
		Date:               time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("record payment failed: %v", err)
	}

	if out.OutstandingBalance != 0 {
		t.Fatalf("expected outstanding balance 0, got %v", out.OutstandingBalance)
	}
	if out.Status != StatusClosed {
		t.Fatalf("expected status CLOSED, got %v", out.Status)
	}
}

func TestRecordPayment_OverpaymentRejected(t *testing.T) {
	repo := newFakeDebtRepo()
	svc := NewService(repo)
	ctx := context.Background()

	d := mustCreateDebt(t, svc, 5000)

	_, err := svc.RecordPayment(ctx, testUID, d.ID, PaymentInput{
		PrincipalComponent: 6000,
		InterestComponent:  100,
		Date:               time.Now().UTC(),
	})
	if err == nil {
		t.Fatalf("expected overpayment to be rejected")
	}
	if !apperr.IsKind(err, apperr.Validation) {
		t.Fatalf("expected validation error kind, got %v", err)
	}
}

func TestRecordPayment_AlreadyClosedRejected(t *testing.T) {
	repo := newFakeDebtRepo()
	svc := NewService(repo)
	ctx := context.Background()

	d := mustCreateDebt(t, svc, 1000)

	if _, err := svc.RecordPayment(ctx, testUID, d.ID, PaymentInput{
		PrincipalComponent: 1000,
		InterestComponent:  50,
		Date:               time.Now().UTC(),
	}); err != nil {
		t.Fatalf("first payment failed: %v", err)
	}

	if _, err := svc.RecordPayment(ctx, testUID, d.ID, PaymentInput{
		PrincipalComponent: 10,
		InterestComponent:  1,
		Date:               time.Now().UTC(),
	}); err == nil {
		t.Fatalf("expected payment on a closed debt to be rejected")
	}
}

func TestRecordPayment_SequentialPaymentsAccumulateInterest(t *testing.T) {
	repo := newFakeDebtRepo()
	svc := NewService(repo)
	ctx := context.Background()

	d := mustCreateDebt(t, svc, 10000)

	if _, err := svc.RecordPayment(ctx, testUID, d.ID, PaymentInput{
		PrincipalComponent: 3000,
		InterestComponent:  500,
		Date:               time.Now().UTC(),
	}); err != nil {
		t.Fatalf("first payment failed: %v", err)
	}

	out, err := svc.RecordPayment(ctx, testUID, d.ID, PaymentInput{
		PrincipalComponent: 3000,
		InterestComponent:  400,
		Date:               time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("second payment failed: %v", err)
	}

	if out.OutstandingBalance != 4000 {
		t.Fatalf("expected outstanding balance 4000, got %v", out.OutstandingBalance)
	}
	if out.TotalInterestPaid != 900 {
		t.Fatalf("expected cumulative interest paid 900, got %v", out.TotalInterestPaid)
	}
}
