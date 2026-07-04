package incomerepo

import (
	"context"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/income"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository/testutil"
)

func TestUpdate_ConvertsAmountToDecimal128AndRoundTrips(t *testing.T) {
	db := testutil.ConnectTestDB(t)
	repo := New(db, logging.New())
	ctx := context.Background()

	uid := common.UserID("507f1f77bcf86cd799439011")
	rec := income.Model{
		UserID: uid,
		Amount: 100.00,
		Date:   time.Now().UTC(),
		Source: "salary",
	}
	id, err := repo.Create(ctx, rec)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Regression: Update used to write a raw float64 into the amount field
	// instead of Decimal128, which broke decoding on the next List call.
	ok, err := repo.Update(ctx, uid, id, map[string]any{"amount": 250.75})
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
	if len(items) != 1 || items[0].Amount != 250.75 {
		t.Fatalf("expected amount 250.75 after update, got %+v", items)
	}
}

func TestUpdate_SoftDeletedRecordNotEditable(t *testing.T) {
	db := testutil.ConnectTestDB(t)
	repo := New(db, logging.New())
	ctx := context.Background()

	uid := common.UserID("507f1f77bcf86cd799439011")
	rec := income.Model{
		UserID: uid,
		Amount: 500,
		Date:   time.Now().UTC(),
		Source: "bonus",
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
	ok, err := repo.Update(ctx, uid, id, map[string]any{"amount": 999.0})
	if err != nil {
		t.Fatalf("update on soft-deleted record errored unexpectedly: %v", err)
	}
	if ok {
		t.Fatalf("expected update on soft-deleted record to report no match, but it succeeded")
	}
}
