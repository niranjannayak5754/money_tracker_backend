package debt

import (
	"context"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

// closeEpsilon absorbs rounding noise from amounts persisted to 2 decimal
// places, mirroring investment.Close's treatment of a fully-paid balance.
const closeEpsilon = 0.005

type Service interface {
	Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error)
	List(ctx context.Context, userID common.UserID) ([]Model, error)
	Update(ctx context.Context, userID common.UserID, id common.DebtID, in UpdateInput) error
	RecordPayment(ctx context.Context, userID common.UserID, id common.DebtID, in PaymentInput) (Model, error)
	Delete(ctx context.Context, userID common.UserID, id common.DebtID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type CreateInput struct {
	Name         string
	Principal    float64
	InterestRate float64
	EMIAmount    float64
	TenureMonths int
	StartDate    time.Time
	Notes        string
}

type UpdateInput struct {
	Name         *string
	InterestRate *float64
	EMIAmount    *float64
	TenureMonths *int
	Notes        *string
}

// PaymentInput records a single EMI/prepayment. PrincipalComponent and
// InterestComponent are supplied by the caller from their lender's
// amortization schedule — the app has no way to derive the split itself.
type PaymentInput struct {
	PrincipalComponent float64
	InterestComponent  float64
	Date               time.Time
	Notes              string
}

func (s *service) Create(ctx context.Context, userID common.UserID, in CreateInput) (Model, error) {
	if in.Name == "" {
		return Model{}, apperr.ValidationErr("name is required")
	}
	if in.Principal <= 0 {
		return Model{}, apperr.ValidationErr("principal must be greater than zero")
	}
	if in.InterestRate < 0 {
		return Model{}, apperr.ValidationErr("interest rate cannot be negative")
	}

	now := time.Now().UTC()
	d := Model{
		UserID:             userID,
		Name:               in.Name,
		Principal:          in.Principal,
		InterestRate:       in.InterestRate,
		EMIAmount:          in.EMIAmount,
		TenureMonths:       in.TenureMonths,
		StartDate:          shared.ChooseDate(in.StartDate),
		OutstandingBalance: in.Principal,
		TotalInterestPaid:  0,
		Status:             StatusActive,
		Notes:              in.Notes,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	id, err := s.repo.Create(ctx, d)
	if err != nil {
		return Model{}, apperr.InternalErr("failed to create debt", err)
	}
	d.ID = id

	return d, nil
}

func (s *service) List(ctx context.Context, userID common.UserID) ([]Model, error) {
	items, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, apperr.InternalErr("failed to list debts", err)
	}
	if items == nil {
		return []Model{}, nil
	}
	return items, nil
}

func (s *service) Update(ctx context.Context, userID common.UserID, id common.DebtID, in UpdateInput) error {
	set := map[string]any{"updated_at": time.Now().UTC()}

	if in.Name != nil {
		if *in.Name == "" {
			return apperr.ValidationErr("name cannot be empty")
		}
		set["name"] = *in.Name
	}
	if in.InterestRate != nil {
		if *in.InterestRate < 0 {
			return apperr.ValidationErr("interest rate cannot be negative")
		}
		set["interest_rate"] = *in.InterestRate
	}
	if in.EMIAmount != nil {
		set["emi_amount"] = *in.EMIAmount
	}
	if in.TenureMonths != nil {
		set["tenure_months"] = *in.TenureMonths
	}
	if in.Notes != nil {
		set["notes"] = *in.Notes
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return apperr.InternalErr("failed to update debt", err)
	}
	if !updated {
		return apperr.NotFoundErr("debt not found")
	}
	return nil
}

func (s *service) RecordPayment(ctx context.Context, userID common.UserID, id common.DebtID, in PaymentInput) (Model, error) {
	if in.PrincipalComponent <= 0 {
		return Model{}, apperr.ValidationErr("principal component must be greater than zero")
	}
	if in.InterestComponent < 0 {
		return Model{}, apperr.ValidationErr("interest component cannot be negative")
	}

	current, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		if repository.DataNotFoundErr(err) {
			return Model{}, apperr.NotFoundErr("debt not found")
		}
		return Model{}, apperr.InternalErr("failed to load debt", err)
	}

	if current.Status == StatusClosed {
		return Model{}, apperr.ValidationErr("debt is already closed")
	}

	if in.PrincipalComponent > current.OutstandingBalance+closeEpsilon {
		return Model{}, apperr.ValidationErr("principal component cannot exceed the outstanding balance")
	}

	newOutstanding := current.OutstandingBalance - in.PrincipalComponent
	newInterestPaid := current.TotalInterestPaid + in.InterestComponent

	newStatus := StatusActive
	if newOutstanding <= closeEpsilon {
		newOutstanding = 0
		newStatus = StatusClosed
	}

	set := map[string]any{
		"outstanding_balance": newOutstanding,
		"total_interest_paid": newInterestPaid,
		"status":              string(newStatus),
		"updated_at":          time.Now().UTC(),
	}

	updated, err := s.repo.Update(ctx, userID, id, set)
	if err != nil {
		return Model{}, apperr.InternalErr("failed to record payment", err)
	}
	if !updated {
		return Model{}, apperr.NotFoundErr("debt not found")
	}

	current.OutstandingBalance = newOutstanding
	current.TotalInterestPaid = newInterestPaid
	current.Status = newStatus

	return *current, nil
}

func (s *service) Delete(ctx context.Context, userID common.UserID, id common.DebtID) error {
	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return apperr.InternalErr("failed to delete debt", err)
	}
	if !deleted {
		return apperr.NotFoundErr("debt not found")
	}
	return nil
}
