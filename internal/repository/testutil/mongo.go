// Package testutil provides a real-Mongo test database for repository-level
// regression tests. Tests using ConnectTestDB are skipped unless MONGO_TEST_URI
// is set, so `go test ./...` stays fast and safe to run without a database.
package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectTestDB connects to MONGO_TEST_URI and returns a uniquely named
// database that is dropped when the test completes.
func ConnectTestDB(t *testing.T) *mongo.Database {
	t.Helper()

	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set, skipping repository regression test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect to MONGO_TEST_URI failed: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping MONGO_TEST_URI failed: %v", err)
	}

	dbName := fmt.Sprintf("money_tracker_test_%d", time.Now().UnixNano())
	db := client.Database(dbName)

	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()
		_ = db.Drop(dropCtx)
		_ = client.Disconnect(dropCtx)
	})

	return db
}
