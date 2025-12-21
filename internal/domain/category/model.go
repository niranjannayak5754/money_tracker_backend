package category

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Model struct {
	ID        common.CategoryID `json:"id"`
	UserID    common.UserID     `json:"user_id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"` // expense | income
	Archived  bool              `json:"archived"`
	CreatedAt time.Time         `json:"created_at"`
}
