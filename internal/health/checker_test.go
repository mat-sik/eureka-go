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

	serverOneFailAfter := 2
	serverOneWaitCh := make(chan struct{})
	serverOne := httptest.NewServer(newFailAfterHealthCheckHandler(ctx, serverOneWaitCh, serverOneFailAfter))
	defer serverOne.Close()
	hostOne := getHost(t, serverOne.URL)

	serverTwoChangeStatusAfter := 1
	serverTwoWaitCh := make(chan struct{})
	serverTwoInitialStatus := registry.Healthy
	serverTwoTargetStatus := registry.Down
	serverTwo := httptest.NewServer(newChangeStatusAfterHealthCheckHandler(
		ctx,
		serverTwoWaitCh,
		serverTwoInitialStatus,
		serverTwoTargetStatus,
		serverTwoChangeStatusAfter,
	))
	defer serverTwo.Close()
	hostTwo := getHost(t, serverTwo.URL)

	serverThreeWaitCh := make(chan struct{})
	serverThreeStatus := registry.Healthy
	serverThree := httptest.NewServer(newConstantStatusHealthCheckHandler(ctx, serverThreeWaitCh, serverThreeStatus))
	defer serverThree.Close()
	hostThree := getHost(t, serverThree.URL)

	serviceIDOne := "foo"
	serviceIDTwo := "bar"

	// when
	doRegister(t, eurekaServer.URL, serviceIDOne, hostOne)
	doRegister(t, eurekaServer.URL, serviceIDTwo, hostTwo)
	doRegister(t, eurekaServer.URL, serviceIDTwo, hostThree)

	errCh := make(chan error)

	go func() {
		if err := checker.Run(ctx); err != nil {
			errCh <- err
			return
		}
	}()

	serverOneNotifyCounter := 0
	serverTwoNotifyCounter := 0
	serverThreeNotifyCounter := 0

	i := 0

	for {
		select {
		case err := <-errCh:
			t.Fatal(err)
		case <-serverOneWaitCh:
			i++
			serverOneNotifyCounter++
			if serverOneNotifyCounter == runCheckTimes {
				serverOneWaitCh = nil
			}
		case <-serverTwoWaitCh:
			i++
			serverTwoNotifyCounter++
			if serverTwoNotifyCounter == runCheckTimes {
				serverTwoWaitCh = nil
			}
		case <-serverThreeWaitCh:
			i++
			serverThreeNotifyCounter++
			if serverThreeNotifyCounter == runCheckTimes {
				serverThreeWaitCh = nil
			}
		}
		if serverOneWaitCh == nil && serverTwoWaitCh == nil && serverThreeWaitCh == nil {
			slog.Info("stop", "i", i)
			break
		}
	}

	time.Sleep(time.Second)
	cancel()

	// then
	loggedData := mock.getLoggedData()

	serverOneData := loggedData[hostOne][:runCheckTimes]
	if len(serverOneData) != runCheckTimes {
		t.Fatalf("failAfter got: %d want: %d", len(loggedData[hostOne]), runCheckTimes)
	}
	expectedServerOneData := []registry.Status{registry.Healthy, registry.Healthy, registry.Down, registry.Down}
	if !reflect.DeepEqual(serverOneData, expectedServerOneData) {
		t.Fatalf("failAfter got: %v want: %v", serverOneData, expectedServerOneData)
	}

	serverTwoData := loggedData[hostTwo][:runCheckTimes]
	if len(serverTwoData) != runCheckTimes {
		t.Fatalf("changeStatusAfter got: %d want: %d", len(serverTwoData), runCheckTimes)
	}
	expectedServerTwoData := []registry.Status{registry.Healthy, registry.Healthy, registry.Down, registry.Down}
	if !reflect.DeepEqual(expectedServerTwoData, expectedServerOneData) {
		t.Fatalf("changeStatusAfter got: %v want: %v", serverTwoData, expectedServerTwoData)
	}

	serverThreeData := loggedData[hostThree][:runCheckTimes]
	if len(serverThreeData) != runCheckTimes {
		t.Fatalf("constant got: %d want: %d", len(serverThreeData), runCheckTimes)
	}
	expectedServerThreeData := []registry.Status{registry.Healthy, registry.Healthy, registry.Healthy, registry.Healthy}
	if !reflect.DeepEqual(expectedServerThreeData, expectedServerThreeData) {
		t.Fatalf("constant got: %v want: %v", serverThreeData, expectedServerThreeData)
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

var waitChCounter = atomic.Int32{}

func waitChanHandler(ctx context.Context, waitCh chan<- struct{}, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			select {
			case <-ctx.Done():
				slog.Info("DONE")
			case waitCh <- struct{}{}:
				slog.Info("NOTIFIED", "i", waitChCounter.Add(1))
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

func (m *mockStore) getLoggedData() map[string][]registry.Status {
	m.lock.Lock()
	defer m.lock.Unlock()

	clone := make(map[string][]registry.Status, len(m.hostToStatus))
	for k, v := range m.hostToStatus {
		clone[k] = v
	}

	return clone
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
