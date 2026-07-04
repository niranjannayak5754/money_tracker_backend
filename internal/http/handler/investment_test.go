package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/investment"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/logging"
)

type fakeInvestmentService struct {
	lastUpdateInput investment.UpdateInput
}

func (f *fakeInvestmentService) Create(ctx context.Context, userID common.UserID, in investment.CreateInput) (investment.Model, error) {
	return investment.Model{}, nil
}
func (f *fakeInvestmentService) List(ctx context.Context, userID common.UserID, month string) ([]investment.Model, error) {
	return nil, nil
}
func (f *fakeInvestmentService) Update(ctx context.Context, userID common.UserID, id common.InvestmentID, in investment.UpdateInput) error {
	f.lastUpdateInput = in
	return nil
}
func (f *fakeInvestmentService) Close(ctx context.Context, userID common.UserID, id common.InvestmentID, in investment.CloseInput) (investment.Model, error) {
	return investment.Model{}, nil
}
func (f *fakeInvestmentService) Delete(ctx context.Context, userID common.UserID, id common.InvestmentID) error {
	return nil
}
func (f *fakeInvestmentService) ListTypes(ctx context.Context) ([]investment.TypeDoc, error) {
	return nil, nil
}

// Regression: PUT /investment/{id} decoded Date as a plain *time.Time,
// which requires RFC3339 and rejects date-only strings that Create accepts
// via shared.FlexibleTime.
func TestInvestmentUpdate_AcceptsDateOnlyString(t *testing.T) {
	svc := &fakeInvestmentService{}
	h := NewInvestmentHandler(svc, logging.New())

	req := httptest.NewRequest(http.MethodPut, "/inv-1", strings.NewReader(`{"date":"2026-07-04"}`))
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
