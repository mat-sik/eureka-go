package registry

type GetHostStatusesResponse struct {
	HostStatuses []HostStatus `json:"host_statuses"`
}
