package services

import (
	"context"
	"errors"
)

type OperationRequest struct {
	WalletId      string `json:"walletId"`
	OperationType string `json:"operationType"`
	Amount        int    `json:"amount"`
}

var ErrInvalidAmount = errors.New("amount must be greater than zero")

func PostWallet(operation OperationRequest) (OperationRequest, error) {
	if walletRepo == nil {
		return OperationRequest{}, ErrRepositoryNotInitialized
	}

	if operation.Amount <= 0 {
		return OperationRequest{}, ErrInvalidAmount
	}

	if !validWalletUUID(operation.WalletId) {
		return OperationRequest{}, ErrInvalidUUID
	}

	_, err := walletRepo.ApplyOperation(
		context.Background(),
		operation.WalletId,
		operation.OperationType,
		int64(operation.Amount),
	)
	if err != nil {
		return OperationRequest{}, err
	}

	return operation, nil
}
