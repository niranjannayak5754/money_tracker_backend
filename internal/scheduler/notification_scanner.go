package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/investment"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/notification"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/recurring"
)

const (
	maturityLookahead = 7 * 24 * time.Hour
	billLookahead     = 3 * 24 * time.Hour
)

// NotificationScanner is a read-only background sweep — it never mutates
// investments or recurring templates, only raises in-app notifications
// (no email/push/SMS) via CreateIfNotExists, which dedupes against any
// existing non-dismissed notification for the same entity.
type NotificationScanner struct {
	notifications notification.Service
	investments   investment.Service
	recurring     recurring.Service
	logger        *slog.Logger
}

func NewNotificationScanner(
	notifications notification.Service,
	investments investment.Service,
	recurring recurring.Service,
	logger *slog.Logger,
) *NotificationScanner {
	return &NotificationScanner{
		notifications: notifications,
		investments:   investments,
		recurring:     recurring,
		logger:        logger.With("component", "notification_scanner"),
	}
}

func (s *NotificationScanner) RunOnce(ctx context.Context) {
	now := time.Now().UTC()

	s.scanMaturingInvestments(ctx, now)
	s.scanUpcomingBills(ctx, now)
}

func (s *NotificationScanner) scanMaturingInvestments(ctx context.Context, now time.Time) {
	maturing, err := s.investments.ListMaturingBefore(ctx, now.Add(maturityLookahead))
	if err != nil {
		s.logger.Error("list maturing investments failed", "err", err)
		return
	}

	for _, inv := range maturing {
		if inv.MaturityDate == nil {
			continue
		}
		created, err := s.notifications.CreateIfNotExists(ctx, inv.UserID, notification.CreateInput{
			Type:              notification.TypeFDMaturity,
			Title:             "Investment maturing soon",
			Message:           fmt.Sprintf("%s (%s) matures on %s", inv.DisplayType, investmentLabel(inv), inv.MaturityDate.Format("2006-01-02")),
			RelatedEntityType: "investment",
			RelatedEntityID:   string(inv.ID),
		})
		if err != nil {
			s.logger.Error("create maturity notification failed", "investment_id", inv.ID, "err", err)
			continue
		}
		if created {
			s.logger.Info("created maturity notification", "investment_id", inv.ID, "user_id", inv.UserID)
		}
	}
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

func investmentLabel(inv investment.Model) string {
	if inv.Instrument != "" {
		return inv.Instrument
	}
	return string(inv.ID)
}
