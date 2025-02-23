package name

import "net"

type getIPResponse struct {
	IPs []net.IP `json:"ips"`
}
