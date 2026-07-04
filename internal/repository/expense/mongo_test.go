package expenserepo

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository/testutil"
)

func TestUpdate_SoftDeletedRecordNotEditable(t *testing.T) {
	db := testutil.ConnectTestDB(t)
	repo := New(db, logging.New())
	ctx := context.Background()

	uid := common.UserID("507f1f77bcf86cd799439011")
	catID := common.CategoryID(primitive.NewObjectID().Hex())
	rec := expense.Model{
		UserID:     uid,
		Amount:     42.50,
		Date:       time.Now().UTC(),
		CategoryID: catID,
		Merchant:   "Test Store",
	}
	id, err := repo.Create(ctx, rec)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if id == "" {
		t.Fatalf("expected Create to return a generated expense id")
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
