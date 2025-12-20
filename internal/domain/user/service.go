package user

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

type Service interface {
	Register(ctx context.Context, in RegisterInput) (Model, error)
	Login(ctx context.Context, email, password string) (*Model, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*Model, error)
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
		ID:        primitive.NewObjectID(),
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
	id primitive.ObjectID,
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
