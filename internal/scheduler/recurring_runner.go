// Package scheduler orchestrates background jobs that span multiple
// domains — it belongs above the domain layer since materializing a
// recurring template means calling another domain's own Create (never
// writing that domain's collection directly).
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/income"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/investment"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/recurring"
)

type RecurringRunner struct {
	recurringSvc  recurring.Service
	expenseSvc    expense.Service
	incomeSvc     income.Service
	investmentSvc investment.Service
	logger        *slog.Logger
}

func NewRecurringRunner(
	recurringSvc recurring.Service,
	expenseSvc expense.Service,
	incomeSvc income.Service,
	investmentSvc investment.Service,
	logger *slog.Logger,
) *RecurringRunner {
	return &RecurringRunner{
		recurringSvc:  recurringSvc,
		expenseSvc:    expenseSvc,
		incomeSvc:     incomeSvc,
		investmentSvc: investmentSvc,
		logger:        logger.With("component", "recurring_runner"),
	}
}

// Run ticks every interval until ctx is done, materializing due templates.
// Tolerant of restarts: each tick only asks "what's due right now", so a
// missed period is picked up (one occurrence per tick) on the first tick
// after the process comes back up, rather than requiring precise timing.
func (r *RecurringRunner) Run(ctx context.Context, interval time.Duration) {
	r.RunOnce(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.RunOnce(ctx)
		}
	}
}

// RunOnce performs a single pass: materialize every currently-due template.
// Exported so it can be driven directly (by tests, or an on-demand trigger)
// without waiting on the ticker.
func (r *RecurringRunner) RunOnce(ctx context.Context) {
	due, err := r.recurringSvc.ListDue(ctx, time.Now().UTC())
	if err != nil {
		r.logger.Error("list due templates failed", "err", err)
		return
	}

	for _, tmpl := range due {
		if err := r.materialize(ctx, tmpl); err != nil {
			r.logger.Error("materialize failed", "template_id", tmpl.ID, "entity_type", tmpl.EntityType, "err", err)
			continue
		}
		r.logger.Info("materialized recurring template", "template_id", tmpl.ID, "entity_type", tmpl.EntityType, "user_id", tmpl.UserID)
	}
}

func (r *RecurringRunner) materialize(ctx context.Context, tmpl recurring.Model) error {
	var err error

	switch tmpl.EntityType {
	case recurring.EntityExpense:
		_, err = r.expenseSvc.Create(ctx, tmpl.UserID, expense.CreateInput{
			Amount:     payloadFloat(tmpl.Payload, "amount"),
			Date:       tmpl.NextRunDate,
			CategoryID: common.CategoryID(payloadString(tmpl.Payload, "category_id")),
			Merchant:   payloadString(tmpl.Payload, "merchant"),
			Notes:      payloadString(tmpl.Payload, "notes"),
		})
	case recurring.EntityIncome:
		_, err = r.incomeSvc.Create(ctx, tmpl.UserID, income.CreateInput{
			Amount: payloadFloat(tmpl.Payload, "amount"),
			Date:   tmpl.NextRunDate,
			Source: payloadString(tmpl.Payload, "source"),
			Notes:  payloadString(tmpl.Payload, "notes"),
		})
	case recurring.EntityInvestment:
		_, err = r.investmentSvc.Create(ctx, tmpl.UserID, investment.CreateInput{
			Type:       payloadString(tmpl.Payload, "type"),
			Instrument: payloadString(tmpl.Payload, "instrument"),
			Amount:     payloadFloat(tmpl.Payload, "amount"),
			Date:       tmpl.NextRunDate,
			Notes:      payloadString(tmpl.Payload, "notes"),
		})
	default:
		return fmt.Errorf("unknown entity_type %q", tmpl.EntityType)
	}

	if err != nil {
		return err
	}

	return r.recurringSvc.MarkRun(ctx, tmpl.UserID, tmpl.ID, time.Now().UTC())
}

func payloadFloat(payload map[string]any, key string) float64 {
	switch v := payload[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func payloadString(payload map[string]any, key string) string {
	s, _ := payload[key].(string)
	return s
}
