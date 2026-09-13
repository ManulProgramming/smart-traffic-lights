package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string
	JWTSecret   string
	JWTIssuer   string
	JWTAudience string
	JWTTTL      time.Duration
	SessionTTL  time.Duration

	Argon2Memory     uint32
	Argon2Iterations uint32
	Argon2Threads    uint8
	Argon2KeyLength  uint32
	Argon2SaltLength uint32
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:      get("APP_ENV", "development"),
		HTTPAddr:    get("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTIssuer:   get("JWT_ISSUER", "auth-service"),
		JWTAudience: get("JWT_AUDIENCE", "auth-api"),
	}
	var err error
	cfg.JWTTTL, err = duration("JWT_TTL", "15m")
	if err != nil {
		return Config{}, err
	}
	cfg.SessionTTL, err = duration("SESSION_TTL", "168h")
	if err != nil {
		return Config{}, err
	}

	if cfg.DatabaseURL == "" || cfg.JWTSecret == "" {
		return Config{}, errors.New("DATABASE_URL and JWT_SECRET are required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must be at least 32 bytes")
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must be at least 32 bytes")
	}

	cfg.Argon2Memory, err = getUint32("ARGON2_MEMORY", 64*1024)
	if err != nil {
		return Config{}, err
	}
	cfg.Argon2Iterations, err = getUint32("ARGON2_ITERATIONS", 3)
	if err != nil {
		return Config{}, err
	}
	threads, err := getUint32("ARGON2_THREADS", 4)
	if err != nil || threads == 0 || threads > 255 {
		return Config{}, fmt.Errorf("ARGON2_THREADS must be between 1 and 255")
	}
	cfg.Argon2Threads = uint8(threads)

	cfg.Argon2KeyLength, err = getUint32("ARGON2_KEY_LENGTH", 32)
	if err != nil || cfg.Argon2KeyLength < 16 {
		return Config{}, fmt.Errorf("ARGON2_KEY_LENGTH must be at least 16")
	}

	cfg.Argon2SaltLength, err = getUint32("ARGON2_SALT_LENGTH", 16)
	if err != nil || cfg.Argon2SaltLength < 8 {
		return Config{}, fmt.Errorf("ARGON2_SALT_LENGTH must be at least 8")
	}
	return cfg, nil
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func duration(key, fallback string) (time.Duration, error) {
	v := get(key, fallback)
	return time.ParseDuration(v)
}

func getUint32(key string, fallback uint32) (uint32, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.ParseUint(value, 10, 32)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return uint32(n), nil
}

func intEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
