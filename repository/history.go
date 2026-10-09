package repository

import (
	"context"
	"database/sql"
	"fmt"

	"test-go/dbmodel"
)

func insertHistory(
	ctx context.Context,
	tx *sql.Tx,
	walletID int64,
	amount int64,
	operation string,
) error {
	_, err := tx.ExecContext(
		ctx,
		dbmodel.QueryAddHistory,
		walletID,
		amount,
		operation,
	)
	if err != nil {
		return fmt.Errorf("insert operation history: %w", err)
	}

	return nil
}
