package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidPassword = errors.New("password must be at least 6 characters")
	ErrUserNotFound    = errors.New("user not found")
	ErrBadCredentials  = errors.New("invalid credentials")
)

type Model struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	Email     string             `bson:"email" json:"email"`
	PassHash  []byte             `bson:"pass_hash" json:"-"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

//
// Repository
//

type Repository interface {
	Insert(ctx any, u Model) error
	FindByEmail(ctx any, email string) (*Model, error)
	FindByID(ctx any, id primitive.ObjectID) (*Model, error)
}

//
// Service interface (domain business logic only)
//

type Service interface {
	Register(ctx context.Context, input RegisterInput) (Model, error)
	Login(ctx context.Context, email, password string) (*Model, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*Model, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

//
// Input DTOs
//

type RegisterInput struct {
	Email    string
	Password string
}

//
// Business Logic
//

func (s *service) Register(ctx context.Context, in RegisterInput) (Model, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	if email == "" || !strings.Contains(email, "@") {
		return Model{}, ErrInvalidEmail
	}

	if len(in.Password) < 6 {
		return Model{}, ErrInvalidPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return Model{}, err
	}

	u := Model{
		ID:        primitive.NewObjectID(),
		Email:     email,
		PassHash:  hash,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Insert(ctx, u); err != nil {
		return Model{}, err
	}

	return u, nil
}

func (s *service) Login(ctx context.Context, email, password string) (*Model, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, ErrBadCredentials
	}

	if bcrypt.CompareHashAndPassword(u.PassHash, []byte(password)) != nil {
		return nil, ErrBadCredentials
	}

	return u, nil
}

func (s *service) GetByID(ctx context.Context, id primitive.ObjectID) (*Model, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}
