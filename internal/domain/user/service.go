package user

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/session"
	"golang.org/x/crypto/bcrypt"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
	"github.com/niranjannayak5754/money_tracker_backend/internal/security"
)

// validatePasswordPolicy enforces a minimum length plus letter+digit mix.
func validatePasswordPolicy(pw string) error {
	if len(pw) < config.MIN_PASSWORD_LENGTH {
		return apperr.ValidationErr(fmt.Sprintf("password must be at least %d characters", config.MIN_PASSWORD_LENGTH))
	}

	hasLetter, hasDigit := false, false
	for _, r := range pw {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return apperr.ValidationErr("password must contain at least one letter and one digit")
	}

	return nil
}

type Service interface {
	Register(ctx context.Context, in RegisterInput) (Model, error)
	Login(ctx context.Context, email, password string) (*Model, error)
	GetByID(ctx context.Context, id common.UserID) (*Model, error)
	ResetPassword(ctx context.Context, id common.UserID, oldPassword, newPassword string) error

	// IssueRefreshToken creates and persists a new refresh token for the
	// user, returning the plain value to hand back to the client.
	IssueRefreshToken(ctx context.Context, userID common.UserID) (string, error)

	// RotateRefreshToken validates a presented refresh token, revokes it,
	// and issues a replacement. Returns the token's owner and the new
	// plain refresh token.
	RotateRefreshToken(ctx context.Context, plainToken string) (common.UserID, string, error)

	// RevokeAllSessions invalidates every refresh token for the user —
	// used by logout and password change so a stolen token can't outlive
	// either action.
	RevokeAllSessions(ctx context.Context, userID common.UserID) error
}

type service struct {
	repo     Repository
	sessions session.Repository
}

func NewService(repo Repository, sessions session.Repository) Service {
	return &service{repo: repo, sessions: sessions}
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

	if err := validatePasswordPolicy(in.Password); err != nil {
		return Model{}, err
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

	id, err := s.repo.Create(ctx, u)
	if err != nil {
		return Model{}, apperr.ConflictErr("email already registered")
	}
	u.ID = id

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
			return nil, apperr.UnauthorizedErr("invalid email or password")
		}
		return nil, apperr.InternalErr("login error", err)
	}

	if bcrypt.CompareHashAndPassword(u.PassHash, []byte(password)) != nil {
		return nil, apperr.UnauthorizedErr("invalid email or password")
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
	oldPassword, newPassword string,
) error {
	if oldPassword == "" {
		return apperr.ValidationErr("current password required")
	}
	if newPassword == "" {
		return apperr.ValidationErr("new password required")
	}

	if err := validatePasswordPolicy(newPassword); err != nil {
		return err
	}

	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if repository.DataNotFoundErr(err) {
			return apperr.NotFoundErr("user not found")
		}
		return apperr.InternalErr("change password error", err)
	}

	if bcrypt.CompareHashAndPassword(u.PassHash, []byte(oldPassword)) != nil {
		return apperr.UnauthorizedErr("current password is incorrect")
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

	// Best-effort: the password change already succeeded; a session-revoke
	// failure shouldn't be reported as a failed password change.
	_ = s.sessions.RevokeAllForUser(ctx, id)

	return nil
}

func (s *service) IssueRefreshToken(ctx context.Context, userID common.UserID) (string, error) {
	plain, hash, err := security.GenerateOpaqueToken()
	if err != nil {
		return "", apperr.InternalErr("failed to generate refresh token", err)
	}

	rt := session.RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().AddDate(0, 0, config.REFRESH_TOKEN_TTL_DAYS),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.sessions.Create(ctx, rt); err != nil {
		return "", apperr.InternalErr("failed to store refresh token", err)
	}

	return plain, nil
}

func (s *service) RotateRefreshToken(ctx context.Context, plainToken string) (common.UserID, string, error) {
	if plainToken == "" {
		return "", "", apperr.ValidationErr("refresh token required")
	}

	hash := security.HashToken(plainToken)

	rt, err := s.sessions.FindValidByHash(ctx, hash)
	if err != nil {
		if repository.DataNotFoundErr(err) {
			return "", "", apperr.UnauthorizedErr("invalid or expired refresh token")
		}
		return "", "", apperr.InternalErr("refresh token lookup failed", err)
	}

	if err := s.sessions.RevokeByHash(ctx, hash); err != nil {
		return "", "", apperr.InternalErr("failed to revoke used refresh token", err)
	}

	newPlain, err := s.IssueRefreshToken(ctx, rt.UserID)
	if err != nil {
		return "", "", err
	}

	return rt.UserID, newPlain, nil
}

func (s *service) RevokeAllSessions(ctx context.Context, userID common.UserID) error {
	if err := s.sessions.RevokeAllForUser(ctx, userID); err != nil {
		return apperr.InternalErr("failed to revoke sessions", err)
	}
	return nil
}
