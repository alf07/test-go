package main

import (
	"fmt"
	"net/http"

	"test-go/db"
	"test-go/handler"
	"test-go/repository"
	"test-go/services"
)

func main() {
	conn, err := db.Connect()
	if err != nil {
		fmt.Println("Ошибка подключения к БД:", err)
		return
	}
	defer conn.Close()

	services.SetRepository(repository.New(conn))

	http.HandleFunc("/api/v1/wallets/{WALLET_UUID}", handler.GetBalance)
	http.HandleFunc("/api/v1/wallet", handler.PostWallet)

	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Println("Ошибка сервера:", err)
	}
}
