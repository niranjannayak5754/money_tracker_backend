package income

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

// --- fakes (in-memory, no DB dependency) ---

type fakeIncomeRepo struct {
	mu     sync.Mutex
	items  map[common.IncomeID]*Model
	nextID int
}

func newFakeIncomeRepo() *fakeIncomeRepo {
	return &fakeIncomeRepo{items: map[common.IncomeID]*Model{}}
}

func (f *fakeIncomeRepo) Create(ctx context.Context, m Model) (common.IncomeID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := common.IncomeID(fmt.Sprintf("income-%d", f.nextID))
	cp := m
	cp.ID = id
	f.items[id] = &cp
	return id, nil
}

func (f *fakeIncomeRepo) ListByMonth(ctx context.Context, userID common.UserID, start, end time.Time) ([]Model, error) {
	return nil, nil
}

func (f *fakeIncomeRepo) Update(ctx context.Context, userID common.UserID, id common.IncomeID, set map[string]any) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return false, nil
	}
	if cid, ok := set["category_id"].(common.CategoryID); ok {
		m.CategoryID = cid
	}
	return true, nil
}

func (f *fakeIncomeRepo) Delete(ctx context.Context, userID common.UserID, id common.IncomeID) (bool, error) {
	return false, nil
}

func (f *fakeIncomeRepo) CountByCategory(ctx context.Context, userID common.UserID, categoryID common.CategoryID) (int64, error) {
	return 0, nil
}

func (f *fakeIncomeRepo) ReassignCategory(ctx context.Context, userID common.UserID, fromCategoryID, toCategoryID common.CategoryID) (int64, error) {
	return 0, nil
}

// fakeCategoryRepo lets tests control which category IDs/types validate.
type fakeCategoryRepo struct {
	valid map[common.CategoryID]string // categoryID -> type
}

func (f *fakeCategoryRepo) ExistsForUserWithType(ctx context.Context, userID common.UserID, categoryID common.CategoryID, categoryType string) (bool, error) {
	return f.valid[categoryID] == categoryType, nil
}

// --- tests ---

const testUID = common.UserID("user-1")
const testIncomeCatID = common.CategoryID("cat-income-1")
const testExpenseCatID = common.CategoryID("cat-expense-1")

func newTestService() (Service, *fakeIncomeRepo) {
	repo := newFakeIncomeRepo()
	cats := &fakeCategoryRepo{valid: map[common.CategoryID]string{
		testIncomeCatID:  "income",
		testExpenseCatID: "expense",
	}}
	return NewService(repo, cats), repo
}

func TestCreate_MissingCategory_Rejected(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Create(context.Background(), testUID, CreateInput{Amount: 1000, Date: time.Now().UTC()})
	if err == nil {
		t.Fatalf("expected create without category_id to be rejected")
	}
	if !apperr.IsKind(err, apperr.Validation) {
		t.Fatalf("expected validation error kind, got %v", err)
	}
}

func TestCreate_WrongTypeCategory_Rejected(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Create(context.Background(), testUID, CreateInput{Amount: 1000, Date: time.Now().UTC(), CategoryID: testExpenseCatID})
	if err == nil {
		t.Fatalf("expected create with an expense-type category to be rejected for income")
	}
	if !apperr.IsKind(err, apperr.Validation) {
		t.Fatalf("expected validation error kind, got %v", err)
	}
}

func TestCreate_ValidIncomeCategory_Succeeds(t *testing.T) {
	svc, _ := newTestService()
	rec, err := svc.Create(context.Background(), testUID, CreateInput{Amount: 50000, Date: time.Now().UTC(), CategoryID: testIncomeCatID, Source: "Salary"})
	if err != nil {
		t.Fatalf("expected create with a valid income category to succeed, got %v", err)
	}
	if rec.CategoryID != testIncomeCatID {
		t.Fatalf("expected category_id to be persisted on the record, got %v", rec.CategoryID)
	}
}

func TestUpdate_CategoryRevalidatedOnChange(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	rec, err := svc.Create(ctx, testUID, CreateInput{Amount: 1000, Date: time.Now().UTC(), CategoryID: testIncomeCatID})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	wrongCat := testExpenseCatID
	if err := svc.Update(ctx, testUID, rec.ID, UpdateInput{CategoryID: &wrongCat}); err == nil {
		t.Fatalf("expected update to an expense-type category to be rejected")
	}
}
