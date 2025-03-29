package name

type RegisterIPRequest struct {
	Name string `json:"name"`
	Host string `json:"host"`
}

type RemoveIPRequest struct {
	Name string `json:"name"`
	Host string `json:"host"`
}
