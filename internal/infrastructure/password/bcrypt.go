package password

import (
	users "github.com/MSNZT/orderflow/internal/app/users"
	"golang.org/x/crypto/bcrypt"
)

const defaultBcryptCost = 12

type BcryptHasher struct {
	cost int
}

var _ users.PasswordHasher = (*BcryptHasher)(nil)

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{
		cost: defaultBcryptCost,
	}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func (h *BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
