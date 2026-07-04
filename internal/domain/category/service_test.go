package category

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/ptr"
)

var errNotFound = errors.New("category not found")

// --- fakes (in-memory, no DB dependency) ---

type fakeCategoryRepo struct {
	mu     sync.Mutex
	items  map[common.CategoryID]*Model
	nextID int
}

func newFakeCategoryRepo() *fakeCategoryRepo {
	return &fakeCategoryRepo{items: map[common.CategoryID]*Model{}}
}

func (f *fakeCategoryRepo) Create(ctx context.Context, m Model) (common.CategoryID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := common.CategoryID(fmt.Sprintf("cat-%d", f.nextID))
	m.ID = id
	cp := m
	f.items[id] = &cp
	return id, nil
}

func (f *fakeCategoryRepo) Update(ctx context.Context, userID common.UserID, id common.CategoryID, set map[string]any) (bool, error) {
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
		case "color":
			m.Color = v.(string)
		case "icon":
			m.Icon = v.(string)
		case "archived":
			m.Archived = v.(bool)
		}
	}
	return true, nil
}

func (f *fakeCategoryRepo) List(ctx context.Context, userID common.UserID, archived *bool) ([]Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Model
	for _, m := range f.items {
		if m.UserID != userID {
			continue
		}
		if archived != nil && m.Archived != *archived {
			continue
		}
		out = append(out, *m)
	}
	return out, nil
}

func (f *fakeCategoryRepo) ExistsForUser(ctx context.Context, userID common.UserID, categoryID common.CategoryID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[categoryID]
	if !ok || m.UserID != userID || m.Archived {
		return false, nil
	}
	return true, nil
}

func (f *fakeCategoryRepo) ExistsForUserWithType(ctx context.Context, userID common.UserID, categoryID common.CategoryID, categoryType string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[categoryID]
	if !ok || m.UserID != userID || m.Archived || m.Type != categoryType {
		return false, nil
	}
	return true, nil
}

func (f *fakeCategoryRepo) GetByID(ctx context.Context, userID common.UserID, categoryID common.CategoryID) (Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[categoryID]
	if !ok || m.UserID != userID {
		return Model{}, errNotFound
	}
	return *m, nil
}

type fakeExpenseRepo struct {
	mu            sync.Mutex
	countByCat    map[common.CategoryID]int64
	reassignCalls []struct{ from, to common.CategoryID }
}

func newFakeExpenseRepo() *fakeExpenseRepo {
	return &fakeExpenseRepo{countByCat: map[common.CategoryID]int64{}}
}

func (f *fakeExpenseRepo) ReassignCategory(ctx context.Context, userID common.UserID, fromCategoryID, toCategoryID common.CategoryID) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := f.countByCat[fromCategoryID]
	f.reassignCalls = append(f.reassignCalls, struct{ from, to common.CategoryID }{fromCategoryID, toCategoryID})
	f.countByCat[fromCategoryID] = 0
	return n, nil
}

type fakeIncomeRepo struct {
	mu            sync.Mutex
	countByCat    map[common.CategoryID]int64
	reassignCalls []struct{ from, to common.CategoryID }
}

func newFakeIncomeRepo() *fakeIncomeRepo {
	return &fakeIncomeRepo{countByCat: map[common.CategoryID]int64{}}
}

func (f *fakeIncomeRepo) ReassignCategory(ctx context.Context, userID common.UserID, fromCategoryID, toCategoryID common.CategoryID) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := f.countByCat[fromCategoryID]
	f.reassignCalls = append(f.reassignCalls, struct{ from, to common.CategoryID }{fromCategoryID, toCategoryID})
	f.countByCat[fromCategoryID] = 0
	return n, nil
}

// --- tests ---

const testUID = common.UserID("user-1")

func mustCreateCategory(t *testing.T, svc Service, name, catType string) Model {
	t.Helper()
	m, err := svc.Create(context.Background(), testUID, CreateInput{Name: name, Type: catType})
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	return m
}

func TestUpdate_ArchiveSucceedsEvenWhenExpensesReference(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	expRepo := newFakeExpenseRepo()
	svc := NewService(catRepo, expRepo, newFakeIncomeRepo())
	ctx := context.Background()

	cat := mustCreateCategory(t, svc, "Groceries", "expense")
	expRepo.countByCat[cat.ID] = 3

	// Archiving never blocks on existing references — those expenses keep
	// resolving fine by ID against the now-archived category; only new
	// expenses are prevented from picking it (enforced elsewhere, via
	// ExistsForUserWithType filtering out archived categories).
	if err := svc.Update(ctx, testUID, cat.ID, UpdateInput{Archived: ptr.Bool(true)}); err != nil {
		t.Fatalf("expected archive to succeed even with referencing expenses, got %v", err)
	}
	if len(expRepo.reassignCalls) != 0 {
		t.Fatalf("expected no reassignment without an explicit reassign_to, got %v", expRepo.reassignCalls)
	}
}

func TestUpdate_ArchiveSucceedsWhenNoExpensesReference(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	expRepo := newFakeExpenseRepo()
	svc := NewService(catRepo, expRepo, newFakeIncomeRepo())
	ctx := context.Background()

	cat := mustCreateCategory(t, svc, "Groceries", "expense")

	if err := svc.Update(ctx, testUID, cat.ID, UpdateInput{Archived: ptr.Bool(true)}); err != nil {
		t.Fatalf("expected archive to succeed with no referencing expenses, got %v", err)
	}
}

func TestUpdate_ArchiveWithReassignMovesExpensesFirst(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	expRepo := newFakeExpenseRepo()
	svc := NewService(catRepo, expRepo, newFakeIncomeRepo())
	ctx := context.Background()

	from := mustCreateCategory(t, svc, "Groceries", "expense")
	to := mustCreateCategory(t, svc, "Household", "expense")
	expRepo.countByCat[from.ID] = 5

	err := svc.Update(ctx, testUID, from.ID, UpdateInput{
		Archived:   ptr.Bool(true),
		ReassignTo: &to.ID,
	})
	if err != nil {
		t.Fatalf("expected archive-with-reassign to succeed, got %v", err)
	}

	if len(expRepo.reassignCalls) != 1 || expRepo.reassignCalls[0].from != from.ID || expRepo.reassignCalls[0].to != to.ID {
		t.Fatalf("expected exactly one reassign call from %v to %v, got %v", from.ID, to.ID, expRepo.reassignCalls)
	}

	items, _ := catRepo.List(ctx, testUID, nil)
	for _, m := range items {
		if m.ID == from.ID && !m.Archived {
			t.Fatalf("expected category to be archived after reassign")
		}
	}
}

func TestUpdate_ReassignToSelf_Rejected(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	expRepo := newFakeExpenseRepo()
	svc := NewService(catRepo, expRepo, newFakeIncomeRepo())
	ctx := context.Background()

	cat := mustCreateCategory(t, svc, "Groceries", "expense")

	err := svc.Update(ctx, testUID, cat.ID, UpdateInput{
		Archived:   ptr.Bool(true),
		ReassignTo: &cat.ID,
	})
	if err == nil {
		t.Fatalf("expected reassigning a category to itself to be rejected")
	}
}

func TestCreate_ColorValidation(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	expRepo := newFakeExpenseRepo()
	svc := NewService(catRepo, expRepo, newFakeIncomeRepo())
	ctx := context.Background()

	if _, err := svc.Create(ctx, testUID, CreateInput{Name: "Groceries", Type: "expense", Color: "red"}); err == nil {
		t.Fatalf("expected invalid color format to be rejected")
	}

	if _, err := svc.Create(ctx, testUID, CreateInput{Name: "Groceries", Type: "expense", Color: "#FF00AA"}); err != nil {
		t.Fatalf("expected valid hex color to be accepted, got %v", err)
	}
}

func TestUpdate_ArchiveSucceedsEvenWhenIncomeReferences(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	incRepo := newFakeIncomeRepo()
	svc := NewService(catRepo, newFakeExpenseRepo(), incRepo)
	ctx := context.Background()

	cat := mustCreateCategory(t, svc, "Salary", "income")
	incRepo.countByCat[cat.ID] = 2

	if err := svc.Update(ctx, testUID, cat.ID, UpdateInput{Archived: ptr.Bool(true)}); err != nil {
		t.Fatalf("expected archive to succeed even with referencing income entries, got %v", err)
	}
	if len(incRepo.reassignCalls) != 0 {
		t.Fatalf("expected no reassignment without an explicit reassign_to, got %v", incRepo.reassignCalls)
	}
}

func TestUpdate_ArchiveWithReassignMovesIncomeFirst(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	incRepo := newFakeIncomeRepo()
	svc := NewService(catRepo, newFakeExpenseRepo(), incRepo)
	ctx := context.Background()

	from := mustCreateCategory(t, svc, "Salary", "income")
	to := mustCreateCategory(t, svc, "Freelance", "income")
	incRepo.countByCat[from.ID] = 4

	err := svc.Update(ctx, testUID, from.ID, UpdateInput{
		Archived:   ptr.Bool(true),
		ReassignTo: &to.ID,
	})
	if err != nil {
		t.Fatalf("expected archive-with-reassign to succeed, got %v", err)
	}

	if len(incRepo.reassignCalls) != 1 || incRepo.reassignCalls[0].from != from.ID || incRepo.reassignCalls[0].to != to.ID {
		t.Fatalf("expected exactly one reassign call from %v to %v, got %v", from.ID, to.ID, incRepo.reassignCalls)
	}
}

func TestUpdate_ReassignToDifferentTypeCategory_Rejected(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	expRepo := newFakeExpenseRepo()
	svc := NewService(catRepo, expRepo, newFakeIncomeRepo())
	ctx := context.Background()

	expenseCat := mustCreateCategory(t, svc, "Groceries", "expense")
	incomeCat := mustCreateCategory(t, svc, "Salary", "income")
	expRepo.countByCat[expenseCat.ID] = 1

	err := svc.Update(ctx, testUID, expenseCat.ID, UpdateInput{
		Archived:   ptr.Bool(true),
		ReassignTo: &incomeCat.ID,
	})
	if err == nil {
		t.Fatalf("expected reassigning an expense category to an income-type category to be rejected")
	}
	if !apperr.IsKind(err, apperr.Validation) {
		t.Fatalf("expected validation error kind, got %v", err)
	}
}
