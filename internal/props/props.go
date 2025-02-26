package props

import (
	"context"
	"github.com/sethvargo/go-envconfig"
	"log"
	"time"
)

type ServerProperties struct {
	Port         int           `env:"PORT, default=8080"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT, default=5s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT, default=5s"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT, default=5m"`
}

func NewServerProperties() ServerProperties {
	ctx := context.Background()

	var props ServerProperties
	if err := envconfig.Process(ctx, &props); err != nil {
		log.Fatal(err)
	}

	return props
}
