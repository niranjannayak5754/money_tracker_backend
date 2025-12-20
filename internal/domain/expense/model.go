package expense

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Model struct {
	ID         primitive.ObjectID   `bson:"_id" json:"id"`
	UserID     primitive.ObjectID   `bson:"user_id" json:"user_id"`
	Amount     primitive.Decimal128 `bson:"amount" json:"-"`
	AmountF    float64              `bson:"-" json:"amount"`
	Date       time.Time            `bson:"date" json:"date"`
	CategoryID primitive.ObjectID   `bson:"category_id" json:"category_id"`
	Merchant   string               `bson:"merchant,omitempty" json:"merchant,omitempty"`
	Notes      string               `bson:"notes,omitempty" json:"notes,omitempty"`
	Tags       []string             `bson:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt  time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time            `bson:"updated_at" json:"updated_at"`
}
