package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/notification"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/recurring"
)

const billLookahead = 3 * 24 * time.Hour

// NotificationScanner is a read-only background sweep — it never mutates
// recurring templates, only raises in-app notifications (no email/push/SMS)
// via CreateIfNotExists, which dedupes against any existing non-dismissed
// notification for the same entity.
type NotificationScanner struct {
	notifications notification.Service
	recurring     recurring.Service
	logger        *slog.Logger
}

func NewNotificationScanner(
	notifications notification.Service,
	recurring recurring.Service,
	logger *slog.Logger,
) *NotificationScanner {
	return &NotificationScanner{
		notifications: notifications,
		recurring:     recurring,
		logger:        logger.With("component", "notification_scanner"),
	}
}

func (s *NotificationScanner) RunOnce(ctx context.Context) {
	now := time.Now().UTC()

	s.scanUpcomingBills(ctx, now)
}

func (s *NotificationScanner) scanUpcomingBills(ctx context.Context, now time.Time) {
	upcoming, err := s.recurring.ListUpcoming(ctx, now, now.Add(billLookahead))
	if err != nil {
		s.logger.Error("list upcoming recurring templates failed", "err", err)
		return
	}

	for _, tmpl := range upcoming {
		if tmpl.EntityType != recurring.EntityExpense {
			continue
		}
		created, err := s.notifications.CreateIfNotExists(ctx, tmpl.UserID, notification.CreateInput{
			Type:              notification.TypeBillDue,
			Title:             "Upcoming bill",
			Message:           fmt.Sprintf("A recurring payment is due on %s", tmpl.NextRunDate.Format("2006-01-02")),
			RelatedEntityType: "recurring_template",
			RelatedEntityID:   string(tmpl.ID),
		})
		if err != nil {
			s.logger.Error("create bill-due notification failed", "template_id", tmpl.ID, "err", err)
			continue
		}
		if created {
			s.logger.Info("created bill-due notification", "template_id", tmpl.ID, "user_id", tmpl.UserID)
		}
	}
}
