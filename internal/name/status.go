package name

type HostStatus struct {
	IP     string `json:"ip"`
	Status `json:"status"`
}

type Status string

const (
	Unknown Status = "unknown"
	Healthy Status = "healthy"
	Down    Status = "down"
)
