package main

import (
	"context"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
)

// investmentTypes are the seed values for the `investment_type` collection,
// read by internal/repository/investment/mongo.go's GetTypes.
var investmentTypes = []struct {
	Key  string
	Name string
}{
	{"gold", "Gold"},
	{"savings_account", "Savings Account"},
	{"mutual_fund_sip", "Mutual Fund SIP"},
	{"recurring_deposit", "Recurring Deposit (RD)"},
	{"fixed_deposit", "Fixed Deposit (FD)"},
	{"stocks_equity", "Stocks & Equity"},
}

func main() {
	logger := logging.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mc, err := mongo.Connect(ctx, cfg.MongoURI)
	if err != nil {
		logger.Error("mongo connection failed", "err", err)
		os.Exit(1)
	}
	defer mc.Disconnect(context.Background())

	col := mc.Database(cfg.DBName).Collection("investment_type")

	for _, t := range investmentTypes {
		_, err := col.UpdateOne(
			ctx,
			bson.M{"key": t.Key},
			bson.M{
				"$set": bson.M{
					"name":   t.Name,
					"active": true,
				},
				"$setOnInsert": bson.M{
					"created_at": time.Now().UTC(),
				},
			},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			logger.Error("upsert investment_type failed", "key", t.Key, "err", err)
			os.Exit(1)
		}
		logger.Info("seeded investment_type", "key", t.Key, "name", t.Name)
	}

	logger.Info("investment_type seed complete", "count", len(investmentTypes))
}
