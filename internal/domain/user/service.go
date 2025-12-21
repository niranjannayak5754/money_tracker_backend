package user

import (
	"context"
	"strings"
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"golang.org/x/crypto/bcrypt"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

type Service interface {
	Register(ctx context.Context, in RegisterInput) (Model, error)
	Login(ctx context.Context, email, password string) (*Model, error)
	GetByID(ctx context.Context, id common.UserID) (*Model, error)
	ResetPassword(ctx context.Context, id common.UserID, newPassword string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

type RegisterInput struct {
	Email    string
	Password string
}

func (s *service) Register(
	ctx context.Context,
	in RegisterInput,
) (Model, error) {

	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" || !strings.Contains(email, "@") {
		return Model{}, apperr.ValidationErr("invalid email")
	}

	if len(in.Password) < 6 {
		return Model{}, apperr.ValidationErr("password must be at least 6 characters")
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(in.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return Model{}, apperr.InternalErr("password hashing failed", err)
	}

	u := Model{
		ID:        "",
		Email:     email,
		PassHash:  hash,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return Model{}, apperr.ConflictErr("email already registered")
	}

	return u, nil
}

func (s *service) Login(
	ctx context.Context,
	email, password string,
) (*Model, error) {

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return nil, apperr.ValidationErr("email and password required")
	}

	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if repository.DataNotFoundErr(err) {
			return nil, apperr.UnauthorizedErr("invalid email")
		}
		return nil, apperr.InternalErr("login error", err)
	}

	if bcrypt.CompareHashAndPassword(u.PassHash, []byte(password)) != nil {
		return nil, apperr.UnauthorizedErr("invalid password")
	}

	return u, nil
}

func (s *service) GetByID(
	ctx context.Context,
	id common.UserID,
) (*Model, error) {

	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if repository.DataNotFoundErr(err) {
			return nil, apperr.NotFoundErr("user not found")
		}
		return nil, apperr.InternalErr("get user by id error", err)
	}

	return u, nil
}

func (s *service) ResetPassword(
	ctx context.Context,
	id common.UserID,
	newPassword string,
) error {
	if newPassword == "" {
		return apperr.ValidationErr("new password required")
	}

	if len(newPassword) < config.SIX {
		return apperr.ValidationErr("password must be at least 6 characters")
	}

	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if repository.DataNotFoundErr(err) {
			return apperr.NotFoundErr("user not found")
		}
		return apperr.InternalErr("change password error", err)
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperr.InternalErr("password hashing failed", err)
	}

	if err := s.repo.UpdatePasswordHash(ctx, id, newHash); err != nil {
		if repository.DataNotFoundErr(err) {
			return apperr.NotFoundErr("user not found")
		}
		return apperr.InternalErr("update password failed", err)
	}

	return nil
}
