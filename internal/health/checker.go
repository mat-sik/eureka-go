package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mat-sik/eureka-go/internal/registry"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Checker struct {
	client        *http.Client
	ticker        *time.Ticker
	statusUpdater statusUpdater
}

type statusUpdater interface {
	serviceIDsToHostsGetter
	statusPutter
}

func (c Checker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.ticker.C:
			if err := c.checkAll(ctx); err != nil {
				return err
			}
		}
	}
}

type serviceIDsToHostsGetter interface {
	GetServiceIDsToHosts() map[string][]string
}

func (c Checker) checkAll(ctx context.Context) error {
	serviceIDsToHosts := c.statusUpdater.GetServiceIDsToHosts()
	wg := &sync.WaitGroup{}
	errCh := make(chan error, jobCount(serviceIDsToHosts))
	defer close(errCh)
	for serviceID, hosts := range serviceIDsToHosts {
		for _, host := range hosts {
			wg.Add(1)
			go c.checkJob(ctx, wg, errCh, serviceID, host)
		}
	}
	wg.Wait()

	return collectErrs(errCh)
}

type statusPutter interface {
	Put(serviceID string, host string, status registry.Status)
}

func (c Checker) checkJob(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error, serviceID string, host string) {
	slog.Info("running checker job", "serviceID", serviceID, "host", host)
	defer wg.Done()
	status, err := c.check(ctx, host)
	if err != nil {
		errCh <- err
		return
	}
	slog.Info("checker job finished", "serviceID", serviceID, "host", host, "status", status)
	c.statusUpdater.Put(serviceID, host, status)
}

func (c Checker) check(ctx context.Context, host string) (registry.Status, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getHealthAddr(host), nil)
	if err != nil {
		return registry.Unknown, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return registry.Unknown, err
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "err", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return registry.Down, nil
	}

	var healthResp Response
	if err = json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return registry.Unknown, err
	}

	return healthResp.Status, nil
}

func jobCount(serviceIDsToHosts map[string][]string) int {
	count := 0
	for _, hosts := range serviceIDsToHosts {
		count += len(hosts)
	}
	return count
}

func collectErrs(errCh <-chan error) error {
	errs := make([]error, 0, len(errCh))
	for i := 0; i < len(errCh); i++ {
		errs = append(errs, <-errCh)
	}
	return errors.Join(errs...)
}

func getHealthAddr(host string) string {
	return fmt.Sprintf("http://%s/health", host)
}

func NewChecker(client *http.Client, statusUpdater statusUpdater, duration time.Duration) Checker {
	return Checker{
		client:        client,
		statusUpdater: statusUpdater,
		ticker:        time.NewTicker(duration),
	}
}
