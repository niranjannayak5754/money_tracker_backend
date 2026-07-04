package summaryrepo

import (
	"context"
	"math"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
	expenserepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository/testutil"
)

func TestExpenseTotals_PrecisionMatchesExactSum(t *testing.T) {
	db := testutil.ConnectTestDB(t)
	logger := logging.New()
	expRepo := expenserepo.New(db, logger)
	sumRepo := New(db, logger)
	ctx := context.Background()

	uid := common.UserID("507f1f77bcf86cd799439011")
	catA := common.CategoryID(primitive.NewObjectID().Hex())
	catB := common.CategoryID(primitive.NewObjectID().Hex())
	now := time.Now().UTC()

	amounts := []struct {
		amount float64
		catID  common.CategoryID
	}{
		{10.10, catA},
		{10.20, catA},
		{10.30, catB},
	}

	for _, a := range amounts {
		if _, err := expRepo.Create(ctx, expense.Model{
			UserID:     uid,
			Amount:     a.amount,
			Date:       now,
			CategoryID: a.catID,
		}); err != nil {
			t.Fatalf("create expense failed: %v", err)
		}
	}

	// Regression: the grand total used to be summed in Go from
	// already-rounded per-category float64s, risking float drift.
	total, breakdown, err := sumRepo.ExpenseTotals(ctx, uid, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ExpenseTotals failed: %v", err)
	}

	const want = 30.60
	if math.Abs(total-want) > 0.0001 {
		t.Fatalf("expected total %.2f, got %.10f", want, total)
	}
	if len(breakdown) != 2 {
		t.Fatalf("expected 2 category breakdown entries, got %d", len(breakdown))
	}
}
