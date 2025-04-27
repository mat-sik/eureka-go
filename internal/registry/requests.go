package registry

type RegisterHostRequest struct {
	ServiceID string `json:"service_id"`
	Host      string `json:"host"`
}

type RemoveHostRequest struct {
	ServiceID string `json:"service_id"`
	Host      string `json:"host"`
}
