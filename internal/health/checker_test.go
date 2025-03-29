package health

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/mat-sik/eureka-go/internal/registry"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func Test_foo(t *testing.T) {
	client := &http.Client{}
	store := registry.NewStore()

	checker := Checker{
		client: client,
		store:  store,
	}

	testEurekaServer := httptest.NewServer(registry.RegisterHostHandler{Store: store})
	defer testEurekaServer.Close()

	testClientServer := httptest.NewServer(TestHealthHandler{})
	defer testClientServer.Close()

	parsedURL, _ := url.Parse(testClientServer.URL)
	body, _ := json.Marshal(registry.RegisterIPRequest{ServiceID: "foo", Host: parsedURL.Host})
	req, err := http.NewRequest(http.MethodPost, testEurekaServer.URL, bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	err = checker.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

type TestHealthHandler struct{}

func (th TestHealthHandler) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	resp := Response{
		Status: registry.Healthy,
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}
