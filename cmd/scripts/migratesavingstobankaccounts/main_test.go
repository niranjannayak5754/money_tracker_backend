package main

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository/testutil"
)

func TestRun_MigratesSavingsAccountToBankAccount(t *testing.T) {
	db := testutil.ConnectTestDB(t)
	ctx := context.Background()
	logger := logging.New()

	userID := primitive.NewObjectID()
	investmentID := primitive.NewObjectID()
	amount, _ := primitive.ParseDecimal128("15000.00")

	_, err := db.Collection("investments").InsertOne(ctx, bson.M{
		"_id":        investmentID,
		"user_id":    userID,
		"type":       "savings_account",
		"instrument": "HDFC Savings",
		"amount":     amount,
		"status":     "ACTIVE",
		"date":       time.Now().UTC(),
		"created_at": time.Now().UTC(),
		"updated_at": time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed investment failed: %v", err)
	}

	_, err = db.Collection("investment_type").InsertOne(ctx, bson.M{
		"_id":        primitive.NewObjectID(),
		"key":        "savings_account",
		"name":       "Savings Account",
		"active":     true,
		"created_at": time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed investment_type failed: %v", err)
	}

	migrated, err := run(ctx, db, logger)
	if err != nil {
		t.Fatalf("migration run failed: %v", err)
	}
	if migrated != 1 {
		t.Fatalf("expected 1 record migrated, got %d", migrated)
	}

	var bankAccount struct {
		UserID  primitive.ObjectID   `bson:"user_id"`
		Name    string               `bson:"name"`
		Balance primitive.Decimal128 `bson:"balance"`
	}
	if err := db.Collection("bank_accounts").FindOne(ctx, bson.M{"user_id": userID}).Decode(&bankAccount); err != nil {
		t.Fatalf("expected a bank_account to be created: %v", err)
	}
	if bankAccount.Name != "HDFC Savings" {
		t.Fatalf("expected bank account name 'HDFC Savings', got %q", bankAccount.Name)
	}
	if bankAccount.Balance.String() != amount.String() {
		t.Fatalf("expected balance %v, got %v", amount, bankAccount.Balance)
	}

	var sourceInvestment struct {
		DeletedAt *time.Time `bson:"deleted_at"`
	}
	if err := db.Collection("investments").FindOne(ctx, bson.M{"_id": investmentID}).Decode(&sourceInvestment); err != nil {
		t.Fatalf("find source investment failed: %v", err)
	}
	if sourceInvestment.DeletedAt == nil {
		t.Fatalf("expected source investment to be soft-deleted")
	}

	var typeDoc struct {
		Active bool `bson:"active"`
	}
	if err := db.Collection("investment_type").FindOne(ctx, bson.M{"key": "savings_account"}).Decode(&typeDoc); err != nil {
		t.Fatalf("find investment_type failed: %v", err)
	}
	if typeDoc.Active {
		t.Fatalf("expected savings_account investment_type to be marked inactive")
	}

	// Running again must be a no-op (source already soft-deleted).
	migratedAgain, err := run(ctx, db, logger)
	if err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}
	if migratedAgain != 0 {
		t.Fatalf("expected second run to migrate 0 records, got %d", migratedAgain)
	}
}
