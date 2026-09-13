package httpx

import (
	"encoding/json"
	"net/http"
	"time"
)

type ErrorResponse struct {
	Status      int       `json:"status"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Code        string    `json:"code,omitempty"`
	RequestID   string    `json:"request_id,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, description string) {
	WriteJSON(w, status, ErrorResponse{
		Status:      status,
		Description: description,
		Date:        time.Now().UTC(),
		Code:        code,
		RequestID:   RequestID(r.Context()),
	})
}
