package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jpheneger/chirpy/internal/auth"
	"github.com/jpheneger/chirpy/internal/database"
)

type reqBody struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	ExpiresIn int    `json:"expires_in_seconds"`
}
type respBody struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	body := reqBody{}
	err := decoder.Decode(&body)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	user, err := cfg.db.GetUserByEmail(context.Background(), body.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to authenticate", err)
		return
	}

	success, err := auth.CheckPasswordHash(user.HashedPassword.String, body.Password)
	if !success || err != nil {
		respondWithError(w, http.StatusUnauthorized, "Login failed", err)
		return
	}

	expiresIn := 1 * time.Hour
	if body.ExpiresIn != 0 && body.ExpiresIn <= 3600 {
		expiresIn = time.Duration(body.ExpiresIn) * time.Second
	}

	token, err := auth.MakeJWT(user.ID, cfg.signingSecret, expiresIn)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "lgoin failed due to token error", err)
	}

	respondWithJSON(w, http.StatusOK, respBody{
		Id:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     token,
	})
}

func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	body := reqBody{}
	err := decoder.Decode(&body)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	hpw, err := auth.HashPassword(body.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to hash password", err)
	}
	user, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{
		Email:          body.Email,
		HashedPassword: sql.NullString{String: hpw, Valid: true},
	})
	if err != nil {
		log.Fatalf("unable to create user: %v", err)
	}

	respondWithJSON(w, http.StatusCreated, respBody{
		Id:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	})
}
