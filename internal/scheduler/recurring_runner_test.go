package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/recurring"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
)

// --- fakes ---

type fakeRecurringService struct {
	mu         sync.Mutex
	due        []recurring.Model
	upcoming   []recurring.Model
	markRunIDs []common.RecurringID
	markRunErr error
}

func (f *fakeRecurringService) Create(ctx context.Context, userID common.UserID, in recurring.CreateInput) (recurring.Model, error) {
	return recurring.Model{}, nil
}
func (f *fakeRecurringService) List(ctx context.Context, userID common.UserID) ([]recurring.Model, error) {
	return nil, nil
}
func (f *fakeRecurringService) Update(ctx context.Context, userID common.UserID, id common.RecurringID, in recurring.UpdateInput) error {
	return nil
}
func (f *fakeRecurringService) Delete(ctx context.Context, userID common.UserID, id common.RecurringID) error {
	return nil
}
func (f *fakeRecurringService) ListDue(ctx context.Context, asOf time.Time) ([]recurring.Model, error) {
	return f.due, nil
}
func (f *fakeRecurringService) MarkRun(ctx context.Context, userID common.UserID, id common.RecurringID, ranAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markRunIDs = append(f.markRunIDs, id)
	return f.markRunErr
}
func (f *fakeRecurringService) ListUpcoming(ctx context.Context, from, to time.Time) ([]recurring.Model, error) {
	return f.upcoming, nil
}

type fakeExpenseService struct {
	created []expense.CreateInput
}

func (f *fakeExpenseService) Create(ctx context.Context, userID common.UserID, in expense.CreateInput) (expense.Model, error) {
	f.created = append(f.created, in)
	return expense.Model{}, nil
}
func (f *fakeExpenseService) List(ctx context.Context, userID common.UserID, month, category string) ([]expense.Model, error) {
	return nil, nil
}
func (f *fakeExpenseService) Update(ctx context.Context, userID common.UserID, id common.ExpenseID, in expense.UpdateInput) error {
	return nil
}
func (f *fakeExpenseService) Delete(ctx context.Context, userID common.UserID, id common.ExpenseID) error {
	return nil
}

// --- tests ---

func TestRunOnce_MaterializesExpenseTemplate(t *testing.T) {
	nextRun := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	recSvc := &fakeRecurringService{due: []recurring.Model{{
		ID:          "rec-1",
		UserID:      "user-1",
		EntityType:  recurring.EntityExpense,
		NextRunDate: nextRun,
		Payload: map[string]any{
			"amount":      250.0,
			"category_id": "cat-1",
			"merchant":    "Landlord",
		},
	}}}
	expSvc := &fakeExpenseService{}

	runner := NewRecurringRunner(recSvc, expSvc, logging.New())
	runner.RunOnce(context.Background())

	if len(expSvc.created) != 1 {
		t.Fatalf("expected 1 expense created, got %d", len(expSvc.created))
	}
	got := expSvc.created[0]
	if got.Amount != 250.0 || got.CategoryID != "cat-1" || got.Merchant != "Landlord" || !got.Date.Equal(nextRun) {
		t.Fatalf("unexpected expense create input: %+v", got)
	}
	if len(recSvc.markRunIDs) != 1 || recSvc.markRunIDs[0] != "rec-1" {
		t.Fatalf("expected MarkRun to be called once for rec-1, got %v", recSvc.markRunIDs)
	}
}

func TestRunOnce_NoDueTemplates_CreatesNothing(t *testing.T) {
	recSvc := &fakeRecurringService{due: nil}
	expSvc := &fakeExpenseService{}

	runner := NewRecurringRunner(recSvc, expSvc, logging.New())
	runner.RunOnce(context.Background())

	if len(expSvc.created) != 0 {
		t.Fatalf("expected no records created when nothing is due")
	}
	if len(recSvc.markRunIDs) != 0 {
		t.Fatalf("expected MarkRun not called when nothing is due")
	}
}
