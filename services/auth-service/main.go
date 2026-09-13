package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auth-service/internal/config"
	"auth-service/internal/database"
	"auth-service/internal/handler"
	"auth-service/internal/repository"
	"auth-service/internal/service"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found; using system environment variables")
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepository(pool)
	sessionRepo := repository.NewSessionRepository(pool)
	roleRepo := repository.NewRoleRepository(pool)
	pictureRepo := repository.NewPictureRepository()

	passwordService := service.NewPasswordService(service.Argon2Config{
		Memory:      cfg.Argon2Memory,
		Iterations:  cfg.Argon2Iterations,
		Parallelism: cfg.Argon2Threads,
		KeyLength:   cfg.Argon2KeyLength,
		SaltLength:  cfg.Argon2SaltLength,
	})

	jwtService := service.NewJWTService(service.JWTConfig{
		Secret:    []byte(cfg.JWTSecret),
		Issuer:    cfg.JWTIssuer,
		Audience:  cfg.JWTAudience,
		AccessTTL: cfg.JWTTTL,
	})

	txManager := service.NewTxManager(pool)

	authService := service.NewAuthService(
		userRepo,
		sessionRepo,
		roleRepo,
		pictureRepo,
		passwordService,
		jwtService,
		cfg.SessionTTL,
		txManager,
	)

	userService := service.NewUserService(
		userRepo,
		sessionRepo,
		pictureRepo,
		passwordService,
		txManager,
	)

	mux := handler.NewMux(authService, userService)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP server started", "addr", cfg.HTTPAddr)
		serverErr <- server.ListenAndServe()
	}()

	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
