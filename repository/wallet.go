package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"test-go/dbmodel"
)

var ErrWalletNotFound = errors.New("wallet not found")
var ErrInsufficientFunds = errors.New("insufficient funds")
var ErrInvalidOperation = errors.New("invalid operation")

type Repository struct {
	db *sql.DB
}

func New(conn *sql.DB) *Repository {
	return &Repository{db: conn}
}

// GetBalance получение баланса
func (r *Repository) GetBalance(
	ctx context.Context,
	walletUUID string,
) (float64, error) {
	var id int64
	var uuid string
	var balance float64

	err := r.db.QueryRowContext(
		ctx,
		dbmodel.QueryGetWallet,
		walletUUID,
	).Scan(&id, &uuid, &balance)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrWalletNotFound
	}

	if err != nil {
		return 0, fmt.Errorf("get wallet balance: %w", err)
	}

	return balance, nil
}

func (r *Repository) ApplyOperation(
	ctx context.Context,
	walletUUID string,
	operation string,
	amount int64,
) (float64, error) {
	var query string

	switch operation {
	case "DEPOSIT":
		query = dbmodel.QueryDepositBalance
	case "WITHDRAW":
		query = dbmodel.QueryWithdrawBalance
	default:
		return 0, ErrInvalidOperation
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var walletID int64
	var returnedUUID string
	var balance float64

	err = tx.QueryRowContext(
		ctx,
		query,
		amount,
		walletUUID,
	).Scan(&walletID, &returnedUUID, &balance)

	if errors.Is(err, sql.ErrNoRows) {
		var id int64

		checkErr := tx.QueryRowContext(
			ctx,
			dbmodel.QueryGetWalletID,
			walletUUID,
		).Scan(&id)

		if errors.Is(checkErr, sql.ErrNoRows) {
			return 0, ErrWalletNotFound
		}
		if checkErr != nil {
			return 0, fmt.Errorf("check wallet: %w", checkErr)
		}

		if operation == "WITHDRAW" {
			return 0, ErrInsufficientFunds
		}

		return 0, fmt.Errorf("wallet update returned no rows")
	}

	if err != nil {
		return 0, fmt.Errorf("update wallet balance: %w", err)
	}

	if err := insertHistory(
		ctx,
		tx,
		walletID,
		amount,
		operation,
	); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit wallet operation: %w", err)
	}

	return balance, nil
}
