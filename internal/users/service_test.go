package users

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeUserRepository struct {
	usersByEmail map[string]User
	createErr    error
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		usersByEmail: make(map[string]User),
	}
}

func (r *fakeUserRepository) Create(ctx context.Context, user User) error {
	if r.createErr != nil {
		return r.createErr
	}

	if _, exists := r.usersByEmail[user.Email]; exists {
		return ErrEmailAlreadyUsed
	}

	r.usersByEmail[user.Email] = user
	return nil
}

func (r *fakeUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user, ok := r.usersByEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

func (r *fakeUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	for _, user := range r.usersByEmail {
		if user.ID == id {
			return &user, nil
		}
	}

	return nil, ErrUserNotFound
}

type fakePasswordHasher struct{}

func newFakePasswordHasher() *fakePasswordHasher {
	return &fakePasswordHasher{}
}

func (h fakePasswordHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (h fakePasswordHasher) Compare(hash string, password string) error {
	if hash != "hashed:"+password {
		return errors.New("invalid password")
	}

	return nil
}

func TestService_Register(t *testing.T) {
	ctx := context.Background()

	repo := newFakeUserRepository()
	hasher := newFakePasswordHasher()
	service := NewService(repo, hasher)

	email := uuid.NewString() + "@mail.com"
	password := "valid-password"

	user, err := service.Register(ctx, email, password)
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	if user.ID == uuid.Nil {
		t.Fatalf("expected id to be generated")
	}

	if user.Email != email {
		t.Fatalf("expected emails: %s, got: %s", email, user.Email)
	}

	if user.PasswordHash != "hashed:"+password {
		t.Fatalf("expected password: %s, got: %s", "hashed:"+password, user.PasswordHash)
	}

	if string(user.Role) != string(RoleCustomer) {
		t.Fatalf("expected user role: %s, got: %s", RoleCustomer, user.Role)
	}
}

func TestService_Register_InvalidEmail(t *testing.T) {
	ctx := context.Background()

	repo := newFakeUserRepository()
	hasher := newFakePasswordHasher()
	service := NewService(repo, hasher)

	invalidEmail := "213dddmail.com"
	password := "valid-password"

	_, err := service.Register(ctx, invalidEmail, password)
	if err == nil {
		t.Fatalf("expected error: %s, but got nil", ErrInvalidEmail)
	}
}

func TestService_Register_EmptyEmail(t *testing.T) {
	ctx := context.Background()

	repo := newFakeUserRepository()
	hasher := newFakePasswordHasher()
	service := NewService(repo, hasher)

	emptyEmail := ""
	password := "valid-password"

	_, err := service.Register(ctx, emptyEmail, password)
	if err == nil {
		t.Fatalf("expected error: %s, but got nil", ErrInvalidEmail)
	}
}

func TestService_Register_ShortPassword(t *testing.T) {
	ctx := context.Background()

	repo := newFakeUserRepository()
	hasher := newFakePasswordHasher()
	service := NewService(repo, hasher)

	emptyEmail := "post@mail.com"
	shortPassword := "pass"

	_, err := service.Register(ctx, emptyEmail, shortPassword)
	if err == nil {
		t.Fatalf("expected error: %s, but got nil", ErrInvalidPassword)
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	ctx := context.Background()

	repo := newFakeUserRepository()
	hasher := newFakePasswordHasher()
	service := NewService(repo, hasher)

	email := uuid.NewString() + "@mail.com"
	password := "valid-password"

	_, err := service.Register(ctx, email, password)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	_, err = service.Register(ctx, email, password)
	if err == nil {
		t.Errorf("expected error: %s, got nil", ErrEmailAlreadyUsed)
	}
}
