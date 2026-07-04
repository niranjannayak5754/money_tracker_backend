package recurring

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, m Model) (common.RecurringID, error)
	List(ctx context.Context, userID common.UserID) ([]Model, error)
	GetByID(ctx context.Context, userID common.UserID, id common.RecurringID) (*Model, error)
	Update(ctx context.Context, userID common.UserID, id common.RecurringID, set map[string]any) (bool, error)
	Delete(ctx context.Context, userID common.UserID, id common.RecurringID) (bool, error)

	// ListDue returns active, non-deleted templates (across all users)
	// whose NextRunDate is <= asOf — used by the background scheduler.
	ListDue(ctx context.Context, asOf time.Time) ([]Model, error)

	// ListUpcoming returns active, non-deleted templates (across all
	// users) whose NextRunDate falls strictly between from and to — used
	// by the notification scanner for "bill due soon" reminders, distinct
	// from ListDue's "materialize this now".
	ListUpcoming(ctx context.Context, from, to time.Time) ([]Model, error)
}
