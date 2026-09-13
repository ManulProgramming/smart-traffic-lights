package middleware

import (
	"context"
	"net/http"
	"strings"

	"auth-service/internal/httpx"
	"auth-service/internal/service"
)

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal *service.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (*service.Principal, bool) {
	value, ok := ctx.Value(principalKey{}).(*service.Principal)
	return value, ok && value != nil
}

func Authentication(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				httpx.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "missing Authorization header")
				return
			}

			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				httpx.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid Authorization header")
				return
			}

			principal, err := auth.Validate(r.Context(), parts[1])
			if err != nil {
				httpx.WriteError(w, r, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired token")
				return
			}

			ctx := WithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
