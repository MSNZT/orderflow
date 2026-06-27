package httpmw

import (
	"net/http"

	"github.com/MSNZT/orderflow/internal/transport/http/authcontext"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/MSNZT/orderflow/internal/users"
)

func RequireRole(allowedRoles ...users.Role) func(http.Handler) http.Handler {
	allowedMap := make(map[users.Role]struct{}, len(allowedRoles))

	for _, role := range allowedRoles {
		allowedMap[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := authcontext.UserRole(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			if _, hasAccess := allowedMap[role]; !hasAccess {
				response.Error(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
