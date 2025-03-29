package name

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

type RegisterHostHandler struct {
	Store
}

func (h RegisterHostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

	h.Store.addNew(regReq.Name, ip)
	writer.WriteHeader(http.StatusCreated)
}

type RemoveHostHandler struct {
	Store
}

func (h RemoveHostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

type GetHostStatusesHandler struct {
	Store
}

func (h GetHostStatusesHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	name := request.PathValue("name")

	hostStatuses := h.Store.Get(name)

	resp := getHostStatusesResponse{hostStatuses}
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}
