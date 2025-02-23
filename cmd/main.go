package main

import (
	"errors"
	"github.com/mat-sik/eureka-go/internal/name"
	"log/slog"
	"net/http"
	"time"
)

func main() {
	store := name.NewStore()

	registerIPHandler := name.RegisterIPHandler{Store: store}
	removeIPHandler := name.RemoveIpHandler{Store: store}
	getIPHandler := name.GetIPHandler{Store: store}

	mux := http.NewServeMux()

	mux.Handle("POST /name/register", registerIPHandler)
	mux.Handle("POST /name/remove", removeIPHandler)
	mux.Handle("GET /name/{name}", getIPHandler)

	s := http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  2 * time.Minute,
		Handler:      mux,
	}

	if err := s.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
	}
}
