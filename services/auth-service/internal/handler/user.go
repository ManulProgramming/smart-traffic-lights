package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"auth-service/internal/httpx"
	"auth-service/internal/middleware"
	"auth-service/internal/service"
	"auth-service/internal/validation"
)

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if value := r.URL.Query().Get("limit"); value != "" {
		if n, err := strconv.Atoi(value); err == nil && n > 0 && n <= 100 {
			limit = n
		} else {
			httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_LIMIT", "limit must be between 1 and 100")
			return
		}
	}
	if value := r.URL.Query().Get("offset"); value != "" {
		if n, err := strconv.Atoi(value); err == nil && n >= 0 {
			offset = n
		} else {
			httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_OFFSET", "offset must be zero or greater")
			return
		}
	}

	users, total, err := h.users.List(r.Context(), limit, offset)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	out := make([]map[string]any, 0, len(users))
	for _, user := range users {
		out = append(out, userResponse(user))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"users": out, "total": total, "limit": limit, "offset": offset})
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil || id <= 0 {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid user id")
		return
	}

	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if principal.UserID != id && principal.Role != "ADMIN" {
		httpx.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}

	user, err := h.users.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			httpx.WriteError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
			return
		}
		httpx.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	fmt.Println(user)
	httpx.WriteJSON(w, http.StatusOK, userResponse(user))
}

type updateRequest struct {
	Name            *string `json:"name"`
	Email           *string `json:"email"`
	NewPassword     *string `json:"new_password"`
	CurrentPassword string  `json:"current_password"`
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil || id <= 0 {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid user id")
		return
	}
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok || principal.UserID != id {
		httpx.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "you may only update your own account")
		return
	}

	var req updateRequest
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data;") {
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			httpx.WriteError(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request is too large or invalid")
			return
		}
		if value := r.FormValue("name"); value != "" {
			req.Name = &value
		}
		if value := r.FormValue("email"); value != "" {
			req.Email = &value
		}
		if value := r.FormValue("new_password"); value != "" {
			req.NewPassword = &value
		}
		req.CurrentPassword = r.FormValue("current_password")
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_JSON", "invalid JSON request")
			return
		}
	}

	if !validation.Password(req.CurrentPassword) {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_CURRENT_PASSWORD", "current_password is required")
		return
	}
	if req.Name != nil && !validation.Name(strings.TrimSpace(*req.Name)) {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_NAME", "invalid name")
		return
	}
	if req.Email != nil && !validation.Email(validation.NormalizeEmail(*req.Email)) {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_EMAIL", "invalid email format")
		return
	}
	if req.NewPassword != nil && !validation.Password(*req.NewPassword) {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_PASSWORD", "new_password must contain 8-128 printable characters")
		return
	}

	update := service.UpdateUserRequest{CurrentPassword: req.CurrentPassword, Name: req.Name, Email: req.Email, NewPassword: req.NewPassword}
	if file, _, err := r.FormFile("picture"); err == nil {
		defer file.Close()
		update.Picture, err = service.ReadPicture(file)
		if err != nil {
			httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_PICTURE", err.Error())
			return
		}
		update.HasPicture = true
	}

	user, err := h.users.Update(r.Context(), id, update)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			httpx.WriteError(w, r, http.StatusUnauthorized, "INVALID_CURRENT_PASSWORD", "current password is incorrect")
		case errors.Is(err, service.ErrUserAlreadyExists):
			httpx.WriteError(w, r, http.StatusConflict, "USER_ALREADY_EXISTS", "name or email is already registered")
		case errors.Is(err, service.ErrUserNotFound):
			httpx.WriteError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		default:
			httpx.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "user": userResponse(user)})
}

type deleteRequest struct {
	Password string `json:"password"`
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil || id <= 0 {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid user id")
		return
	}
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok || principal.UserID != id {
		httpx.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "you may only delete your own account")
		return
	}

	var req deleteRequest
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || !validation.Password(req.Password) {
		httpx.WriteError(w, r, http.StatusBadRequest, "INVALID_PASSWORD", "password is required")
		return
	}

	if err := h.users.Delete(r.Context(), id, req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			httpx.WriteError(w, r, http.StatusUnauthorized, "INVALID_PASSWORD", "password is incorrect")
		case errors.Is(err, service.ErrUserNotFound):
			httpx.WriteError(w, r, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		default:
			httpx.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
