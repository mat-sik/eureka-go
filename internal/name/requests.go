package name

type registerIPRequest struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type removeIPRequest struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}
