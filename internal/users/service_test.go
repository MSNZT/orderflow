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

	if user.Role != RoleCustomer {
		t.Fatalf("expected user role: %s, got: %s", RoleCustomer, user.Role)
	}

	savedUser, err := service.repo.GetByEmail(ctx, email)
	if err != nil {
		t.Errorf("get saved user: %v", err)
	}

	if savedUser.ID != user.ID {
		t.Errorf("expected saved user id: %s, got: %s", user.ID, savedUser.ID)
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
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected error: %s, but got: %v", ErrInvalidEmail, err)
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
	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got: %v", err)
	}
}

func TestService_Register_ShortPassword(t *testing.T) {
	ctx := context.Background()

	repo := newFakeUserRepository()
	hasher := newFakePasswordHasher()
	service := NewService(repo, hasher)

	email := "post@mail.com"
	shortPassword := "invalid"

	_, err := service.Register(ctx, email, shortPassword)
	if !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword, got: %v", err)
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
	if !errors.Is(err, ErrEmailAlreadyUsed) {
		t.Errorf("expected ErrEmailAlreadyUsed, got %v", err)
	}
}

func TestService_Login_Success(t *testing.T) {
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

	user, err := service.Login(ctx, email, password)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if user.Email != email {
		t.Fatalf("expected email: %s, got: %s", email, user.Email)
	}

	if user.Role != RoleCustomer {
		t.Fatalf("expected role: %s, got: %s", RoleCustomer, user.Role)
	}

	if user.ID == uuid.Nil {
		t.Fatalf("expected user id, got nil")
	}

}

func TestService_Login_UserNotFound(t *testing.T) {
	ctx := context.Background()

	repo := newFakeUserRepository()
	hasher := newFakePasswordHasher()
	service := NewService(repo, hasher)

	email := uuid.NewString() + "@mail.com"
	password := "valid-password"

	_, err := service.Login(ctx, email, password)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
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

	wrongPassword := "wrong-password"

	_, err = service.Login(ctx, email, wrongPassword)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestService_Login_InvalidEmail(t *testing.T) {
	ctx := context.Background()

	repo := newFakeUserRepository()
	hasher := newFakePasswordHasher()
	service := NewService(repo, hasher)

	invalidEmail := "invalid-email."
	password := "valid-password"

	_, err := service.Login(ctx, invalidEmail, password)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}
