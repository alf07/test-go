package services

import "context"

func GetBalance(path string) (float64, error) {
	if !validWalletUUID(path) {
		return 0, ErrInvalidUUID
	}

	if walletRepo == nil {
		return 0, ErrRepositoryNotInitialized
	}

	return walletRepo.GetBalance(context.Background(), path)
}
