package middleware

import (
	"log/slog"
	"net/http"

	"auth-service/internal/httpx"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("panic recovered", "panic", recovered, "request_id", httpx.RequestID(r.Context()))
				httpx.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
