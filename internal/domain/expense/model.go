package expense

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Model struct {
	ID         primitive.ObjectID   `json:"id"`
	UserID     primitive.ObjectID   `json:"user_id"`
	Amount     primitive.Decimal128 `bson:"amount" json:"-"`
	AmountF    float64              `bson:"-" json:"amount"`
	Date       time.Time            `json:"date"`
	CategoryID primitive.ObjectID   `json:"category_id"`
	Merchant   string               `json:"merchant,omitempty"`
	Notes      string               `json:"notes,omitempty"`
	Tags       []string             `json:"tags,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}
