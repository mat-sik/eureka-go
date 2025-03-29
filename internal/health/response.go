package health

import "github.com/mat-sik/eureka-go/internal/name"

type Response struct {
	Status name.Status `json:"status"`
}
