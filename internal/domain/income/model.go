package income

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Model struct {
	ID        primitive.ObjectID   `bson:"_id" json:"id"`
	UserID    primitive.ObjectID   `bson:"user_id" json:"user_id"`
	Amount    primitive.Decimal128 `bson:"amount" json:"-"`
	AmountF   float64              `bson:"-" json:"amount"`
	Date      time.Time            `bson:"date" json:"date"`
	Source    string               `bson:"source,omitempty" json:"source,omitempty"`
	Notes     string               `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time            `bson:"updated_at" json:"updated_at"`
}
