package health

import (
	"bytes"
	"context"
	"encoding/json"
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
	ctx, cancel := context.WithCancel(context.Background())

	runCheckTimes := 4

	eurekaServer := httptest.NewServer(registry.NewHandler(store))
	defer eurekaServer.Close()

	failAfterServerWaitCh := make(chan struct{})
	failAfter := 2
	failAfterServer := httptest.NewServer(newFailAfterHealthCheckHandler(ctx, failAfterServerWaitCh, failAfter))
	defer failAfterServer.Close()
	failAfterServerHost := getHost(t, failAfterServer.URL)

	changeStatusServerWaitCh := make(chan struct{})
	changeStatusServerInitialStatus := registry.Healthy
	changeStatusServerTargetStatus := registry.Down
	changeStatusAfter := 1
	changeStatusServer := httptest.NewServer(newChangeStatusAfterHealthCheckHandler(
		ctx,
		changeStatusServerWaitCh,
		changeStatusServerInitialStatus,
		changeStatusServerTargetStatus,
		changeStatusAfter,
	))
	defer changeStatusServer.Close()
	changeStatusServerHost := getHost(t, changeStatusServer.URL)

	constantStatusServerWaitCh := make(chan struct{})
	constantStatus := registry.Healthy
	constantStatusServer := httptest.NewServer(newConstantStatusHealthCheckHandler(ctx, constantStatusServerWaitCh, constantStatus))
	defer constantStatusServer.Close()
	constantStatusServerHost := getHost(t, constantStatusServer.URL)

	serviceIDOne := "foo"
	serviceIDTwo := "bar"

	// when
	doRegister(t, eurekaServer.URL, serviceIDOne, failAfterServerHost)
	doRegister(t, eurekaServer.URL, serviceIDTwo, changeStatusServerHost)
	doRegister(t, eurekaServer.URL, serviceIDTwo, constantStatusServerHost)

	errCh := make(chan error)

	go func() {
		if err := checker.Run(ctx); err != nil {
			errCh <- err
			return
		}
	}()

	failAfterServerNotifyCounter := 0
	changeStatusServerNotifyCounter := 0
	constantServerNotifyCounter := 0

	allNotifyCounter := 0

	for {
		select {
		case err := <-errCh:
			t.Fatal(err)
		case <-failAfterServerWaitCh:
			allNotifyCounter++
			failAfterServerNotifyCounter++
			if failAfterServerNotifyCounter == runCheckTimes {
				failAfterServerWaitCh = nil
			}
		case <-changeStatusServerWaitCh:
			allNotifyCounter++
			changeStatusServerNotifyCounter++
			if changeStatusServerNotifyCounter == runCheckTimes {
				changeStatusServerWaitCh = nil
			}
		case <-constantStatusServerWaitCh:
			allNotifyCounter++
			constantServerNotifyCounter++
			if constantServerNotifyCounter == runCheckTimes {
				constantStatusServerWaitCh = nil
			}
		}
		if failAfterServerWaitCh == nil && changeStatusServerWaitCh == nil && constantStatusServerWaitCh == nil {
			slog.Info("stop", "allNotifyCounter", allNotifyCounter)
			break
		}
	}

	time.Sleep(time.Second)
	cancel()

	// then
	loggedStatuses := mock.getLoggedStatuses()

	failAfterServerLoggedStatuses := loggedStatuses[failAfterServerHost][:runCheckTimes]
	if len(failAfterServerLoggedStatuses) != runCheckTimes {
		t.Fatalf("failAfterServerLoggedStatuses length got: %d want: %d", len(loggedStatuses[failAfterServerHost]), runCheckTimes)
	}
	expectedFailAfterServerLoggedStatuses := []registry.Status{registry.Healthy, registry.Healthy, registry.Down, registry.Down}
	if !reflect.DeepEqual(failAfterServerLoggedStatuses, expectedFailAfterServerLoggedStatuses) {
		t.Fatalf("failAfterServerLoggedStatuses got: %v want: %v", failAfterServerLoggedStatuses, expectedFailAfterServerLoggedStatuses)
	}

	changeStatusServerLoggedStatuses := loggedStatuses[changeStatusServerHost][:runCheckTimes]
	if len(changeStatusServerLoggedStatuses) != runCheckTimes {
		t.Fatalf("changeStatusServerLoggedStatuses length got: %d want: %d", len(changeStatusServerLoggedStatuses), runCheckTimes)
	}
	expectedChangeStatusServerLoggedStatuses := []registry.Status{registry.Healthy, registry.Healthy, registry.Down, registry.Down}
	if !reflect.DeepEqual(expectedChangeStatusServerLoggedStatuses, expectedFailAfterServerLoggedStatuses) {
		t.Fatalf("changeStatusServerLoggedStatuses got: %v want: %v", changeStatusServerLoggedStatuses, expectedChangeStatusServerLoggedStatuses)
	}

	constantStatusServerLoggedStatuses := loggedStatuses[constantStatusServerHost][:runCheckTimes]
	if len(constantStatusServerLoggedStatuses) != runCheckTimes {
		t.Fatalf("constantStatusServerLoggedStatuses length got: %d want: %d", len(constantStatusServerLoggedStatuses), runCheckTimes)
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

func newFailAfterHealthCheckHandler(ctx context.Context, waitCh chan<- struct{}, failAfter int) http.Handler {
	return waitChanHandler(ctx, waitCh, failAfterHandler(failAfter, newTestHealthHandler(registry.Healthy)))
}

func newConstantStatusHealthCheckHandler(ctx context.Context, waitCh chan<- struct{}, status registry.Status) http.Handler {
	handler := newTestHealthHandler(status)
	return waitChanHandler(ctx, waitCh, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("constantStatusHandler", "status", "healthy")
		handler.ServeHTTP(w, r)
	}))
}

func newChangeStatusAfterHealthCheckHandler(
	ctx context.Context,
	waitCh chan<- struct{},
	initialStatus registry.Status,
	targetStatus registry.Status,
	after int,
) http.Handler {
	return waitChanHandler(ctx, waitCh, changeStatusAfterHandler(initialStatus, targetStatus, after))
}

func waitChanHandler(ctx context.Context, waitCh chan<- struct{}, handler http.Handler) http.Handler {
	waitChCounter := atomic.Int32{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			select {
			case <-ctx.Done():
				slog.Info("DONE")
			case waitCh <- struct{}{}:
				slog.Info("NOTIFIED", "invocation", waitChCounter.Add(1))
			}
		}()
		handler.ServeHTTP(w, r)
	})
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

func changeStatusAfterHandler(initialStatus registry.Status, targetStatus registry.Status, after int) http.Handler {
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

	hostToStatus map[string][]registry.Status
	lock         sync.Mutex
}

func (m *mockStore) GetServiceIDsToHosts() map[string][]string {
	return m.store.GetServiceIDsToHosts()
}

func (m *mockStore) Put(serviceID string, host string, status registry.Status) {
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
	store   = registry.NewStore()
	mock    = &mockStore{
		store:        store,
		hostToStatus: make(map[string][]registry.Status),
		lock:         sync.Mutex{},
	}
	checker = NewChecker(
		client,
		mock,
		100*time.Millisecond,
	)
)
