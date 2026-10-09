package handler

import (
	"encoding/json"
	"errors"
	_ "fmt"
	"log"
	"net/http"
	"test-go/repository"
	"test-go/services"
)

func GetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := r.PathValue("WALLET_UUID")

	result, err := services.GetBalance(path)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidUUID):
			writeJSONError(w, http.StatusBadRequest, err.Error())

		case errors.Is(err, repository.ErrWalletNotFound):
			writeJSONError(w, http.StatusNotFound, err.Error())

		default:
			log.Printf("get balance: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("write balance response: %v", err)
	}
}
