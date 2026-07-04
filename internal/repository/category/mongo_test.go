package categoryrepo

import (
	"context"
	"testing"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository/testutil"
	"go.mongodb.org/mongo-driver/bson"
)

func TestCreateAndUpdate_WriteAuditEntries(t *testing.T) {
	db := testutil.ConnectTestDB(t)
	repo := New(db, logging.New())
	ctx := context.Background()

	uid := common.UserID("507f1f77bcf86cd799439011")
	cat := category.Model{
		UserID: uid,
		Name:   "Groceries",
		Type:   "expense",
	}

	// Regression: category previously had zero audit logging, unlike every
	// other domain (expense/income/investment).
	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	items, err := repo.List(ctx, uid, nil)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected 1 category after create, got %d, err=%v", len(items), err)
	}
	id := items[0].ID

	auditCol := db.Collection("audit_logs")

	createCount, err := auditCol.CountDocuments(ctx, bson.M{"entity": "category", "entity_id": string(id), "action": "create"})
	if err != nil {
		t.Fatalf("count create audit entries failed: %v", err)
	}
	if createCount != 1 {
		t.Fatalf("expected 1 create audit entry for category, got %d", createCount)
	}

	newName := "Groceries & Household"
	ok, err := repo.Update(ctx, uid, id, map[string]any{"name": newName})
	if err != nil || !ok {
		t.Fatalf("update failed: ok=%v err=%v", ok, err)
	}

	updateCount, err := auditCol.CountDocuments(ctx, bson.M{"entity": "category", "entity_id": string(id), "action": "update"})
	if err != nil {
		t.Fatalf("count update audit entries failed: %v", err)
	}
	if updateCount != 1 {
		t.Fatalf("expected 1 update audit entry for category, got %d", updateCount)
	}
}
