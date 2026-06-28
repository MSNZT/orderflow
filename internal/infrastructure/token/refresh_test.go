package token

import "testing"

func TestToken_GenerateRefreshToken_NotEmpty(t *testing.T) {
	token, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	if token == "" {
		t.Fatalf("token must not be empty: %s", token)
	}
}

func TestToken_GenerateRefreshToken_Unique(t *testing.T) {
	token, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	token2, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	if token == token2 {
		t.Fatalf("token must be unique: %s, %s", token, token2)
	}
}

func TestToken_HashRefreshToken_Same(t *testing.T) {
	token := "refresh-token"
	hash := hashRefreshToken(token)
	hash2 := hashRefreshToken(token)

	if hash != hash2 {
		t.Fatalf("refresh token hash must be the same: %s, %s", hash, hash2)
	}
}

func TestToken_HashRefreshToken_Distinct(t *testing.T) {
	token := "refresh-token"
	token2 := "refresh-token-2"

	hash := hashRefreshToken(token)
	hash2 := hashRefreshToken(token2)

	if hash == hash2 {
		t.Fatalf("refresh token hash must be unique: %s, %s", hash, hash2)
	}
}

func TestToken_HashRefreshToken_NotEqualRaw(t *testing.T) {
	token := "refresh-token"
	hash := hashRefreshToken(token)

	if token == hash {
		t.Fatalf("hash must not be equal to raw token: %s", token)
	}
}

func TestToken_HashRefreshToken_SHA256HexLength(t *testing.T) {
	token := "refresh-token"
	hash := hashRefreshToken(token)

	if len(hash) != 64 {
		t.Fatalf("expected hash length to be 64, got: %d", len(hash))
	}
}
