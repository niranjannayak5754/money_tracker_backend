package recurring

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

// --- fake repository (in-memory, no DB dependency) ---

type fakeRecurringRepo struct {
	mu     sync.Mutex
	items  map[common.RecurringID]*Model
	nextID int
}

func newFakeRecurringRepo() *fakeRecurringRepo {
	return &fakeRecurringRepo{items: map[common.RecurringID]*Model{}}
}

func (f *fakeRecurringRepo) Create(ctx context.Context, m Model) (common.RecurringID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := common.RecurringID(fmt.Sprintf("rec-%d", f.nextID))
	m.ID = id
	cp := m
	f.items[id] = &cp
	return id, nil
}

func (f *fakeRecurringRepo) List(ctx context.Context, userID common.UserID) ([]Model, error) {
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

func (f *fakeRecurringRepo) GetByID(ctx context.Context, userID common.UserID, id common.RecurringID) (*Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return nil, repository.ErrNotFound
	}
	cp := *m
	return &cp, nil
}

func (f *fakeRecurringRepo) Update(ctx context.Context, userID common.UserID, id common.RecurringID, set map[string]any) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return false, nil
	}
	for k, v := range set {
		switch k {
		case "day_of_month":
			m.DayOfMonth = v.(int)
		case "payload":
			m.Payload = v.(map[string]any)
		case "end_date":
			d := v.(time.Time)
			m.EndDate = &d
		case "active":
			m.Active = v.(bool)
		case "last_run_date":
			d := v.(time.Time)
			m.LastRunDate = &d
		case "next_run_date":
			m.NextRunDate = v.(time.Time)
		}
	}
	return true, nil
}

func (f *fakeRecurringRepo) Delete(ctx context.Context, userID common.UserID, id common.RecurringID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return false, nil
	}
	m.Active = false
	return true, nil
}

func (f *fakeRecurringRepo) ListDue(ctx context.Context, asOf time.Time) ([]Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Model
	for _, m := range f.items {
		if m.Active && !m.NextRunDate.After(asOf) {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (f *fakeRecurringRepo) ListUpcoming(ctx context.Context, from, to time.Time) ([]Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Model
	for _, m := range f.items {
		if m.Active && m.NextRunDate.After(from) && !m.NextRunDate.After(to) {
			out = append(out, *m)
		}
	}
	return out, nil
}

// --- tests ---

const testUID = common.UserID("user-1")

func mustCreateTemplate(t *testing.T, svc Service, entity EntityType, start time.Time, endDate *time.Time) Model {
	t.Helper()
	payload := map[string]any{"amount": 500.0}
	switch entity {
	case EntityExpense:
		payload["category_id"] = "cat-1"
	case EntityInvestment:
		payload["type"] = "fixed_deposit"
	}

	m, err := svc.Create(context.Background(), testUID, CreateInput{
		EntityType: entity,
		Frequency:  FrequencyMonthly,
		DayOfMonth: start.Day(),
		Payload:    payload,
		StartDate:  start,
		EndDate:    endDate,
	})
	if err != nil {
		t.Fatalf("create template failed: %v", err)
	}
	return m
}

func TestListDue_DeactivatesTemplatesPastEndDate(t *testing.T) {
	repo := newFakeRecurringRepo()
	svc := NewService(repo)
	ctx := context.Background()

	past := time.Now().UTC().AddDate(0, -1, 0)
	end := past.AddDate(0, 0, 5) // end date is before "now", so this occurrence is past-end
	tmpl := mustCreateTemplate(t, svc, EntityExpense, past, &end)

	// Force NextRunDate past the EndDate to simulate an overdue template.
	_, err := repo.Update(ctx, testUID, tmpl.ID, map[string]any{"next_run_date": time.Now().UTC()})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	due, err := svc.ListDue(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("list due failed: %v", err)
	}
	for _, d := range due {
		if d.ID == tmpl.ID {
			t.Fatalf("expected past-end-date template to be excluded from due list")
		}
	}

	got, err := repo.GetByID(ctx, testUID, tmpl.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if got.Active {
		t.Fatalf("expected past-end-date template to be deactivated as a side effect")
	}
}

func TestListDue_ReturnsInBoundsTemplates(t *testing.T) {
	repo := newFakeRecurringRepo()
	svc := NewService(repo)
	ctx := context.Background()

	past := time.Now().UTC().AddDate(0, -1, 0)
	tmpl := mustCreateTemplate(t, svc, EntityIncome, past, nil)

	due, err := svc.ListDue(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("list due failed: %v", err)
	}
	found := false
	for _, d := range due {
		if d.ID == tmpl.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected in-bounds overdue template to be returned as due")
	}
}

func TestMarkRun_AdvancesNextRunDateAndSetsLastRunDate(t *testing.T) {
	repo := newFakeRecurringRepo()
	svc := NewService(repo)
	ctx := context.Background()

	start := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	tmpl := mustCreateTemplate(t, svc, EntityIncome, start, nil)

	ranAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	if err := svc.MarkRun(ctx, testUID, tmpl.ID, ranAt); err != nil {
		t.Fatalf("mark run failed: %v", err)
	}

	got, err := repo.GetByID(ctx, testUID, tmpl.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}

	wantNext := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	if !got.NextRunDate.Equal(wantNext) {
		t.Fatalf("expected next run date %v, got %v", wantNext, got.NextRunDate)
	}
	if got.LastRunDate == nil || !got.LastRunDate.Equal(ranAt) {
		t.Fatalf("expected last run date %v, got %v", ranAt, got.LastRunDate)
	}
	if !got.Active {
		t.Fatalf("expected template to remain active with no end date")
	}
}

func TestMarkRun_DeactivatesWhenNextExceedsEndDate(t *testing.T) {
	repo := newFakeRecurringRepo()
	svc := NewService(repo)
	ctx := context.Background()

	start := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC) // next occurrence (Feb 15) exceeds this
	tmpl := mustCreateTemplate(t, svc, EntityIncome, start, &end)

	if err := svc.MarkRun(ctx, testUID, tmpl.ID, start); err != nil {
		t.Fatalf("mark run failed: %v", err)
	}

	got, err := repo.GetByID(ctx, testUID, tmpl.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if got.Active {
		t.Fatalf("expected template to be deactivated once its next occurrence would exceed end_date")
	}
}

func TestUpdate_EditingPayloadDoesNotTouchSchedule(t *testing.T) {
	repo := newFakeRecurringRepo()
	svc := NewService(repo)
	ctx := context.Background()

	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	tmpl := mustCreateTemplate(t, svc, EntityIncome, start, nil)
	originalNextRun := tmpl.NextRunDate

	newPayload := map[string]any{"amount": 999.0, "source": "salary"}
	if err := svc.Update(ctx, testUID, tmpl.ID, UpdateInput{Payload: newPayload}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	got, err := repo.GetByID(ctx, testUID, tmpl.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if got.Payload["amount"] != 999.0 {
		t.Fatalf("expected payload to be updated, got %v", got.Payload)
	}
	if !got.NextRunDate.Equal(originalNextRun) {
		t.Fatalf("expected NextRunDate to be untouched by a payload edit, got %v vs original %v", got.NextRunDate, originalNextRun)
	}
}
