package server

import (
	"fmt"
	"github.com/mat-sik/eureka-go/internal/props"
	"net/http"
)

func NewServer(serverProps props.ServerProperties, handler http.Handler) http.Server {
	return http.Server{
		Addr:         fmt.Sprintf(":%d", serverProps.Port),
		ReadTimeout:  serverProps.ReadTimeout,
		WriteTimeout: serverProps.WriteTimeout,
		IdleTimeout:  serverProps.IdleTimeout,
		Handler:      handler,
	}
}
