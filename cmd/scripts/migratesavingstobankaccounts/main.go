package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
	mongoplatform "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
)

// migrateSavingsToBankAccounts moves every investment with type
// "savings_account" into the new bank_accounts collection: a savings
// account isn't really an investment (no return/risk profile), so this
// keeps "investments" meaning something that actually has one. Source
// investment records are soft-deleted, not hard-deleted, so the audit
// trail and any historical reporting stays intact.
func main() {
	logger := logging.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mc, err := mongoplatform.Connect(ctx, cfg.MongoURI)
	if err != nil {
		logger.Error("mongo connection failed", "err", err)
		os.Exit(1)
	}
	defer mc.Disconnect(context.Background())

	migrated, err := run(ctx, mc.Database(cfg.DBName), logger)
	if err != nil {
		os.Exit(1)
	}

	fmt.Printf("migration complete: %d record(s) migrated, savings_account marked inactive\n", migrated)
}

// run performs the migration against db and returns the number of
// investment records migrated. Split out from main for testability.
func run(ctx context.Context, db *mongo.Database, logger *slog.Logger) (int, error) {
	investmentsCol := db.Collection("investments")
	bankAccountsCol := db.Collection("bank_accounts")
	investmentTypesCol := db.Collection("investment_type")

	cur, err := investmentsCol.Find(ctx, bson.M{
		"type":       "savings_account",
		"deleted_at": bson.M{"$exists": false},
	})
	if err != nil {
		logger.Error("find savings_account investments failed", "err", err)
		return 0, err
	}
	defer cur.Close(ctx)

	var toMigrate []struct {
		ID         primitive.ObjectID   `bson:"_id"`
		UserID     primitive.ObjectID   `bson:"user_id"`
		Instrument string               `bson:"instrument,omitempty"`
		Amount     primitive.Decimal128 `bson:"amount"`
	}
	if err := cur.All(ctx, &toMigrate); err != nil {
		logger.Error("decode savings_account investments failed", "err", err)
		return 0, err
	}

	migrated := 0
	for _, inv := range toMigrate {
		name := inv.Instrument
		if name == "" {
			name = "Savings"
		}

		now := time.Now().UTC()
		_, err := bankAccountsCol.InsertOne(ctx, bson.M{
			"_id":        primitive.NewObjectID(),
			"user_id":    inv.UserID,
			"name":       name,
			"balance":    inv.Amount,
			"created_at": now,
			"updated_at": now,
		})
		if err != nil {
			logger.Error("insert bank_account failed", "investment_id", inv.ID.Hex(), "err", err)
			return migrated, err
		}

		_, err = investmentsCol.UpdateOne(ctx,
			bson.M{"_id": inv.ID},
			bson.M{"$set": bson.M{"deleted_at": now, "deleted_by": inv.UserID}},
		)
		if err != nil {
			logger.Error("soft-delete source investment failed", "investment_id", inv.ID.Hex(), "err", err)
			return migrated, err
		}

		migrated++
		logger.Info("migrated savings_account investment to bank_account", "investment_id", inv.ID.Hex(), "user_id", inv.UserID.Hex(), "name", name)
	}

	_, err = investmentTypesCol.UpdateOne(ctx,
		bson.M{"key": "savings_account"},
		bson.M{"$set": bson.M{"active": false}},
	)
	if err != nil {
		logger.Error("deactivate savings_account investment_type failed", "err", err)
		return migrated, err
	}

	return migrated, nil
}
