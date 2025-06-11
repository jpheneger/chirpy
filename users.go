package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jpheneger/chirpy/internal/auth"
	"github.com/jpheneger/chirpy/internal/database"
)

type reqBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type respBody struct {
	Id           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	AccessToken  string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
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

	success, err := auth.CheckPasswordHash(user.HashedPassword, body.Password)
	if !success || err != nil {
		respondWithError(w, http.StatusUnauthorized, "Login failed", err)
		return
	}

	expiresIn := 1 * time.Hour
	accessToken, err := auth.MakeJWT(user.ID, cfg.signingSecret, expiresIn)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "lgoin failed due to access token error", err)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "lgoin failed due to refresh token error", err)
		return
	}
	result, err := cfg.db.CreateRefreshToken(context.Background(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		ExpiresAt: time.Now().AddDate(0, 0, 60),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to store refresh token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, respBody{
		Id:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		AccessToken:  accessToken,
		RefreshToken: result.Token,
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
		return
	}
	user, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{
		Email:          body.Email,
		HashedPassword: hpw,
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

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type respBody struct {
		Token string `json:"token"`
	}
	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		respondWithError(w, http.StatusUnauthorized, "no authorization header found", errors.New("no authorization header provided"))
		return
	}
	token := strings.Split(authorization, "Bearer ")[1]

	refreshToken, err := cfg.db.GetRefreshToken(context.Background(), token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "token not found", err)
		return
	} else if refreshToken.ExpiresAt.Before(time.Now()) {
		respondWithError(w, http.StatusUnauthorized, "refresh token is expired", errors.New("refresh token is expired"))
	} else if refreshToken.RevokedAt.Valid {
		respondWithError(w, http.StatusUnauthorized, "refresh token is revoked", errors.New("refresh token is revoked"))
	}

	authToken, err := auth.MakeJWT(refreshToken.UserID, cfg.signingSecret, 1*time.Hour)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unable to create auth token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, respBody{
		Token: authToken,
	})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		respondWithError(w, http.StatusUnauthorized, "no authorization header found", errors.New("no authorization header provided"))
		return
	}

	token := strings.Split(authorization, "Bearer ")[1]

	err := cfg.db.RevokeToken(context.Background(), token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unable to revoke refresh token", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
