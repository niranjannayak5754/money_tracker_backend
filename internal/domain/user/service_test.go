package user

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/niranjannayak5754/money_tracker_backend/internal/apperr"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/session"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

// --- fakes (in-memory, no DB dependency — pure business-logic tests) ---

type fakeUserRepo struct {
	mu      sync.Mutex
	byID    map[common.UserID]*Model
	byEmail map[string]*Model
	nextID  int
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byID: map[common.UserID]*Model{}, byEmail: map[string]*Model{}}
}

func (f *fakeUserRepo) Create(ctx context.Context, u Model) (common.UserID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.byEmail[u.Email]; exists {
		return "", fmt.Errorf("duplicate email")
	}
	f.nextID++
	u.ID = common.UserID(fmt.Sprintf("user-%d", f.nextID))
	cp := u
	f.byID[u.ID] = &cp
	f.byEmail[u.Email] = &cp
	return u.ID, nil
}

func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byEmail[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id common.UserID) (*Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) UpdatePasswordHash(ctx context.Context, id common.UserID, hash []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	u.PassHash = hash
	return nil
}

type fakeSessionRepo struct {
	mu             sync.Mutex
	tokens         map[string]*session.RefreshToken
	revokeAllCalls []common.UserID
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{tokens: map[string]*session.RefreshToken{}}
}

func (f *fakeSessionRepo) Create(ctx context.Context, rt session.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := rt
	f.tokens[rt.TokenHash] = &cp
	return nil
}

func (f *fakeSessionRepo) FindValidByHash(ctx context.Context, tokenHash string) (*session.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rt, ok := f.tokens[tokenHash]
	if !ok || rt.RevokedAt != nil || time.Now().UTC().After(rt.ExpiresAt) {
		return nil, repository.ErrNotFound
	}
	cp := *rt
	return &cp, nil
}

func (f *fakeSessionRepo) RevokeByHash(ctx context.Context, tokenHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if rt, ok := f.tokens[tokenHash]; ok {
		now := time.Now().UTC()
		rt.RevokedAt = &now
	}
	return nil
}

func (f *fakeSessionRepo) RevokeAllForUser(ctx context.Context, userID common.UserID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revokeAllCalls = append(f.revokeAllCalls, userID)
	now := time.Now().UTC()
	for _, rt := range f.tokens {
		if rt.UserID == userID && rt.RevokedAt == nil {
			rt.RevokedAt = &now
		}
	}
	return nil
}

// --- test helpers ---

func mustCreateUser(t *testing.T, repo *fakeUserRepo, email, password string) common.UserID {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt hash failed: %v", err)
	}
	u := Model{Email: email, PassHash: hash, CreatedAt: time.Now().UTC()}
	id, err := repo.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return id
}

// --- regression tests ---

func TestLogin_UnknownEmailAndWrongPassword_SameErrorMessage(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewService(repo, newFakeSessionRepo())
	ctx := context.Background()

	mustCreateUser(t, repo, "known@example.com", "correctpass1")

	_, err1 := svc.Login(ctx, "doesnotexist@example.com", "whatever123")
	_, err2 := svc.Login(ctx, "known@example.com", "wrongpassword1")

	if err1 == nil || err2 == nil {
		t.Fatalf("expected both logins to fail, got err1=%v err2=%v", err1, err2)
	}
	// Regression: previously "invalid email" vs "invalid password" leaked
	// which branch failed, enabling user enumeration.
	if err1.Error() != err2.Error() {
		t.Fatalf("expected identical error messages, got %q vs %q", err1.Error(), err2.Error())
	}
}

func TestResetPassword_WrongOldPassword_Rejected(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewService(repo, newFakeSessionRepo())
	ctx := context.Background()

	uid := mustCreateUser(t, repo, "user@example.com", "correctpass1")

	err := svc.ResetPassword(ctx, uid, "wrongoldpass1", "newpassword1")
	if err == nil {
		t.Fatalf("expected error for wrong old password, got nil")
	}
	if !apperr.IsKind(err, apperr.Unauthorized) {
		t.Fatalf("expected unauthorized error kind, got %v", err)
	}

	// password must remain unchanged
	u, _ := repo.FindByID(ctx, uid)
	if bcrypt.CompareHashAndPassword(u.PassHash, []byte("correctpass1")) != nil {
		t.Fatalf("password was changed despite wrong old password")
	}
}

func TestResetPassword_CorrectOldPassword_RevokesAllSessions(t *testing.T) {
	repo := newFakeUserRepo()
	sessions := newFakeSessionRepo()
	svc := NewService(repo, sessions)
	ctx := context.Background()

	uid := mustCreateUser(t, repo, "user@example.com", "correctpass1")

	plain, err := svc.IssueRefreshToken(ctx, uid)
	if err != nil {
		t.Fatalf("issue refresh token failed: %v", err)
	}

	if err := svc.ResetPassword(ctx, uid, "correctpass1", "newpassword1"); err != nil {
		t.Fatalf("reset password failed: %v", err)
	}

	if len(sessions.revokeAllCalls) != 1 || sessions.revokeAllCalls[0] != uid {
		t.Fatalf("expected RevokeAllForUser to be called once for %v, got %v", uid, sessions.revokeAllCalls)
	}

	// the pre-reset refresh token must no longer be usable
	if _, _, err := svc.RotateRefreshToken(ctx, plain); err == nil {
		t.Fatalf("expected old refresh token to be rejected after password reset")
	}

	// new password must actually work
	u, _ := repo.FindByID(ctx, uid)
	if bcrypt.CompareHashAndPassword(u.PassHash, []byte("newpassword1")) != nil {
		t.Fatalf("new password was not persisted")
	}
}

func TestRefreshTokenRevocation_OldTokenRejectedAfterRevokeAll(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewService(repo, newFakeSessionRepo())
	ctx := context.Background()

	uid := mustCreateUser(t, repo, "user@example.com", "correctpass1")

	plain, err := svc.IssueRefreshToken(ctx, uid)
	if err != nil {
		t.Fatalf("issue refresh token failed: %v", err)
	}

	// Regression: this mirrors login -> logout -> refresh-with-old-token,
	// which previously succeeded because logout was a stateless no-op.
	if err := svc.RevokeAllSessions(ctx, uid); err != nil {
		t.Fatalf("revoke all sessions failed: %v", err)
	}

	if _, _, err := svc.RotateRefreshToken(ctx, plain); err == nil {
		t.Fatalf("expected refresh with a revoked token to fail")
	}
}

func TestRotateRefreshToken_InvalidatesOldTokenOnRotation(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewService(repo, newFakeSessionRepo())
	ctx := context.Background()

	uid := mustCreateUser(t, repo, "user@example.com", "correctpass1")

	plain, err := svc.IssueRefreshToken(ctx, uid)
	if err != nil {
		t.Fatalf("issue refresh token failed: %v", err)
	}

	newUID, newPlain, err := svc.RotateRefreshToken(ctx, plain)
	if err != nil {
		t.Fatalf("rotate refresh token failed: %v", err)
	}
	if newUID != uid {
		t.Fatalf("expected uid %v, got %v", uid, newUID)
	}
	if newPlain == plain {
		t.Fatalf("expected a new token value on rotation")
	}

	// reusing the old (rotated-away) token must fail
	if _, _, err := svc.RotateRefreshToken(ctx, plain); err == nil {
		t.Fatalf("expected reused rotated-out token to be rejected")
	}
}

func TestValidatePasswordPolicy(t *testing.T) {
	tests := []struct {
		name    string
		pw      string
		wantErr bool
	}{
		{"too short", "ab1", true},
		{"no digit", "abcdefgh", true},
		{"no letter", "12345678", true},
		{"valid", "abcdefg1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePasswordPolicy(tt.pw)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for password %q, got nil", tt.pw)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error for password %q, got %v", tt.pw, err)
			}
		})
	}
}
