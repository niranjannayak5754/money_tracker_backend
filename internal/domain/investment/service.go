package investment

import (
	"context"
	"fmt"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

type Service interface {
	Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error)
	List(ctx context.Context, userID common.UserID, month string) ([]Model, error)
	Update(ctx context.Context, userID common.UserID, id common.InvestmentID, in UpdateInput) error
	Close(ctx context.Context, userID common.UserID, id common.InvestmentID, in CloseInput) (Model, error)
	Delete(ctx context.Context, userID common.UserID, id common.InvestmentID) error
	ListTypes(ctx context.Context) ([]TypeDoc, error)

	// GetXIRR computes a single investment's annualized return.
	GetXIRR(ctx context.Context, userID common.UserID, id common.InvestmentID) (float64, error)

	// GetPortfolioXIRR computes a blended annualized return across all of
	// a user's investments.
	GetPortfolioXIRR(ctx context.Context, userID common.UserID) (float64, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type CreateInput struct {
	Type       string
	Instrument string
	Amount     float64
	Date       time.Time
	Notes      string
}

type UpdateInput struct {
	Type       *string
	Instrument *string
	Amount     *float64
	Date       *time.Time
	Notes      *string
}

// CloseInput records a full or partial withdrawal from an investment.
// CostBasisConsumed is however much of the original invested amount this
// withdrawal represents — the caller supplies it explicitly since the
// backend has no live market pricing to infer it from.
type CloseInput struct {
	WithdrawnAmount   float64
	CostBasisConsumed float64
	Date              time.Time
	Notes             string
}

func (s *service) Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error) {
	if in.Amount <= 0 {
		return Model{}, apperr.ValidationErr("amount must be greater than zero")
	}

	displayType, err := s.validateType(ctx, in.Type)
	if err != nil {
		return Model{}, err
	}

	now := time.Now().UTC()

	rec := Model{
		ID:              "",
		UserID:          userID,
		Type:            in.Type,
		DisplayType:     displayType,
		Instrument:      in.Instrument,
		Amount:          in.Amount,
		RetrievedAmount: 0,
		RealizedPnl:     0,
		Status:          StatusActive,
		Date:            shared.ChooseDate(in.Date),
		Notes:           in.Notes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	id, err := s.repo.Create(ctx, rec)
	if err != nil {
		return Model{}, apperr.InternalErr("failed to create investment", err)
	}
	rec.ID = id

	return rec, nil
}

func (s *service) List(ctx context.Context, userID common.UserID, month string) ([]Model, error) {
	start, end, err := shared.MonthRange(month)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.ListByMonth(ctx, userID, start, end)
	if err != nil {
		return nil, apperr.InternalErr("failed to list investments", err)
	}

	if items == nil {
		return []Model{}, nil
	}
	return items, nil
}

func (s *service) Update(ctx context.Context, userID common.UserID, id common.InvestmentID, in UpdateInput) error {
	set := map[string]any{"updated_at": time.Now().UTC()}

	if in.Type != nil {
		displayType, err := s.validateType(ctx, *in.Type)
		if err != nil {
			return err
		}
		set["type"] = *in.Type
		set["display_type"] = displayType
	}
	if in.Instrument != nil {
		set["instrument"] = *in.Instrument
	}
	if in.Amount != nil {
		if *in.Amount <= 0 {
			return apperr.ValidationErr("amount must be greater than zero")
		}
		set["amount"] = *in.Amount
	}
	if in.Date != nil {
		set["date"] = shared.ChooseDate(*in.Date)
	}
	if in.Notes != nil {
		set["notes"] = *in.Notes
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return apperr.InternalErr("failed to update investment", err)
	}
	if !updated {
		return apperr.NotFoundErr("investment not found")
	}
	return nil
}

// closeEpsilon absorbs float/decimal rounding noise (amounts are persisted
// rounded to 2 decimal places) so a fully-consumed remaining basis of e.g.
// 0.004 due to rounding is still treated as fully closed.
const closeEpsilon = 0.005

// Close records a full or partial withdrawal from an investment: the
// withdrawn amount and however much of the original cost basis it consumes.
// Realized PnL is the difference between what was withdrawn and the cost
// basis consumed; the investment's remaining Amount is reduced accordingly,
// and its Status transitions to PARTIALLY_RETRIEVED or CLOSED.
func (s *service) Close(ctx context.Context, userID common.UserID, id common.InvestmentID, in CloseInput) (Model, error) {
	if in.WithdrawnAmount <= 0 {
		return Model{}, apperr.ValidationErr("withdrawn amount must be greater than zero")
	}
	if in.CostBasisConsumed <= 0 {
		return Model{}, apperr.ValidationErr("cost basis consumed must be greater than zero")
	}

	current, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		if repository.DataNotFoundErr(err) {
			return Model{}, apperr.NotFoundErr("investment not found")
		}
		return Model{}, apperr.InternalErr("failed to load investment", err)
	}

	if current.Status == StatusClosed {
		return Model{}, apperr.ValidationErr("investment is already closed")
	}

	if in.CostBasisConsumed > current.Amount+closeEpsilon {
		return Model{}, apperr.ValidationErr("cost basis consumed cannot exceed the investment's remaining amount")
	}

	newAmount := current.Amount - in.CostBasisConsumed
	newRetrieved := current.RetrievedAmount + in.WithdrawnAmount
	newRealizedPnl := current.RealizedPnl + (in.WithdrawnAmount - in.CostBasisConsumed)

	newStatus := StatusPartiallyRetrieved
	if newAmount <= closeEpsilon {
		newAmount = 0
		newStatus = StatusClosed
	}

	set := map[string]any{
		"amount":           newAmount,
		"retrieved_amount": newRetrieved,
		"realized_pnl":     newRealizedPnl,
		"status":           string(newStatus),
		"updated_at":       time.Now().UTC(),
	}
	if in.Notes != "" {
		closeDate := shared.ChooseDate(in.Date)
		note := fmt.Sprintf("[closed %s] %s", closeDate.Format(config.STANDARD_DATE), in.Notes)
		if current.Notes != "" {
			note = current.Notes + "\n" + note
		}
		set["notes"] = note
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return Model{}, apperr.InternalErr("failed to close investment", err)
	}
	if !updated {
		return Model{}, apperr.NotFoundErr("investment not found")
	}

	current.Amount = newAmount
	current.RetrievedAmount = newRetrieved
	current.RealizedPnl = newRealizedPnl
	current.Status = newStatus
	if v, ok := set["notes"].(string); ok {
		current.Notes = v
	}

	return *current, nil
}

func (s *service) ListTypes(ctx context.Context) ([]TypeDoc, error) {
	types, err := s.repo.GetTypes(ctx)
	if err != nil {
		return nil, apperr.InternalErr("failed to fetch investment types", err)
	}

	if types == nil {
		return []TypeDoc{}, nil
	}

	return types, nil
}

func (s *service) validateType(ctx context.Context, typ string) (string, error) {
	if typ == "" {
		return "", apperr.ValidationErr("type is required")
	}
	types, err := s.repo.GetTypes(ctx)
	if err != nil {
		return "", apperr.InternalErr("failed to validate investment type", err)
	}
	if len(types) == 0 {
		return "", apperr.ValidationErr("no investment types available")
	}
	for _, t := range types {
		if t.Key == typ && t.Active {
			return t.Name, nil
		}
	}
	return "", apperr.ValidationErr("invalid investment type")
}

func (s *service) Delete(ctx context.Context, userID common.UserID, id common.InvestmentID) error {
	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to delete investment", err)
	}
	if !deleted {
		return apperr.NotFoundErr("investment not found")
	}
	return nil
}

// cashFlows builds a simplified 2-point cash-flow pair for an investment:
// the original invested amount as an outflow at its creation date, and its
// current total value (still-held remaining basis plus everything already
// withdrawn) as an inflow as of asOf.
//
// Known limitation: there's no dated ledger of individual withdrawal
// events, only running totals (Amount/RetrievedAmount/RealizedPnl), so
// multiple partial withdrawals at different real dates are all folded into
// one terminal cash flow on asOf. This understates the true XIRR when
// significant withdrawals happened well before asOf, since it assumes the
// withdrawn money stayed invested longer than it actually did.
func cashFlows(inv Model, asOf time.Time) []shared.CashFlow {
	originalInvested := inv.Amount + inv.RetrievedAmount - inv.RealizedPnl
	currentValue := inv.Amount + inv.RetrievedAmount

	return []shared.CashFlow{
		{Date: inv.Date, Amount: -originalInvested},
		{Date: asOf, Amount: currentValue},
	}
}

func (s *service) GetXIRR(ctx context.Context, userID common.UserID, id common.InvestmentID) (float64, error) {
	inv, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		if repository.DataNotFoundErr(err) {
			return 0, apperr.NotFoundErr("investment not found")
		}
		return 0, apperr.InternalErr("failed to load investment", err)
	}

	rate, err := shared.XIRR(cashFlows(*inv, time.Now().UTC()))
	if err != nil {
		return 0, apperr.ValidationErr("unable to compute XIRR for this investment: " + err.Error())
	}
	return rate, nil
}

func (s *service) GetPortfolioXIRR(ctx context.Context, userID common.UserID) (float64, error) {
	items, err := s.repo.ListByMonth(ctx, userID, time.Time{}, time.Time{})
	if err != nil {
		return 0, apperr.InternalErr("failed to load investments", err)
	}

	now := time.Now().UTC()
	var flows []shared.CashFlow
	for _, inv := range items {
		flows = append(flows, cashFlows(inv, now)...)
	}

	if len(flows) < 2 {
		return 0, apperr.ValidationErr("not enough investment activity to compute a portfolio return")
	}

	rate, err := shared.XIRR(flows)
	if err != nil {
		return 0, apperr.ValidationErr("unable to compute portfolio XIRR: " + err.Error())
	}
	return rate, nil
}
