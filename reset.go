package main

import (
	"context"
	"log"
	"net/http"
)

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)

	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("reset only available in 'dev' platform"))
	}
	err := cfg.db.DeleteUsers(context.Background())
	if err != nil {
		log.Fatalf("Unable to delete users: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}
