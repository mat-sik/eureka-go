package main

import (
	"errors"
	"github.com/mat-sik/eureka-go/internal/props"
	"github.com/mat-sik/eureka-go/internal/registry"
	"github.com/mat-sik/eureka-go/internal/server"
	"log/slog"
	"net/http"
)

func main() {
	store := registry.NewStore()
	handler := registry.NewHandler(store)
	s := server.NewServer(props.NewServerProperties(), handler)
	if err := s.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
	}
}
