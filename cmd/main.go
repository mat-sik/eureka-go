package main

import (
	"errors"
	"github.com/mat-sik/eureka-go/internal/name"
	"github.com/mat-sik/eureka-go/internal/props"
	"github.com/mat-sik/eureka-go/internal/server"
	"log/slog"
	"net/http"
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

	serverProps := props.NewServerProperties()

	s := server.NewServer(serverProps, mux)
	if err := s.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
	}
}
