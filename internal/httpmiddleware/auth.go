package httpmiddleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MSNZT/orderflow/internal/authcontext"
	"github.com/MSNZT/orderflow/internal/httpresponse"
	"github.com/MSNZT/orderflow/internal/token"
	"github.com/MSNZT/orderflow/internal/users"
	"github.com/google/uuid"
)

type TokenParser interface {
	ParseAccessToken(accessToken string) (*token.Claims, error)
}

func Auth(tokenParser TokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accessToken, err := extractAuthorizationToken(r)
			if err != nil {
				httpresponse.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			claims, err := tokenParser.ParseAccessToken(accessToken)
			if err != nil {
				httpresponse.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				httpresponse.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			role := users.Role(claims.Role)
			ctx := authcontext.WithUser(r.Context(), userID, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractAuthorizationToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return "", fmt.Errorf("authorization token is missing")
	}

	token, found := strings.CutPrefix(authHeader, "Bearer ")
	if !found {
		return "", fmt.Errorf("invalid auth header format")
	}

	return token, nil
}
