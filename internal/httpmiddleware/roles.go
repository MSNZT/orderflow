package httpmiddleware

import (
	"fmt"
	"net/http"

	"github.com/MSNZT/orderflow/internal/authcontext"
	"github.com/MSNZT/orderflow/internal/httpresponse"
	"github.com/MSNZT/orderflow/internal/users"
)

func RequireRole(allowedRoles ...users.Role) func(http.Handler) http.Handler {
	allowedMap := make(map[users.Role]struct{}, len(allowedRoles))

	for _, role := range allowedRoles {
		allowedMap[role] = struct{}{}
	}
	fmt.Println(allowedMap)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := authcontext.UserRole(r.Context())
			if !ok {
				httpresponse.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			if _, hasAccess := allowedMap[role]; !hasAccess {
				httpresponse.Error(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
