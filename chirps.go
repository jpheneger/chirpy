package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jpheneger/chirpy/internal/auth"
	"github.com/jpheneger/chirpy/internal/database"
)

const SUB_STRING = "****"

type Chirp struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) handlerAllChirps(w http.ResponseWriter, r *http.Request) {
	type resp []Chirp

	allChirps, err := cfg.db.GetAllChirps(context.Background())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get chirps from db", err)
	}

	responseBody := resp{}
	for _, _chirp := range allChirps {
		responseBody = append(responseBody, Chirp{
			Id:        _chirp.ID,
			CreatedAt: _chirp.CreatedAt,
			UpdatedAt: _chirp.UpdatedAt,
			Body:      _chirp.Body,
			UserId:    _chirp.UserID,
		})
	}

	respondWithJSON(w, http.StatusOK, responseBody)
}

func (cfg *apiConfig) handlerChirpById(w http.ResponseWriter, r *http.Request) {
	reqChirpId := r.PathValue("chirpID")
	chirpId := uuid.MustParse(reqChirpId)
	chirp, err := cfg.db.GetChirpById(context.Background(), chirpId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Unable to get chrip by ID: "+chirpId.String(), err)
		return
	}

	respondWithJSON(w, http.StatusOK, Chirp{
		Id:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserId:    chirp.UserID,
	})
}

func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	reqBody := req{}
	err := decoder.Decode(&reqBody)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldnt decode request body", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid authorization token provided - expected: 'Authorization: Bearer <TOKEN>'", err)
	}
	userId, err := auth.ValidateJWT(token, cfg.signingSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid authorization token provided - invalid user", err)
	}

	user, err := cfg.db.GetUserById(context.Background(), userId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch user from db", err)
	}

	const maxChirpLength = 140
	if len(reqBody.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	regex := regexp.MustCompile("(?i)(kerfuffle)|(sharbert)|(fornax)")
	newText := regex.ReplaceAllString(reqBody.Body, SUB_STRING)
	matchCount := regex.FindAllString(reqBody.Body, -1)
	fmt.Printf("Match count: %d, replacement text: %s\n", len(matchCount), newText)

	newChirp, err := cfg.db.CreateChirp(context.Background(), database.CreateChirpParams{
		Body:   newText,
		UserID: user.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create chirp", err)
	}

	respondWithJSON(w, http.StatusCreated, Chirp{
		Id:        newChirp.ID,
		CreatedAt: newChirp.CreatedAt,
		UpdatedAt: newChirp.UpdatedAt,
		Body:      newChirp.Body,
		UserId:    newChirp.UserID,
	})
}
