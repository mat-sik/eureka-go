package health

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mat-sik/eureka-go/internal/registry"
	"net/http"
)

type Checker struct {
	client *http.Client
	store  registry.Store
}

func (c *Checker) Run(ctx context.Context) error {
	for {
		if ctx.Done() != nil {
			return nil
		}
		for serviceID, hosts := range c.store.GetServiceIDsToHosts() {
			for _, host := range hosts {
				if err := c.checkJob(ctx, serviceID, host); err != nil {
					return err
				}
			}
		}
	}
}

func (c *Checker) checkJob(ctx context.Context, serviceID string, host string) error {
	status, err := c.check(ctx, host)
	if err != nil {
		return err
	}
	c.store.Put(serviceID, host, status)
	return nil
}

func (c *Checker) check(ctx context.Context, host string) (registry.Status, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getHealthAddr(host), nil)
	if err != nil {
		return registry.Unknown, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return registry.Unknown, err
	}
	defer resp.Body.Close()

	var healthResp Response
	if err = json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return registry.Unknown, err
	}

	return healthResp.Status, nil
}

func getHealthAddr(host string) string {
	return fmt.Sprintf("http://%s/health", host)
}
