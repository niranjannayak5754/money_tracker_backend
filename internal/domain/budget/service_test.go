package budget

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

// --- fake repository (in-memory, no DB dependency) ---

type fakeBudgetRepo struct {
	mu     sync.Mutex
	items  map[common.BudgetID]*Model
	nextID int
}

func newFakeBudgetRepo() *fakeBudgetRepo {
	return &fakeBudgetRepo{items: map[common.BudgetID]*Model{}}
}

func sameCategory(a, b *common.CategoryID) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// Set emulates the real Mongo repo's upsert-by-(user,category,effective_from)
// semantics, so service-level tests can verify Set doesn't duplicate rows.
func (f *fakeBudgetRepo) Set(ctx context.Context, m Model) (common.BudgetID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for id, existing := range f.items {
		if existing.UserID == m.UserID && sameCategory(existing.CategoryID, m.CategoryID) && existing.EffectiveFrom == m.EffectiveFrom {
			existing.Amount = m.Amount
			existing.UpdatedAt = m.UpdatedAt
			return id, nil
		}
	}

	f.nextID++
	id := common.BudgetID(fmt.Sprintf("budget-%d", f.nextID))
	cp := m
	cp.ID = id
	f.items[id] = &cp
	return id, nil
}

func (f *fakeBudgetRepo) List(ctx context.Context, userID common.UserID) ([]Model, error) {
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

func (f *fakeBudgetRepo) Delete(ctx context.Context, userID common.UserID, id common.BudgetID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return false, nil
	}
	delete(f.items, id)
	return true, nil
}

// --- tests ---

const testUID = common.UserID("user-1")

func categoryID(s string) *common.CategoryID {
	c := common.CategoryID(s)
	return &c
}

func TestResolveForMonth_PicksLatestEffectiveFromNotJustNewestRowOverall(t *testing.T) {
	repo := newFakeBudgetRepo()
	svc := NewService(repo)
	ctx := context.Background()

	// Insert out of chronological order: the row created most recently
	// (2026-01) has an EARLIER EffectiveFrom than the one created first
	// (2026-06) — resolution must go by EffectiveFrom, not insertion order.
	if _, err := svc.Set(ctx, testUID, SetInput{CategoryID: categoryID("cat-1"), Amount: 6000, EffectiveFrom: "2026-06"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if _, err := svc.Set(ctx, testUID, SetInput{CategoryID: categoryID("cat-1"), Amount: 5000, EffectiveFrom: "2026-01"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	resolvedMarch, err := svc.ResolveForMonth(ctx, testUID, "2026-03")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if got := resolvedMarch.ByCategory[common.CategoryID("cat-1")]; got != 5000 {
		t.Fatalf("expected March to resolve to the Jan-effective budget (5000), got %v", got)
	}

	resolvedJuly, err := svc.ResolveForMonth(ctx, testUID, "2026-07")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if got := resolvedJuly.ByCategory[common.CategoryID("cat-1")]; got != 6000 {
		t.Fatalf("expected July to resolve to the Jun-effective budget (6000), got %v", got)
	}
}

func TestResolveForMonth_UnbudgetedCategoryNotInResult(t *testing.T) {
	repo := newFakeBudgetRepo()
	svc := NewService(repo)
	ctx := context.Background()

	if _, err := svc.Set(ctx, testUID, SetInput{CategoryID: categoryID("cat-1"), Amount: 1000, EffectiveFrom: "2026-01"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	resolved, err := svc.ResolveForMonth(ctx, testUID, "2026-07")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if _, ok := resolved.ByCategory[common.CategoryID("cat-2")]; ok {
		t.Fatalf("expected unbudgeted category to be absent from the result, not a zero entry")
	}
	if len(resolved.ByCategory) != 1 {
		t.Fatalf("expected exactly 1 budgeted category, got %d", len(resolved.ByCategory))
	}
}

func TestResolveForMonth_OverallAndPerCategoryIndependent(t *testing.T) {
	repo := newFakeBudgetRepo()
	svc := NewService(repo)
	ctx := context.Background()

	if _, err := svc.Set(ctx, testUID, SetInput{CategoryID: nil, Amount: 20000, EffectiveFrom: "2026-01"}); err != nil {
		t.Fatalf("set overall failed: %v", err)
	}
	if _, err := svc.Set(ctx, testUID, SetInput{CategoryID: categoryID("cat-1"), Amount: 3000, EffectiveFrom: "2026-01"}); err != nil {
		t.Fatalf("set category failed: %v", err)
	}

	resolved, err := svc.ResolveForMonth(ctx, testUID, "2026-07")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if resolved.Overall == nil || *resolved.Overall != 20000 {
		t.Fatalf("expected overall budget 20000, got %v", resolved.Overall)
	}
	if got := resolved.ByCategory[common.CategoryID("cat-1")]; got != 3000 {
		t.Fatalf("expected cat-1 budget 3000, got %v", got)
	}
}

func TestResolveForMonth_BeforeAnyEffectiveDate_NotBudgeted(t *testing.T) {
	repo := newFakeBudgetRepo()
	svc := NewService(repo)
	ctx := context.Background()

	if _, err := svc.Set(ctx, testUID, SetInput{CategoryID: categoryID("cat-1"), Amount: 1000, EffectiveFrom: "2026-06"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	resolved, err := svc.ResolveForMonth(ctx, testUID, "2026-01")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if _, ok := resolved.ByCategory[common.CategoryID("cat-1")]; ok {
		t.Fatalf("expected no budget to apply before its effective_from month")
	}
}

func TestSet_SameMonthUpsertsRatherThanDuplicating(t *testing.T) {
	repo := newFakeBudgetRepo()
	svc := NewService(repo)
	ctx := context.Background()

	first, err := svc.Set(ctx, testUID, SetInput{CategoryID: categoryID("cat-1"), Amount: 1000, EffectiveFrom: "2026-01"})
	if err != nil {
		t.Fatalf("first set failed: %v", err)
	}
	second, err := svc.Set(ctx, testUID, SetInput{CategoryID: categoryID("cat-1"), Amount: 1500, EffectiveFrom: "2026-01"})
	if err != nil {
		t.Fatalf("second set failed: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected the same budget row to be reused for the same (category, effective_from), got %v vs %v", first.ID, second.ID)
	}

	rows, err := svc.List(ctx, testUID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 row after upserting the same month twice, got %d", len(rows))
	}
	if rows[0].Amount != 1500 {
		t.Fatalf("expected amount to be updated to 1500, got %v", rows[0].Amount)
	}
}
