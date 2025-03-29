package health

import (
	"github.com/mat-sik/eureka-go/internal/registry"
)

type Response struct {
	Status registry.Status `json:"status"`
}
