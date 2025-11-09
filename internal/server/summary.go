package server

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/httpx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/contextutils"
)

func (c *Container) Summary(w http.ResponseWriter, r *http.Request) {
	uidHex := contextutils.UID(r.Context())
	uid, _ := primitive.ObjectIDFromHex(uidHex)

	// derive month range
	month := r.URL.Query().Get("month")
	start, end := shared.MonthRange(month)

	incomeTotal, err := aggregateTotal(r.Context(), c.DB.Collection("income"), uid, start, end)
	if err != nil {
		httpx.ServerErr(w, err)
		return
	}

	expTotal, catBreakdown, err := aggregateExpenses(r.Context(), c.DB.Collection("expenses"), uid, start, end)
	if err != nil {
		httpx.ServerErr(w, err)
		return
	}

	httpx.JSON(w, 200, map[string]any{
		"income_total":       incomeTotal,
		"expense_total":      expTotal,
		"savings":            incomeTotal - expTotal,
		"category_breakdown": catBreakdown,
	})
}

//
// Helpers
//

func aggregateTotal(ctx context.Context, col *mongo.Collection, uid primitive.ObjectID, start, end time.Time) (float64, error) {
	cur, err := col.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.M{
			"user_id": uid,
			"date": bson.M{
				"$gte": start,
				"$lt":  end,
			},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$amount"},
		}}},
	})
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	var out struct {
		Total primitive.Decimal128 `bson:"total"`
	}

	if cur.Next(ctx) {
		if err := cur.Decode(&out); err != nil {
			return 0, err
		}
		return shared.Decimal128ToFloat(out.Total), nil
	}

	return 0, nil
}

func aggregateExpenses(ctx context.Context, col *mongo.Collection, uid primitive.ObjectID, start, end time.Time) (float64, []map[string]any, error) {
	cur, err := col.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.M{
			"user_id": uid,
			"date": bson.M{
				"$gte": start,
				"$lt":  end,
			},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   "$category_id",
			"total": bson.M{"$sum": "$amount"},
		}}},
		// join with categories
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "categories",
			"localField":   "_id",
			"foreignField": "_id",
			"as":           "cat",
		}}},
		// unwind cat array
		bson.D{{Key: "$unwind", Value: bson.M{
			"path":                       "$cat",
			"preserveNullAndEmptyArrays": true,
		}}},
	})
	if err != nil {
		return 0, nil, err
	}
	defer cur.Close(ctx)

	var total float64
	var cats []map[string]any

	for cur.Next(ctx) {
		var x struct {
			CatID primitive.ObjectID   `bson:"_id"`
			Total primitive.Decimal128 `bson:"total"`
			Cat   struct {
				Name string `bson:"name"`
			} `bson:"cat"`
		}
		if err := cur.Decode(&x); err != nil {
			return 0, nil, err
		}

		f := shared.Decimal128ToFloat(x.Total)
		total += f

		cats = append(cats, map[string]any{
			"category_id":   x.CatID.Hex(),
			"category_name": x.Cat.Name, // ✅ now filled
			"total":         f,
		})
	}

	return total, cats, nil
}
