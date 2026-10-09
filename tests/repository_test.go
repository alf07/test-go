package test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"test-go/repository"
)

func TestGetBalance(t *testing.T) {
	conn, repo := setupRepository(t)

	walletUUID, _ := createTestWallet(t, conn, 123)

	got, err := repo.GetBalance(context.Background(), walletUUID)
	if err != nil {
		t.Fatalf("GetBalance returned error: %v", err)
	}

	if got != 123 {
		t.Errorf("balance = %v, want 123", got)
	}

	_, err = repo.GetBalance(context.Background(), newTestUUID(t))
	if !errors.Is(err, repository.ErrWalletNotFound) {
		t.Errorf("error = %v, want ErrWalletNotFound", err)
	}
}

func TestDepositUpdatesBalanceAndHistory(t *testing.T) {
	conn, repo := setupRepository(t)

	walletUUID, walletID := createTestWallet(t, conn, 100)

	balance, err := repo.ApplyOperation(
		context.Background(),
		walletUUID,
		"DEPOSIT",
		25,
	)
	if err != nil {
		t.Fatalf("deposit returned error: %v", err)
	}

	if balance != 125 {
		t.Errorf("returned balance = %v, want 125", balance)
	}

	assertBalance(t, repo, walletUUID, 125)

	if got := historyCount(t, conn, walletID); got != 1 {
		t.Fatalf("history count = %d, want 1", got)
	}

	var amount float64
	var operation, status string

	err = conn.QueryRow(
		`SELECT amount, operation, status
		 FROM history
		 WHERE wallet_id = $1`,
		walletID,
	).Scan(&amount, &operation, &status)

	if err != nil {
		t.Fatalf("read history: %v", err)
	}

	if amount != 25 || operation != "DEPOSIT" || status != "COMPLETED" {
		t.Errorf(
			"unexpected history: amount=%v operation=%s status=%s",
			amount, operation, status,
		)
	}
}

func TestWithdrawUpdatesBalanceAndHistory(t *testing.T) {
	conn, repo := setupRepository(t)

	walletUUID, walletID := createTestWallet(t, conn, 100)

	balance, err := repo.ApplyOperation(
		context.Background(),
		walletUUID,
		"WITHDRAW",
		35,
	)
	if err != nil {
		t.Fatalf("withdraw returned error: %v", err)
	}

	if balance != 65 {
		t.Errorf("returned balance = %v, want 65", balance)
	}

	assertBalance(t, repo, walletUUID, 65)

	if got := historyCount(t, conn, walletID); got != 1 {
		t.Errorf("history count = %d, want 1", got)
	}
}

func TestInsufficientFundsDoesNotChangeBalance(t *testing.T) {
	conn, repo := setupRepository(t)

	walletUUID, walletID := createTestWallet(t, conn, 10)

	_, err := repo.ApplyOperation(
		context.Background(),
		walletUUID,
		"WITHDRAW",
		11,
	)

	if !errors.Is(err, repository.ErrInsufficientFunds) {
		t.Fatalf("error = %v, want ErrInsufficientFunds", err)
	}

	assertBalance(t, repo, walletUUID, 10)

	if got := historyCount(t, conn, walletID); got != 0 {
		t.Errorf("history count = %d, want 0", got)
	}
}

func TestInvalidOperationDoesNotChangeBalance(t *testing.T) {
	conn, repo := setupRepository(t)

	walletUUID, walletID := createTestWallet(t, conn, 100)

	_, err := repo.ApplyOperation(
		context.Background(),
		walletUUID,
		"TRANSFER",
		10,
	)

	if !errors.Is(err, repository.ErrInvalidOperation) {
		t.Fatalf("error = %v, want ErrInvalidOperation", err)
	}

	assertBalance(t, repo, walletUUID, 100)

	if got := historyCount(t, conn, walletID); got != 0 {
		t.Errorf("history count = %d, want 0", got)
	}
}

func TestOperationWithMissingWallet(t *testing.T) {
	_, repo := setupRepository(t)

	_, err := repo.ApplyOperation(
		context.Background(),
		newTestUUID(t),
		"DEPOSIT",
		10,
	)

	if !errors.Is(err, repository.ErrWalletNotFound) {
		t.Errorf("error = %v, want ErrWalletNotFound", err)
	}
}

func TestConcurrentDeposits(t *testing.T) {
	conn, repo := setupRepository(t)

	walletUUID, walletID := createTestWallet(t, conn, 0)

	const operations = 1000
	const amount int64 = 1

	ctx, cancel := context.WithTimeout(
		context.Background(),
		60*time.Second,
	)
	defer cancel()

	start := make(chan struct{})
	errs := make(chan error, operations)

	var wg sync.WaitGroup

	for i := 0; i < operations; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			_, err := repo.ApplyOperation(
				ctx,
				walletUUID,
				"DEPOSIT",
				amount,
			)

			errs <- err
		}()
	}

	close(start)

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("concurrent deposit failed: %v", err)
		}
	}

	assertBalance(t, repo, walletUUID, operations)

	if got := historyCount(t, conn, walletID); got != operations {
		t.Errorf("history count = %d, want %d", got, operations)
	}
}

func TestConcurrentWithdrawalsNeverMakeBalanceNegative(t *testing.T) {
	conn, repo := setupRepository(t)

	const initialBalance int64 = 1000
	const amount int64 = 20
	const operations = 100

	walletUUID, walletID := createTestWallet(t, conn, initialBalance)

	start := make(chan struct{})
	errs := make(chan error, operations)

	var wg sync.WaitGroup

	for i := 0; i < operations; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			_, err := repo.ApplyOperation(
				context.Background(),
				walletUUID,
				"WITHDRAW",
				amount,
			)

			errs <- err
		}()
	}

	close(start)

	wg.Wait()
	close(errs)

	var successful, insufficient int

	for err := range errs {
		switch {
		case err == nil:
			successful++

		case errors.Is(err, repository.ErrInsufficientFunds):
			insufficient++

		default:
			t.Errorf("unexpected withdrawal error: %v", err)
		}
	}

	if successful != 50 {
		t.Errorf("successful withdrawals = %d, want 50", successful)
	}

	if insufficient != 50 {
		t.Errorf("insufficient-funds responses = %d, want 50", insufficient)
	}

	assertBalance(t, repo, walletUUID, 0)

	if got := historyCount(t, conn, walletID); got != 50 {
		t.Errorf("history count = %d, want 50", got)
	}
}
