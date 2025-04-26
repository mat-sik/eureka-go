package registry

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
)

type RegisterHostHandler struct {
	store *Store
}

func (h RegisterHostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var regReq RegisterHostRequest
	if err := json.NewDecoder(request.Body).Decode(&regReq); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	_, _, err := net.SplitHostPort(regReq.Host)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	h.store.addNew(regReq.ServiceID, regReq.Host)
	writer.WriteHeader(http.StatusCreated)
}

type RemoveHostHandler struct {
	store *Store
}

func (h RemoveHostHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var remReq RemoveHostRequest
	if err := json.NewDecoder(request.Body).Decode(&remReq); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	_, _, err := net.SplitHostPort(remReq.Host)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	h.store.Remove(remReq.ServiceID, remReq.Host)
}

type GetHostStatusesHandler struct {
	store *Store
}

func (h GetHostStatusesHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	name := request.PathValue("serviceID")

	hostStatuses := h.store.Get(name)

	resp := GetHostStatusesResponse{hostStatuses}
	respBody, err := json.Marshal(resp)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	if _, err = writer.Write(respBody); err != nil {
		slog.Error("Failed to respond", "response:", resp, "err:", err)
	}
}

func NewHandler(store *Store) http.Handler {
	mux := http.NewServeMux()

	registerIPHandler := &RegisterHostHandler{store: store}
	removeIPHandler := &RemoveHostHandler{store: store}
	getIPHandler := &GetHostStatusesHandler{store: store}

	mux.Handle("POST /service-id/register", registerIPHandler)
	mux.Handle("POST /service-id/remove", removeIPHandler)
	mux.Handle("GET /service-id/{serviceID}", getIPHandler)

	return mux
}
