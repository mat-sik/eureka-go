package name

import (
	"encoding/json"
	"net"
	"net/http"
)

type RegisterHostHandler struct {
	Store
}

func (h RegisterHostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var regReq RegisterIPRequest
	if err := json.NewDecoder(request.Body).Decode(&regReq); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	_, _, err := net.SplitHostPort(regReq.Host)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	h.Store.addNew(regReq.Name, regReq.Host)
	writer.WriteHeader(http.StatusCreated)
}

type RemoveHostHandler struct {
	Store
}

func (h RemoveHostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var remReq RemoveIPRequest
	if err := json.NewDecoder(request.Body).Decode(&remReq); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	_, _, err := net.SplitHostPort(remReq.Host)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	h.Store.Remove(remReq.Name, remReq.Host)
}

type GetHostStatusesHandler struct {
	Store
}

func (h GetHostStatusesHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	name := request.PathValue("name")

	hostStatuses := h.Store.Get(name)

	resp := GetHostStatusesResponse{hostStatuses}
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}
