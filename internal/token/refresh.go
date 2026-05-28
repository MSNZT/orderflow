package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const refreshTokenSizeBytes = 32

func GenerateRefreshToken() (string, error) {
	b := make([]byte, refreshTokenSizeBytes)
	_, err := rand.Read(b)

	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), err
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
