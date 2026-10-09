package services

import (
	"errors"

	"test-go/repository"
)

var walletRepo *repository.Repository

var ErrRepositoryNotInitialized = errors.New("repository not initialized")

func SetRepository(r *repository.Repository) {
	walletRepo = r
}
