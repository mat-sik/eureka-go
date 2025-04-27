package registry

type HostStatus struct {
	Host   string `json:"host"`
	Status Status `json:"status"`
}

type Status string

const (
	Unknown Status = "unknown"
	Healthy Status = "healthy"
	Down    Status = "down"
)
