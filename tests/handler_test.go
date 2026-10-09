package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func performRequest(
	api http.Handler,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(
		method,
		path,
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	return rec
}

func TestHTTPGetBalance(t *testing.T) {
	conn, repo := setupRepository(t)

	walletUUID, _ := createTestWallet(t, conn, 77)
	api := newAPI(t, repo)

	t.Run("success", func(t *testing.T) {
		rec := performRequest(
			api,
			http.MethodGet,
			"/api/v1/wallets/"+walletUUID,
			"",
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
		}

		var balance float64
		if err := json.Unmarshal(rec.Body.Bytes(), &balance); err != nil {
			t.Fatalf("decode balance: %v", err)
		}

		if balance != 77 {
			t.Errorf("balance = %v, want 77", balance)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		rec := performRequest(
			api,
			http.MethodGet,
			"/api/v1/wallets/not-a-uuid",
			"",
		)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("wallet not found", func(t *testing.T) {
		rec := performRequest(
			api,
			http.MethodGet,
			"/api/v1/wallets/"+newTestUUID(t),
			"",
		)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestHTTPPostInvalidJSON(t *testing.T) {
	_, repo := setupRepository(t)
	api := newAPI(t, repo)

	rec := performRequest(
		api,
		http.MethodPost,
		"/api/v1/wallet",
		`{"walletId":`,
	)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHTTPPostDeposit(t *testing.T) {
	conn, repo := setupRepository(t)

	walletUUID, walletID := createTestWallet(t, conn, 100)
	api := newAPI(t, repo)

	body := fmt.Sprintf(
		`{"walletId":%q,"operationType":"DEPOSIT","amount":25}`,
		walletUUID,
	)

	rec := performRequest(
		api,
		http.MethodPost,
		"/api/v1/wallet",
		body,
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
	}

	assertBalance(t, repo, walletUUID, 125)

	if got := historyCount(t, conn, walletID); got != 1 {
		t.Errorf("history count = %d, want 1", got)
	}
}

func TestHTTPPostErrorStatuses(t *testing.T) {
	conn, repo := setupRepository(t)

	walletUUID, walletID := createTestWallet(t, conn, 10)
	missingUUID := newTestUUID(t)

	api := newAPI(t, repo)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name: "zero amount",
			body: fmt.Sprintf(
				`{"walletId":%q,"operationType":"DEPOSIT","amount":0}`,
				walletUUID,
			),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "unknown operation",
			body: fmt.Sprintf(
				`{"walletId":%q,"operationType":"TRANSFER","amount":1}`,
				walletUUID,
			),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "insufficient funds",
			body: fmt.Sprintf(
				`{"walletId":%q,"operationType":"WITHDRAW","amount":11}`,
				walletUUID,
			),
			wantStatus: http.StatusConflict,
		},
		{
			name: "missing wallet",
			body: fmt.Sprintf(
				`{"walletId":%q,"operationType":"DEPOSIT","amount":1}`,
				missingUUID,
			),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := performRequest(
				api,
				http.MethodPost,
				"/api/v1/wallet",
				tc.body,
			)

			if rec.Code != tc.wantStatus {
				t.Errorf(
					"status = %d, want %d; body: %s",
					rec.Code,
					tc.wantStatus,
					rec.Body,
				)
			}
		})
	}

	assertBalance(t, repo, walletUUID, 10)

	if got := historyCount(t, conn, walletID); got != 0 {
		t.Errorf("history count = %d, want 0", got)
	}
}

func TestHTTPMethodNotAllowed(t *testing.T) {
	_, repo := setupRepository(t)
	api := newAPI(t, repo)

	tests := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/api/v1/wallet",
		},
		{
			method: http.MethodPost,
			path:   "/api/v1/wallets/" + newTestUUID(t),
		},
	}

	for _, tc := range tests {
		rec := performRequest(api, tc.method, tc.path, "")

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf(
				"%s %s: status = %d, want 405",
				tc.method,
				tc.path,
				rec.Code,
			)
		}
	}
}
