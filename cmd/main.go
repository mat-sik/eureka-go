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
	s := server.NewServer(props.NewServerProperties(), registry.NewHandler())
	if err := s.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
	}
}
