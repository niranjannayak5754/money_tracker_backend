package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
)

type fakeExpenseService struct {
	lastUpdateInput expense.UpdateInput
}

func (f *fakeExpenseService) Create(ctx context.Context, userID common.UserID, in expense.CreateInput) (expense.Model, error) {
	return expense.Model{}, nil
}
func (f *fakeExpenseService) List(ctx context.Context, userID common.UserID, month, category string) ([]expense.Model, error) {
	return nil, nil
}
func (f *fakeExpenseService) Update(ctx context.Context, userID common.UserID, id common.ExpenseID, in expense.UpdateInput) error {
	f.lastUpdateInput = in
	return nil
}
func (f *fakeExpenseService) Delete(ctx context.Context, userID common.UserID, id common.ExpenseID) error {
	return nil
}

// Regression: PUT /expenses/{id} decoded Date as a plain *time.Time, which
// requires RFC3339 and rejects date-only strings that Create accepts via
// shared.FlexibleTime.
func TestExpenseUpdate_AcceptsDateOnlyString(t *testing.T) {
	svc := &fakeExpenseService{}
	h := NewExpenseHandler(svc, logging.New())

	req := httptest.NewRequest(http.MethodPut, "/exp-1", strings.NewReader(`{"date":"2026-07-04"}`))
	req = req.WithContext(requestctx.WithUID(req.Context(), "user-1"))
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if svc.lastUpdateInput.Date == nil {
		t.Fatalf("expected Date to be parsed, got nil")
	}
	want := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	if !svc.lastUpdateInput.Date.Equal(want) {
		t.Fatalf("expected date %v, got %v", want, *svc.lastUpdateInput.Date)
	}
}
