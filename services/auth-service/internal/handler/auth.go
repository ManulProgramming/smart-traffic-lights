package handler

import (
	"auth-service/internal/model"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"auth-service/internal/httpx"
	"auth-service/internal/middleware"
	"auth-service/internal/service"
	"auth-service/internal/validation"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
		httpx.WriteError(w, r, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "multipart/form-data is required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		httpx.WriteError(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request is too large or invalid")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	email := validation.NormalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")
	if !validation.Name(name) {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_NAME", "name must contain 3-80 letters, digits, or underscores")
		return
	}
	if !validation.Email(email) {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_EMAIL", "invalid email format")
		return
	}
	if !validation.Password(password) {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_PASSWORD", "password must contain 8-128 printable characters")
		return
	}

	var picture []byte
	hasPicture := false
	if file, _, err := r.FormFile("picture"); err == nil {
		defer file.Close()
		picture, err = service.ReadPicture(file)
		if err != nil {
			httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_PICTURE", err.Error())
			return
		}
		hasPicture = true
	}

	result, err := h.auth.Register(r.Context(), service.RegisterRequest{
		Name: name, Email: email, Password: password, Picture: picture, HasPicture: hasPicture,
	})
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			httpx.WriteError(w, r, http.StatusConflict, "USER_ALREADY_EXISTS", "name or email is already registered")
			return
		}
		httpx.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"token": result.Token, "expires_at": result.ExpiresAt, "user": userResponse(result.User),
	})
}

type loginRequest struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req loginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_JSON", "invalid JSON request")
		return
	}

	login := strings.TrimSpace(req.Name)
	if login == "" {
		login = strings.TrimSpace(req.Email)
	}
	if !validation.Login(login) || !validation.Password(req.Password) {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_LOGIN_REQUEST", "invalid login or password format")
		return
	}

	result, err := h.auth.Login(r.Context(), login, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			httpx.WriteError(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
			return
		}
		httpx.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"token": result.Token, "expires_at": result.ExpiresAt})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if err := h.auth.Logout(r.Context(), principal); err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *AuthHandler) Validate(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"valid": false, "user_id": nil})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"valid": true, "user_id": principal.UserID})
}

func userResponse(user *model.User) map[string]any {
	return map[string]any{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"created_at": user.CreatedAt,
		"role":       user.Role,
	}
}
