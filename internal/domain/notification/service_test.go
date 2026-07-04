package notification

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

// --- fake repository (in-memory, no DB dependency) ---

type fakeNotificationRepo struct {
	mu     sync.Mutex
	items  map[common.NotificationID]*Model
	nextID int
}

func newFakeNotificationRepo() *fakeNotificationRepo {
	return &fakeNotificationRepo{items: map[common.NotificationID]*Model{}}
}

func (f *fakeNotificationRepo) Create(ctx context.Context, m Model) (common.NotificationID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := common.NotificationID(fmt.Sprintf("notif-%d", f.nextID))
	m.ID = id
	cp := m
	f.items[id] = &cp
	return id, nil
}

func (f *fakeNotificationRepo) List(ctx context.Context, userID common.UserID, onlyUnread bool) ([]Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Model
	for _, m := range f.items {
		if m.UserID != userID || m.DismissedAt != nil {
			continue
		}
		if onlyUnread && m.ReadAt != nil {
			continue
		}
		out = append(out, *m)
	}
	return out, nil
}

func (f *fakeNotificationRepo) UnreadCount(ctx context.Context, userID common.UserID) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var count int64
	for _, m := range f.items {
		if m.UserID == userID && m.DismissedAt == nil && m.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

func (f *fakeNotificationRepo) MarkRead(ctx context.Context, userID common.UserID, id common.NotificationID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return false, nil
	}
	if m.ReadAt == nil {
		now := time.Now().UTC()
		m.ReadAt = &now
	}
	return true, nil
}

func (f *fakeNotificationRepo) Dismiss(ctx context.Context, userID common.UserID, id common.NotificationID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.items[id]
	if !ok || m.UserID != userID {
		return false, nil
	}
	if m.DismissedAt == nil {
		now := time.Now().UTC()
		m.DismissedAt = &now
	}
	return true, nil
}

func (f *fakeNotificationRepo) ExistsActive(ctx context.Context, userID common.UserID, notifType Type, relatedEntityType, relatedEntityID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range f.items {
		if m.UserID == userID && m.Type == notifType && m.RelatedEntityType == relatedEntityType && m.RelatedEntityID == relatedEntityID && m.DismissedAt == nil {
			return true, nil
		}
	}
	return false, nil
}

// --- tests ---

const testUID = common.UserID("user-1")

func TestCreateIfNotExists_DedupesUntilDismissed(t *testing.T) {
	repo := newFakeNotificationRepo()
	svc := NewService(repo)
	ctx := context.Background()

	in := CreateInput{
		Type:              TypeBudgetExceeded,
		Title:             "Budget exceeded",
		RelatedEntityType: "category",
		RelatedEntityID:   "cat-1:2026-07",
	}

	created, err := svc.CreateIfNotExists(ctx, testUID, in)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if !created {
		t.Fatalf("expected first call to create a notification")
	}

	created, err = svc.CreateIfNotExists(ctx, testUID, in)
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}
	if created {
		t.Fatalf("expected second call to be deduped (not created)")
	}

	items, err := svc.List(ctx, testUID, false)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 notification to exist, got %d", len(items))
	}
}

func TestCreateIfNotExists_CreatesAgainAfterDismiss(t *testing.T) {
	repo := newFakeNotificationRepo()
	svc := NewService(repo)
	ctx := context.Background()

	in := CreateInput{
		Type:              TypeFDMaturity,
		Title:             "Investment maturing soon",
		RelatedEntityType: "investment",
		RelatedEntityID:   "inv-1",
	}

	created, err := svc.CreateIfNotExists(ctx, testUID, in)
	if err != nil || !created {
		t.Fatalf("expected first call to create, created=%v err=%v", created, err)
	}

	items, err := svc.List(ctx, testUID, false)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected 1 notification, got %d, err=%v", len(items), err)
	}
	if err := svc.Dismiss(ctx, testUID, items[0].ID); err != nil {
		t.Fatalf("dismiss failed: %v", err)
	}

	created, err = svc.CreateIfNotExists(ctx, testUID, in)
	if err != nil {
		t.Fatalf("create after dismiss failed: %v", err)
	}
	if !created {
		t.Fatalf("expected a fresh notification to be created after the previous one was dismissed")
	}
}

func TestMarkRead_ReducesUnreadCount(t *testing.T) {
	repo := newFakeNotificationRepo()
	svc := NewService(repo)
	ctx := context.Background()

	n1, err := svc.Create(ctx, testUID, CreateInput{Type: TypeBillDue, Title: "Bill 1"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := svc.Create(ctx, testUID, CreateInput{Type: TypeBillDue, Title: "Bill 2"}); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	count, err := svc.UnreadCount(ctx, testUID)
	if err != nil || count != 2 {
		t.Fatalf("expected unread count 2, got %d, err=%v", count, err)
	}

	if err := svc.MarkRead(ctx, testUID, n1.ID); err != nil {
		t.Fatalf("mark read failed: %v", err)
	}

	count, err = svc.UnreadCount(ctx, testUID)
	if err != nil || count != 1 {
		t.Fatalf("expected unread count 1 after marking one read, got %d, err=%v", count, err)
	}
}

func TestDismiss_RemovesFromList(t *testing.T) {
	repo := newFakeNotificationRepo()
	svc := NewService(repo)
	ctx := context.Background()

	n, err := svc.Create(ctx, testUID, CreateInput{Type: TypeBillDue, Title: "Bill"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if err := svc.Dismiss(ctx, testUID, n.ID); err != nil {
		t.Fatalf("dismiss failed: %v", err)
	}

	items, err := svc.List(ctx, testUID, false)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected dismissed notification to be excluded from list, got %+v", items)
	}
}

func TestList_UnreadFilterExcludesRead(t *testing.T) {
	repo := newFakeNotificationRepo()
	svc := NewService(repo)
	ctx := context.Background()

	n1, err := svc.Create(ctx, testUID, CreateInput{Type: TypeBillDue, Title: "Bill 1"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := svc.Create(ctx, testUID, CreateInput{Type: TypeBillDue, Title: "Bill 2"}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := svc.MarkRead(ctx, testUID, n1.ID); err != nil {
		t.Fatalf("mark read failed: %v", err)
	}

	unread, err := svc.List(ctx, testUID, true)
	if err != nil {
		t.Fatalf("list unread failed: %v", err)
	}
	if len(unread) != 1 {
		t.Fatalf("expected 1 unread notification, got %d", len(unread))
	}

	all, err := svc.List(ctx, testUID, false)
	if err != nil {
		t.Fatalf("list all failed: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 total (non-dismissed) notifications, got %d", len(all))
	}
}
