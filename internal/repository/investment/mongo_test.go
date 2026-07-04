package investmentrepo

import (
	"context"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/investment"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository/testutil"
)

func TestUpdate_ConvertsAmountToDecimal128AndRoundTrips(t *testing.T) {
	db := testutil.ConnectTestDB(t)
	repo := New(db, logging.New())
	ctx := context.Background()

	uid := common.UserID("507f1f77bcf86cd799439011")
	rec := investment.Model{
		UserID:      uid,
		Type:        "fixed_deposit",
		DisplayType: "Fixed Deposit (FD)",
		Amount:      10000,
		Status:      investment.StatusActive,
		Date:        time.Now().UTC(),
	}
	id, err := repo.Create(ctx, rec)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Regression: Update used to write a raw float64 into the amount field
	// instead of Decimal128, which broke decoding on the next List call.
	ok, err := repo.Update(ctx, uid, id, map[string]any{"amount": 15000.50})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !ok {
		t.Fatalf("update reported no match")
	}

	items, err := repo.ListByMonth(ctx, uid, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("list after update failed (Decimal128 type regression): %v", err)
	}
	if len(items) != 1 || items[0].Amount != 15000.50 {
		t.Fatalf("expected amount 15000.50 after update, got %+v", items)
	}
}

func TestUpdate_SoftDeletedRecordNotEditable(t *testing.T) {
	db := testutil.ConnectTestDB(t)
	repo := New(db, logging.New())
	ctx := context.Background()

	uid := common.UserID("507f1f77bcf86cd799439011")
	rec := investment.Model{
		UserID:      uid,
		Type:        "gold",
		DisplayType: "Gold",
		Amount:      20000,
		Status:      investment.StatusActive,
		Date:        time.Now().UTC(),
	}
	id, err := repo.Create(ctx, rec)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	deleted, err := repo.Delete(ctx, uid, id)
	if err != nil || !deleted {
		t.Fatalf("delete failed: deleted=%v err=%v", deleted, err)
	}

	// Regression: Update previously filtered only on {_id, user_id}, so a
	// soft-deleted record could still be edited via PUT.
	ok, err := repo.Update(ctx, uid, id, map[string]any{"amount": 1.0})
	if err != nil {
		t.Fatalf("update on soft-deleted record errored unexpectedly: %v", err)
	}
	if ok {
		t.Fatalf("expected update on soft-deleted record to report no match, but it succeeded")
	}
}
