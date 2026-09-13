package handler

import (
	"net/http"

	"auth-service/internal/middleware"
	"auth-service/internal/service"
)

func NewMux(auth *service.AuthService, users *service.UserService) http.Handler {
	authHandler := NewAuthHandler(auth)
	userHandler := NewUserHandler(users)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /api/v1/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/v1/validate", authHandler.Validate)

	mux.HandleFunc("GET /api/v1/users", userHandler.List)
	mux.HandleFunc("GET /api/v1/user/{id}", userHandler.Get)
	mux.HandleFunc("PATCH /api/v1/user/{id}", userHandler.Update)
	mux.HandleFunc("DELETE /api/v1/user/{id}", userHandler.Delete)

	return middleware.Recovery(
		middleware.Logging(
			middleware.RequestID(
				middleware.AuthenticationRouter(auth, mux),
			),
		),
	)
}
