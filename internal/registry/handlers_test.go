package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func Test_RegisterHost_SingleServiceTwoHosts(t *testing.T) {
	// given
	registerURL := "/service-id/register"

	serviceID := "one"
	hostOne := "127.0.0.1:8080"
	hostTwo := "127.0.0.1:8081"

	// when
	reqReq := RegisterHostRequest{ServiceID: serviceID, Host: hostOne}
	respOne := doRequest(t, http.MethodPost, registerURL, reqReq)

	reqReq = RegisterHostRequest{ServiceID: serviceID, Host: hostTwo}
	respTwo := doRequest(t, http.MethodPost, registerURL, reqReq)

	// then
	if respOne.Code != http.StatusCreated {
		t.Fatalf("status code: got %v, want %v", respOne.Code, http.StatusCreated)
	}
	if respTwo.Code != http.StatusCreated {
		t.Fatalf("status code: got %v, want %v", respTwo.Code, http.StatusCreated)
	}

	if len(serviceIDToHostStatuses) != 1 {
		t.Fatalf("len(serviceIDToHostStatuses) != %d", len(serviceIDToHostStatuses))
	}
	hosts, ok := serviceIDToHostStatuses[serviceID]
	if !ok {
		t.Fatalf("serviceID: %s not registered", serviceID)
	}

	if len(hosts) != 2 {
		t.Fatalf("len(hosts) = %d, want 2", len(hosts))
	}
	for _, actual := range []string{hostOne, hostTwo} {
		status, ok := hosts[actual]
		if !ok {
			t.Fatalf("hosts[%s] not found", actual)
		}
		if status != Unknown {
			t.Fatalf("hosts[%s] = %s, want Unknown", actual, status)
		}
	}
}

func Test_RegisterHost_TwoServicesTwoHosts(t *testing.T) {
	// clean up
	cleanUp()

	// given
	registerURL := "/service-id/register"

	serviceIDOne := "one"
	hostOne := "127.0.0.1:8080"
	serviceIDTwo := "two"
	hostTwo := "127.0.0.1:8081"

	// when
	reqReq := RegisterHostRequest{ServiceID: serviceIDOne, Host: hostOne}
	respOne := doRequest(t, http.MethodPost, registerURL, reqReq)

	reqReq = RegisterHostRequest{ServiceID: serviceIDTwo, Host: hostTwo}
	respTwo := doRequest(t, http.MethodPost, registerURL, reqReq)

	// then
	if respOne.Code != http.StatusCreated {
		t.Fatalf("status code: got %v, want %v", respOne.Code, http.StatusCreated)
	}
	if respTwo.Code != http.StatusCreated {
		t.Fatalf("status code: got %v, want %v", respTwo.Code, http.StatusCreated)
	}

	if len(serviceIDToHostStatuses) != 2 {
		t.Fatalf("len(serviceIDToHostStatuses) != %d", len(serviceIDToHostStatuses))
	}
	hosts, ok := serviceIDToHostStatuses[serviceIDOne]
	if !ok {
		t.Fatalf("serviceID: %s not registered", serviceIDOne)
	}

	if len(hosts) != 1 {
		t.Fatalf("len(hosts) = %d, want 1", len(hosts))
	}
	status, ok := hosts[hostOne]
	if !ok {
		t.Fatalf("hosts[%s] not found", hostOne)
	}
	if status != Unknown {
		t.Fatalf("hosts[%s] = %s, want Unknown", hostOne, status)
	}

	hosts, ok = serviceIDToHostStatuses[serviceIDTwo]
	if !ok {
		t.Fatalf("serviceID: %s not registered", serviceIDTwo)
	}
	status, ok = hosts[hostTwo]
	if !ok {
		t.Fatalf("hosts[%s] not found", hostTwo)
	}
	if status != Unknown {
		t.Fatalf("hosts[%s] = %s, want Unknown", hostTwo, status)
	}
}

func Test_RegisterHost_InvalidHost(t *testing.T) {
	// given
	registerURL := "/service-id/register"

	serviceID := "one"
	hostOne := "wrong host"

	// when
	reqReq := RegisterHostRequest{ServiceID: serviceID, Host: hostOne}
	resp := doRequest(t, http.MethodPost, registerURL, reqReq)

	// then
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status code %d, want %d", resp.Code, http.StatusBadRequest)
	}
}

func Test_RegisterHost_BrokenRequest(t *testing.T) {
	// given
	registerURL := "/service-id/register"

	// when
	resp := doBrokenRequest(http.MethodPost, registerURL)

	// then
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status code %d, want %d", resp.Code, http.StatusInternalServerError)
	}
}

func Test_RemoveHost_RemoveNotAll(t *testing.T) {
	// given
	registerURL := "/service-id/register"
	removeURL := "/service-id/remove"

	serviceID := "one"
	hostOne := "127.0.0.1:8080"
	hostTwo := "127.0.0.1:8081"

	// when
	reqReq := RegisterHostRequest{ServiceID: serviceID, Host: hostOne}
	respOne := doRequest(t, http.MethodPost, registerURL, reqReq)

	reqReq = RegisterHostRequest{ServiceID: serviceID, Host: hostTwo}
	respTwo := doRequest(t, http.MethodPost, registerURL, reqReq)

	remReq := RemoveHostRequest{ServiceID: serviceID, Host: hostOne}
	respThree := doRequest(t, http.MethodPost, removeURL, remReq)

	// then
	if respOne.Code != http.StatusCreated {
		t.Fatalf("status code: got %v, want %v", respOne.Code, http.StatusCreated)
	}
	if respTwo.Code != http.StatusCreated {
		t.Fatalf("status code: got %v, want %v", respTwo.Code, http.StatusCreated)
	}
	if respThree.Code != http.StatusOK {
		t.Fatalf("status code: got %v, want %v", respThree.Code, http.StatusOK)
	}

	hosts, ok := serviceIDToHostStatuses[serviceID]
	if !ok {
		t.Fatalf("serviceID: %s not registered", serviceID)
	}

	if len(hosts) != 1 {
		t.Fatalf("len(hosts) = %d, want 1", len(hosts))
	}
	status, ok := hosts[hostOne]
	if ok {
		t.Fatalf("hosts[%s] = %v, want nil", hostOne, status)
	}

	status, ok = hosts[hostTwo]
	if !ok {
		t.Fatalf("hosts[%s] not found", hostTwo)
	}
	if status != Unknown {
		t.Fatalf("hosts[%s] = %s, want Unknown", hostOne, status)
	}
}

func Test_RemoveHost_RemoveAll(t *testing.T) {
	// clean up
	cleanUp()

	// given
	registerURL := "/service-id/register"
	removeURL := "/service-id/remove"

	serviceID := "one"
	hostOne := "127.0.0.1:8080"

	// when
	reqReq := RegisterHostRequest{ServiceID: serviceID, Host: hostOne}
	respOne := doRequest(t, http.MethodPost, registerURL, reqReq)

	remReq := RemoveHostRequest{ServiceID: serviceID, Host: hostOne}
	respTwo := doRequest(t, http.MethodPost, removeURL, remReq)

	// then
	if respOne.Code != http.StatusCreated {
		t.Fatalf("status code: got %v, want %v", respOne.Code, http.StatusCreated)
	}
	if respTwo.Code != http.StatusOK {
		t.Fatalf("status code: got %v, want %v", respTwo.Code, http.StatusOK)
	}

	_, ok := serviceIDToHostStatuses[serviceID]
	if ok {
		t.Fatalf("serviceID: %s is registered", serviceID)
	}
}

func Test_RemoveHost_RemoveNotExisting(t *testing.T) {
	// clean up
	cleanUp()

	// given
	removeURL := "/service-id/remove"

	serviceID := "not-exist"
	hostOne := "127.0.0.1:8080"

	// when
	req := RemoveHostRequest{ServiceID: serviceID, Host: hostOne}
	resp := doRequest(t, http.MethodPost, removeURL, req)

	// then
	if resp.Code != http.StatusOK {
		t.Fatalf("status code: got %v, want %v", resp.Code, http.StatusOK)
	}

	_, ok := serviceIDToHostStatuses[serviceID]
	if ok {
		t.Fatalf("serviceID: %s is registered", serviceID)
	}
}

func Test_GetHostStatuses_EmptyResponse(t *testing.T) {
	// given
	getURL := "/service-id/not-exist"

	// when
	resp := doNoBodyRequest(http.MethodGet, getURL)

	// then
	if resp.Code != http.StatusOK {
		t.Fatalf("status code: got %v, want %v", resp.Code, http.StatusOK)
	}

	var got GetHostStatusesResponse
	err := json.Unmarshal(resp.Body.Bytes(), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := GetHostStatusesResponse{
		HostStatuses: []HostStatus{},
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func Test_GetHostStatuses_NormalResponse(t *testing.T) {
	// given
	registerURL := "/service-id/register"

	serviceID := "normalResponse"
	getURL := fmt.Sprintf("/service-id/%s", serviceID)

	hostOne := "127.0.0.1:8080"
	hostTwo := "127.0.0.1:8081"

	// when
	reqReq := RegisterHostRequest{ServiceID: serviceID, Host: hostOne}
	respOne := doRequest(t, http.MethodPost, registerURL, reqReq)

	reqReq = RegisterHostRequest{ServiceID: serviceID, Host: hostTwo}
	respTwo := doRequest(t, http.MethodPost, registerURL, reqReq)

	respThree := doNoBodyRequest(http.MethodGet, getURL)

	// then
	if respOne.Code != http.StatusCreated {
		t.Fatalf("status code: got %v, want %v", respOne.Code, http.StatusCreated)
	}
	if respTwo.Code != http.StatusCreated {
		t.Fatalf("status code: got %v, want %v", respTwo.Code, http.StatusCreated)
	}
	if respThree.Code != http.StatusOK {
		t.Fatalf("status code: got %v, want %v", respThree.Code, http.StatusOK)
	}

	want := GetHostStatusesResponse{
		HostStatuses: []HostStatus{
			{
				Host:   hostOne,
				Status: Unknown,
			},
			{
				Host:   hostTwo,
				Status: Unknown,
			},
		},
	}
	var got GetHostStatusesResponse
	err := json.Unmarshal(respThree.Body.Bytes(), &got)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func doNoBodyRequest(method string, target string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, nil))

	return recorder
}

func doRequest(t *testing.T, method string, target string, v any) *httptest.ResponseRecorder {
	defer buffer.Reset()
	if err := encoder.Encode(v); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, buffer))

	return recorder
}

func doBrokenRequest(method string, target string) *httptest.ResponseRecorder {
	defer buffer.Reset()
	buffer.WriteString("wrong")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, buffer))

	return recorder
}

func cleanUp() {
	for key := range serviceIDToHostStatuses {
		delete(serviceIDToHostStatuses, key)
	}
}

var (
	serviceIDToHostStatuses = make(map[string]map[string]Status)
	handler                 = NewHandler(NewStoreFrom(serviceIDToHostStatuses))
	buffer                  = bytes.NewBuffer(make([]byte, 0, 1024))
	encoder                 = json.NewEncoder(buffer)
)
