package name

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

type RegisterIPHandler struct {
	Store
}

func (h RegisterIPHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var regReq registerIPRequest
	if err := json.NewDecoder(request.Body).Decode(&regReq); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	ip, err := parseIP(regReq.IP)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	h.Store.Add(regReq.Name, ip)
	writer.WriteHeader(http.StatusCreated)
}

type RemoveIpHandler struct {
	Store
}

func (h RemoveIpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var remReq removeIPRequest
	if err := json.NewDecoder(request.Body).Decode(&remReq); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	ip, err := parseIP(remReq.IP)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	h.Store.Remove(remReq.Name, ip)
}

func parseIP(ip string) (net.IP, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid ip Address provided: %s", ip)
	}
	return parsedIP, nil
}

type GetIPHandler struct {
	Store
}

func (h GetIPHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	name := request.PathValue("name")

	ips := h.Store.Get(name)

	resp := getIPResponse{ips}
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}
