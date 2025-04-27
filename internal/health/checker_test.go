package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mat-sik/eureka-go/internal/registry"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Test_Checker(t *testing.T) {
	// given
	store := registry.NewStore()

	ctx, cancel := context.WithCancel(context.Background())

	eurekaServer := httptest.NewServer(registry.NewHandler(store))
	defer eurekaServer.Close()

	failAfter := 2
	failAfterServer := httptest.NewServer(newFailAfterHealthCheckHandler(failAfter))
	defer failAfterServer.Close()

	changeStatusServerInitialStatus := registry.Healthy
	changeStatusServerTargetStatus := registry.Down
	changeStatusAfter := 1
	changeStatusServer := httptest.NewServer(newChangeStatusAfterHealthCheckHandler(
		changeStatusServerInitialStatus,
		changeStatusServerTargetStatus,
		changeStatusAfter,
	))
	defer changeStatusServer.Close()

	constantStatus := registry.Healthy
	constantStatusServer := httptest.NewServer(newConstantStatusHealthCheckHandler(constantStatus))
	defer constantStatusServer.Close()

	notifyCh := make(chan struct{})
	mock := &mockStore{
		store: store,

		ctx:                     ctx,
		notifyCh:                notifyCh,
		notifyInvocationCounter: atomic.Int32{},
		hostToStatus:            make(map[string][]registry.Status),
		lock:                    sync.Mutex{},
	}
	checker := NewChecker(
		client,
		mock,
		100*time.Millisecond,
	)

	// when
	serviceIDOne := "foo"
	failAfterServerHost := getHost(t, failAfterServer.URL)
	doRegister(t, eurekaServer.URL, serviceIDOne, failAfterServerHost)

	serviceIDTwo := "bar"
	changeStatusServerHost := getHost(t, changeStatusServer.URL)
	doRegister(t, eurekaServer.URL, serviceIDTwo, changeStatusServerHost)

	constantStatusServerHost := getHost(t, constantStatusServer.URL)
	doRegister(t, eurekaServer.URL, serviceIDTwo, constantStatusServerHost)

	errCh := make(chan error)

	go func() {
		if err := checker.Run(ctx); err != nil {
			errCh <- err
			return
		}
	}()

	runCheckTimes := 12
	notifyCounter := 0

loop:
	for {
		select {
		case err := <-errCh:
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			break loop
		case <-notifyCh:
			notifyCounter++
			if notifyCounter == runCheckTimes {
				notifyCh = nil
				cancel()
			}
		}
	}

	// then
	loggedStatuses := mock.getLoggedStatuses()

	failAfterServerInvocationCount := runCheckTimes / 3
	failAfterServerLoggedStatuses := loggedStatuses[failAfterServerHost][:failAfterServerInvocationCount]
	if len(failAfterServerLoggedStatuses) != failAfterServerInvocationCount {
		t.Fatalf("failAfterServerLoggedStatuses length got: %d want: %d", len(loggedStatuses[failAfterServerHost]), failAfterServerInvocationCount)
	}
	expectedFailAfterServerLoggedStatuses := []registry.Status{registry.Healthy, registry.Healthy, registry.Down, registry.Down}
	if !reflect.DeepEqual(failAfterServerLoggedStatuses, expectedFailAfterServerLoggedStatuses) {
		t.Fatalf("failAfterServerLoggedStatuses got: %v want: %v", failAfterServerLoggedStatuses, expectedFailAfterServerLoggedStatuses)
	}

	changeStatusServerInvocationCount := runCheckTimes / 3
	changeStatusServerLoggedStatuses := loggedStatuses[changeStatusServerHost][:changeStatusServerInvocationCount]
	if len(changeStatusServerLoggedStatuses) != changeStatusServerInvocationCount {
		t.Fatalf("changeStatusServerLoggedStatuses length got: %d want: %d", len(changeStatusServerLoggedStatuses), changeStatusServerInvocationCount)
	}
	expectedChangeStatusServerLoggedStatuses := []registry.Status{registry.Healthy, registry.Healthy, registry.Down, registry.Down}
	if !reflect.DeepEqual(expectedChangeStatusServerLoggedStatuses, expectedFailAfterServerLoggedStatuses) {
		t.Fatalf("changeStatusServerLoggedStatuses got: %v want: %v", changeStatusServerLoggedStatuses, expectedChangeStatusServerLoggedStatuses)
	}

	constantStatusServerInvocationCount := runCheckTimes / 3
	constantStatusServerLoggedStatuses := loggedStatuses[constantStatusServerHost][:constantStatusServerInvocationCount]
	if len(constantStatusServerLoggedStatuses) != constantStatusServerInvocationCount {
		t.Fatalf("constantStatusServerLoggedStatuses length got: %d want: %d", len(constantStatusServerLoggedStatuses), constantStatusServerInvocationCount)
	}
	expectedConstantStatusServerLoggedStatuses := []registry.Status{registry.Healthy, registry.Healthy, registry.Healthy, registry.Healthy}
	if !reflect.DeepEqual(expectedConstantStatusServerLoggedStatuses, expectedConstantStatusServerLoggedStatuses) {
		t.Fatalf("constantStatusServerLoggedStatuses got: %v want: %v", constantStatusServerLoggedStatuses, expectedConstantStatusServerLoggedStatuses)
	}
}

func doRegister(t *testing.T, targetURL string, serviceID string, host string) *http.Response {
	defer buffer.Reset()
	regReq := registry.RegisterHostRequest{ServiceID: serviceID, Host: host}
	if err := encoder.Encode(regReq); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/service-id/register", targetURL), buffer)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func getHost(t *testing.T, rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return parsedURL.Host
}

func newFailAfterHealthCheckHandler(failAfter int) http.Handler {
	return failAfterHandler(failAfter, newTestHealthHandler(registry.Healthy))
}

func failAfterHandler(failAfter int, handler http.Handler) http.Handler {
	counter := atomic.Int64{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		invocation := counter.Add(1)
		if invocation > int64(failAfter) {
			slog.Info("failAfterHandler", "failAfter", failAfter, "invocation", invocation, "status", "down")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		slog.Info("failAfterHandler", "failAfter", failAfter, "invocation", invocation, "status", "healthy")
		handler.ServeHTTP(w, r)
	})
}

func newConstantStatusHealthCheckHandler(status registry.Status) http.Handler {
	handler := newTestHealthHandler(status)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("constantStatusHandler", "status", "healthy")
		handler.ServeHTTP(w, r)
	})
}

func newChangeStatusAfterHealthCheckHandler(initialStatus registry.Status, targetStatus registry.Status, after int) http.Handler {
	counter := atomic.Int64{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := initialStatus
		invocation := counter.Add(1)
		if invocation > int64(after) {
			status = targetStatus
		}
		slog.Info("changeStatusHandler", "after", after, "invocation", invocation, "status", status)
		newTestHealthHandler(status).ServeHTTP(w, r)
	})
}

type TestHealthHandler struct {
	status registry.Status
}

func (th TestHealthHandler) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	slog.Info("health handler", "status", th.status)
	resp := Response{
		Status: th.status,
	}
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

func newTestHealthHandler(status registry.Status) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health", TestHealthHandler{status: status})
	return mux
}

type mockStore struct {
	store *registry.Store

	ctx                     context.Context
	notifyCh                chan<- struct{}
	notifyInvocationCounter atomic.Int32
	hostToStatus            map[string][]registry.Status
	lock                    sync.Mutex
}

func (m *mockStore) GetServiceIDsToHosts() map[string][]string {
	return m.store.GetServiceIDsToHosts()
}

func (m *mockStore) Put(serviceID string, host string, status registry.Status) {
	defer func() {
		select {
		case <-m.ctx.Done():
			slog.Info("DONE")
		case m.notifyCh <- struct{}{}:
			slog.Info("NOTIFIED", "invocation", m.notifyInvocationCounter.Add(1))
		}
	}()

	m.lock.Lock()
	m.hostToStatus[host] = append(m.hostToStatus[host], status)
	m.lock.Unlock()

	m.store.Put(serviceID, host, status)
}

func (m *mockStore) getLoggedStatuses() map[string][]registry.Status {
	m.lock.Lock()
	defer m.lock.Unlock()

	loggedStatuses := make(map[string][]registry.Status, len(m.hostToStatus))
	for k, v := range m.hostToStatus {
		loggedStatuses[k] = v
	}

	return loggedStatuses
}

var (
	buffer  = bytes.NewBuffer(make([]byte, 0, 1024))
	encoder = json.NewEncoder(buffer)
	client  = &http.Client{}
)
