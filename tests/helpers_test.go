package test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"test-go/handler"
	"test-go/repository"
	"test-go/services"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	if os.Getenv("DATABASE_URL") == "" {
		if err := godotenv.Load("../config.env"); err != nil {
			t.Fatalf("load config.env: %v", err)
		}
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL is not set")
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		t.Fatalf("ping database: %v", err)
	}

	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	return conn
}

func setupRepository(t *testing.T) (*sql.DB, *repository.Repository) {
	t.Helper()

	conn := openTestDB(t)
	return conn, repository.New(conn)
}

func newTestUUID(t *testing.T) string {
	t.Helper()

	var b [16]byte

	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("generate UUID: %v", err)
	}

	// UUID version 4 и variant RFC 4122.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	)
}

func createTestWallet(
	t *testing.T,
	conn *sql.DB,
	initialBalance int64,
) (string, int64) {
	t.Helper()

	walletUUID := newTestUUID(t)

	var walletID int64

	err := conn.QueryRow(
		`INSERT INTO wallet (wallet_uuid, balance)
		 VALUES ($1, $2)
		 RETURNING id`,
		walletUUID,
		initialBalance,
	).Scan(&walletID)

	if err != nil {
		t.Fatalf("create test wallet: %v", err)
	}

	// Удаляем только данные, созданные этим тестом.
	// Cleanup выполняется и при падении теста.
	t.Cleanup(func() {
		if _, err := conn.Exec(
			`DELETE FROM history WHERE wallet_id = $1`,
			walletID,
		); err != nil {
			t.Errorf("delete test history: %v", err)
		}

		if _, err := conn.Exec(
			`DELETE FROM wallet WHERE id = $1`,
			walletID,
		); err != nil {
			t.Errorf("delete test wallet: %v", err)
		}
	})

	return walletUUID, walletID
}

func historyCount(t *testing.T, conn *sql.DB, walletID int64) int64 {
	t.Helper()

	var count int64

	err := conn.QueryRow(
		`SELECT COUNT(*) FROM history WHERE wallet_id = $1`,
		walletID,
	).Scan(&count)

	if err != nil {
		t.Fatalf("count wallet history: %v", err)
	}

	return count
}

func assertBalance(
	t *testing.T,
	repo *repository.Repository,
	walletUUID string,
	want float64,
) {
	t.Helper()

	got, err := repo.GetBalance(context.Background(), walletUUID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	if got != want {
		t.Errorf("balance = %v, want %v", got, want)
	}
}

func newAPI(t *testing.T, repo *repository.Repository) http.Handler {
	t.Helper()

	services.SetRepository(repo)

	t.Cleanup(func() {
		services.SetRepository(nil)
	})

	mux := http.NewServeMux()
	mux.HandleFunc(
		"/api/v1/wallets/{WALLET_UUID}",
		handler.GetBalance,
	)
	mux.HandleFunc(
		"/api/v1/wallet",
		handler.PostWallet,
	)

	return mux
}
