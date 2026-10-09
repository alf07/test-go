package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"test-go/repository"
	"test-go/services"
)

type OperationRequest struct {
	WalletId      string `json:"walletId"`
	OperationType string `json:"operationType"`
	Amount        int    `json:"amount"`
}

func PostWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var operation OperationRequest
	if err := decoder.Decode(&operation); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	result, err := services.PostWallet(
		services.OperationRequest(operation),
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidUUID),
			errors.Is(err, services.ErrInvalidAmount),
			errors.Is(err, repository.ErrInvalidOperation):
			writeJSONError(w, http.StatusBadRequest, err.Error())

		case errors.Is(err, repository.ErrWalletNotFound):
			writeJSONError(w, http.StatusNotFound, err.Error())

		case errors.Is(err, repository.ErrInsufficientFunds):
			writeJSONError(w, http.StatusConflict, err.Error())

		default:
			log.Printf("post wallet: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("write operation response: %v", err)
	}
}
