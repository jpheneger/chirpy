// mycode.go
package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jpheneger/chirpy/internal/auth"

	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T) {
	result, err := auth.MakeJWT(uuid.New(), "test", 5*time.Minute)
	if err != nil {
		t.Errorf("MakeJWT failed with error: %v", err)
	}
	expected := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

	if !strings.HasPrefix(result, expected) {
		t.Errorf("MakeJWT = %s; want %s", result, expected)
	}
}

func TestValidateJWT(t *testing.T) {
	result, err := auth.MakeJWT(uuid.New(), "test", 5*time.Minute)
	if err != nil {
		t.Errorf("MakeJWT failed with error: %v", err)
	}
	userId, err := auth.ValidateJWT(result, "test")
	if err != nil {
		t.Errorf("unable to validate token: %s - err:%v", result, err)
	}

	if userId == uuid.Nil {
		t.Errorf("userID from token is not a valid UUID")
	}
}

func TestGetBearerToken(t *testing.T) {
	expected := "THISISMYTOKEN"
	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", expected))

	token, err := auth.GetBearerToken(headers)
	if err != nil {
		t.Errorf("unable to get token from headers: %v", err)
	} else if token == "" {
		t.Errorf("empty token retrieved, expected %v", expected)
	} else if token != expected {
		t.Errorf("expected %s, got %s", expected, token)
	}
}
