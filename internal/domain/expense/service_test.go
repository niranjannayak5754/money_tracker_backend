package expense

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/budget"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/notification"
)

// --- fakes (in-memory, no DB dependency) ---

type fakeExpenseRepo struct {
	mu     sync.Mutex
	items  map[common.ExpenseID]*Model
	nextID int
}

func newFakeExpenseRepo() *fakeExpenseRepo {
	return &fakeExpenseRepo{items: map[common.ExpenseID]*Model{}}
}

func (f *fakeExpenseRepo) Create(ctx context.Context, m Model) (common.ExpenseID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := common.ExpenseID(fmt.Sprintf("exp-%d", f.nextID))
	m.ID = id
	cp := m
	f.items[id] = &cp
	return id, nil
}

func (f *fakeExpenseRepo) ListByMonth(ctx context.Context, userID common.UserID, start, end time.Time, categoryID *common.CategoryID) ([]Model, error) {
	return nil, nil
}

func (f *fakeExpenseRepo) Update(ctx context.Context, userID common.UserID, id common.ExpenseID, set map[string]any) (bool, error) {
	return false, nil
}

func (f *fakeExpenseRepo) Delete(ctx context.Context, userID common.UserID, id common.ExpenseID) (bool, error) {
	return false, nil
}

func (f *fakeExpenseRepo) CountByCategory(ctx context.Context, userID common.UserID, categoryID common.CategoryID) (int64, error) {
	return 0, nil
}

func (f *fakeExpenseRepo) ReassignCategory(ctx context.Context, userID common.UserID, fromCategoryID, toCategoryID common.CategoryID) (int64, error) {
	return 0, nil
}

func (f *fakeExpenseRepo) SumByCategoryForMonth(ctx context.Context, userID common.UserID, categoryID common.CategoryID, start, end time.Time) (float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var total float64
	for _, m := range f.items {
		if m.UserID == userID && m.CategoryID == categoryID && !m.Date.Before(start) && m.Date.Before(end) {
			total += m.Amount
		}
	}
	return total, nil
}

type fakeCategoryRepo struct{}

func (f *fakeCategoryRepo) ExistsForUser(ctx context.Context, userID common.UserID, categoryID common.CategoryID) (bool, error) {
	return true, nil
}

type fakeBudgetService struct {
	byCategory map[common.CategoryID]float64
}

func (f *fakeBudgetService) Set(ctx context.Context, userID common.UserID, in budget.SetInput) (budget.Model, error) {
	return budget.Model{}, nil
}
func (f *fakeBudgetService) List(ctx context.Context, userID common.UserID) ([]budget.Model, error) {
	return nil, nil
}
func (f *fakeBudgetService) Delete(ctx context.Context, userID common.UserID, id common.BudgetID) error {
	return nil
}
func (f *fakeBudgetService) ResolveForMonth(ctx context.Context, userID common.UserID, month string) (budget.Resolved, error) {
	return budget.Resolved{ByCategory: f.byCategory}, nil
}

type fakeNotificationService struct {
	mu          sync.Mutex
	createCalls []notification.CreateInput
}

func (f *fakeNotificationService) Create(ctx context.Context, userID common.UserID, in notification.CreateInput) (notification.Model, error) {
	return notification.Model{}, nil
}
func (f *fakeNotificationService) List(ctx context.Context, userID common.UserID, onlyUnread bool) ([]notification.Model, error) {
	return nil, nil
}
func (f *fakeNotificationService) UnreadCount(ctx context.Context, userID common.UserID) (int64, error) {
	return 0, nil
}
func (f *fakeNotificationService) MarkRead(ctx context.Context, userID common.UserID, id common.NotificationID) error {
	return nil
}
func (f *fakeNotificationService) Dismiss(ctx context.Context, userID common.UserID, id common.NotificationID) error {
	return nil
}
func (f *fakeNotificationService) CreateIfNotExists(ctx context.Context, userID common.UserID, in notification.CreateInput) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.createCalls {
		if c.Type == in.Type && c.RelatedEntityType == in.RelatedEntityType && c.RelatedEntityID == in.RelatedEntityID {
			return false, nil
		}
	}
	f.createCalls = append(f.createCalls, in)
	return true, nil
}

// --- tests ---

const testUID = common.UserID("user-1")
const testCatID = common.CategoryID("cat-1")

func TestCreate_BudgetExceeded_FiresNotificationOnceForMultipleExpenses(t *testing.T) {
	repo := newFakeExpenseRepo()
	notif := &fakeNotificationService{}
	budgetSvc := &fakeBudgetService{byCategory: map[common.CategoryID]float64{testCatID: 1000}}
	svc := NewService(repo, &fakeCategoryRepo{}, budgetSvc, notif)
	ctx := context.Background()

	date := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	// First expense keeps spend under budget (900 <= 1000) — no notification.
	if _, err := svc.Create(ctx, testUID, CreateInput{Amount: 900, Date: date, CategoryID: testCatID}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if len(notif.createCalls) != 0 {
		t.Fatalf("expected no notification while under budget, got %d", len(notif.createCalls))
	}

	// Second expense pushes it over (1100 > 1000) — exactly one notification.
	if _, err := svc.Create(ctx, testUID, CreateInput{Amount: 200, Date: date, CategoryID: testCatID}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if len(notif.createCalls) != 1 {
		t.Fatalf("expected exactly 1 notification after exceeding budget, got %d", len(notif.createCalls))
	}

	// A third expense, still over budget in the same month, must NOT create
	// a second notification — this is the "once per category per month"
	// requirement, not once per expense.
	if _, err := svc.Create(ctx, testUID, CreateInput{Amount: 50, Date: date, CategoryID: testCatID}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if len(notif.createCalls) != 1 {
		t.Fatalf("expected still exactly 1 notification (deduped), got %d", len(notif.createCalls))
	}
}

func TestCreate_NoBudgetSet_NoNotification(t *testing.T) {
	repo := newFakeExpenseRepo()
	notif := &fakeNotificationService{}
	budgetSvc := &fakeBudgetService{byCategory: map[common.CategoryID]float64{}}
	svc := NewService(repo, &fakeCategoryRepo{}, budgetSvc, notif)
	ctx := context.Background()

	if _, err := svc.Create(ctx, testUID, CreateInput{Amount: 5000, Date: time.Now().UTC(), CategoryID: testCatID}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if len(notif.createCalls) != 0 {
		t.Fatalf("expected no notification when the category has no budget set, got %d", len(notif.createCalls))
	}
}

func TestCreate_DifferentMonths_GetIndependentNotifications(t *testing.T) {
	repo := newFakeExpenseRepo()
	notif := &fakeNotificationService{}
	budgetSvc := &fakeBudgetService{byCategory: map[common.CategoryID]float64{testCatID: 100}}
	svc := NewService(repo, &fakeCategoryRepo{}, budgetSvc, notif)
	ctx := context.Background()

	julyDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	augDate := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	if _, err := svc.Create(ctx, testUID, CreateInput{Amount: 200, Date: julyDate, CategoryID: testCatID}); err != nil {
		t.Fatalf("create failed (july): %v", err)
	}
	if _, err := svc.Create(ctx, testUID, CreateInput{Amount: 200, Date: augDate, CategoryID: testCatID}); err != nil {
		t.Fatalf("create failed (august): %v", err)
	}

	if len(notif.createCalls) != 2 {
		t.Fatalf("expected a separate notification for each month exceeded, got %d: %+v", len(notif.createCalls), notif.createCalls)
	}
}

func TestCreate_BudgetCheckFailureDoesNotFailExpenseCreate(t *testing.T) {
	repo := newFakeExpenseRepo()
	svc := NewService(repo, &fakeCategoryRepo{}, nil, nil) // nil budgets/notifications
	ctx := context.Background()

	if _, err := svc.Create(ctx, testUID, CreateInput{Amount: 100, Date: time.Now().UTC(), CategoryID: testCatID}); err != nil {
		t.Fatalf("expected expense creation to succeed even without budget/notification wiring, got: %v", err)
	}
}
