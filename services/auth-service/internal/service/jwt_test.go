package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTGenerateAndParse(t *testing.T) {
	s := NewJWTService(JWTConfig{
		Secret:    []byte("01234567890123456789012345678901"),
		Issuer:    "issuer",
		Audience:  "audience",
		AccessTTL: time.Minute,
	})
	id := uuid.New()
	token, _, err := s.Generate(42, "USER", id, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	claims, err := s.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "42" || claims.Role != "USER" || claims.SessionID != id.String() {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}
