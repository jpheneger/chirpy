package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
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

	authorId := r.URL.Query().Get("author_id")
	sortDir := r.URL.Query().Get("sort")
	if sortDir == "" {
		sortDir = "asc"
	}

	var chirps []database.Chirp
	if authorId == "" {
		allChirps, err := cfg.db.GetAllChirps(context.Background())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't get all chirps from db", err)
			return
		}
		chirps = allChirps
	} else {
		allChirps, err := cfg.db.GetAllChirpsForUser(context.Background(), uuid.MustParse(authorId))
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't get all chirps for user from db", err)
			return
		}
		chirps = allChirps
	}

	responseBody := resp{}
	for _, _chirp := range chirps {
		responseBody = append(responseBody, Chirp{
			Id:        _chirp.ID,
			CreatedAt: _chirp.CreatedAt,
			UpdatedAt: _chirp.UpdatedAt,
			Body:      _chirp.Body,
			UserId:    _chirp.UserID,
		})
	}

	if sortDir == "asc" {
		sort.Slice(responseBody, func(i, j int) bool { return responseBody[i].CreatedAt.Before(responseBody[j].CreatedAt) })
	} else {
		sort.Slice(responseBody, func(i, j int) bool { return responseBody[i].CreatedAt.After(responseBody[j].CreatedAt) })
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
		respondWithError(w, http.StatusUnauthorized, "invalid authorization token provided", err)
		return
	}

	matches, _ := regexp.Match("[0-9a-f]{64}", []byte(token))
	if matches {
		respondWithError(w, http.StatusUnauthorized, "Attempting to use refresh token as access token", err)
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.signingSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid authorization token provided - invalid user", err)
		return
	}

	user, err := cfg.db.GetUserById(context.Background(), userId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch user from db", err)
		return
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
		return
	}

	respondWithJSON(w, http.StatusCreated, Chirp{
		Id:        newChirp.ID,
		CreatedAt: newChirp.CreatedAt,
		UpdatedAt: newChirp.UpdatedAt,
		Body:      newChirp.Body,
		UserId:    newChirp.UserID,
	})
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		respondWithError(w, http.StatusUnauthorized, "no authorization header found", errors.New("no authorization header provided"))
		return
	}
	token := strings.Split(authorization, "Bearer ")[1]
	userId, err := auth.ValidateJWT(token, cfg.signingSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid JWT token", err)
		return
	}

	reqChirpId := r.PathValue("chirpID")
	chirpId := uuid.MustParse(reqChirpId)

	chirp, err := cfg.db.GetChirpById(context.Background(), chirpId)
	if err != nil {
		respondWithError(w, http.StatusForbidden, "Unable to fetch chrip by ID: "+chirpId.String(), err)
		return
	}

	if chirp.UserID != userId {
		respondWithError(w, http.StatusForbidden, "user is not chrip owner", err)
		return
	}

	err = cfg.db.DeleteChripById(context.Background(), database.DeleteChripByIdParams{
		ID:     chirpId,
		UserID: userId,
	})
	if err != nil {
		respondWithError(w, http.StatusForbidden, "Unable to delete chrip by ID: "+chirpId.String(), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
