package middleware

import (
	"net/http"
	"strings"

	"auth-service/internal/service"
)

func AuthenticationRouter(auth *service.AuthService, next http.Handler) http.Handler {
	protected := Authentication(auth)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		protectedPath := strings.HasPrefix(r.URL.Path, "/api/v1/logout") ||
			strings.HasPrefix(r.URL.Path, "/api/v1/validate") ||
			strings.HasPrefix(r.URL.Path, "/api/v1/users") ||
			strings.HasPrefix(r.URL.Path, "/api/v1/user/")
		if protectedPath {
			protected(next).ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
