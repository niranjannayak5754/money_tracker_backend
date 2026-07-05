package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/notification"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/recurring"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
)

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

func TestNotificationScanner_CreatesBillDueNotificationForUpcomingExpenseTemplate(t *testing.T) {
	nextRun := time.Now().UTC().AddDate(0, 0, 2)
	recSvc := &fakeRecurringService{upcoming: []recurring.Model{
		{ID: "rec-1", UserID: "user-1", EntityType: recurring.EntityExpense, NextRunDate: nextRun},
	}}
	notifSvc := &fakeNotificationService{}

	scanner := NewNotificationScanner(notifSvc, recSvc, logging.New())
	scanner.RunOnce(context.Background())

	if len(notifSvc.createCalls) != 1 {
		t.Fatalf("expected 1 bill-due notification, got %d", len(notifSvc.createCalls))
	}
	got := notifSvc.createCalls[0]
	if got.Type != notification.TypeBillDue || got.RelatedEntityType != "recurring_template" || got.RelatedEntityID != "rec-1" {
		t.Fatalf("unexpected notification: %+v", got)
	}
}

func TestNotificationScanner_DoesNotDuplicateAcrossTicks(t *testing.T) {
	nextRun := time.Now().UTC().AddDate(0, 0, 2)
	recSvc := &fakeRecurringService{upcoming: []recurring.Model{
		{ID: "rec-1", UserID: "user-1", EntityType: recurring.EntityExpense, NextRunDate: nextRun},
	}}
	notifSvc := &fakeNotificationService{}

	scanner := NewNotificationScanner(notifSvc, recSvc, logging.New())

	// Simulate two ticker runs, as would happen an hour apart.
	scanner.RunOnce(context.Background())
	scanner.RunOnce(context.Background())

	if len(notifSvc.createCalls) != 1 {
		t.Fatalf("expected exactly 1 notification across two ticker runs, got %d", len(notifSvc.createCalls))
	}
}

func TestNotificationScanner_IgnoresNonExpenseRecurringTemplates(t *testing.T) {
	nextRun := time.Now().UTC().AddDate(0, 0, 2)
	recSvc := &fakeRecurringService{upcoming: []recurring.Model{
		{ID: "rec-other", UserID: "user-1", EntityType: recurring.EntityType("something_else"), NextRunDate: nextRun},
	}}
	notifSvc := &fakeNotificationService{}

	scanner := NewNotificationScanner(notifSvc, recSvc, logging.New())
	scanner.RunOnce(context.Background())

	if len(notifSvc.createCalls) != 0 {
		t.Fatalf("expected no bill-due notifications for non-expense templates, got %d: %+v", len(notifSvc.createCalls), notifSvc.createCalls)
	}
}

func TestNotificationScanner_NoActivity_CreatesNothing(t *testing.T) {
	recSvc := &fakeRecurringService{}
	notifSvc := &fakeNotificationService{}

	scanner := NewNotificationScanner(notifSvc, recSvc, logging.New())
	scanner.RunOnce(context.Background())

	if len(notifSvc.createCalls) != 0 {
		t.Fatalf("expected no notifications when nothing is upcoming, got %d", len(notifSvc.createCalls))
	}
}
