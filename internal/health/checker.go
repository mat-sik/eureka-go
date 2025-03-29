package health

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mat-sik/eureka-go/internal/name"
	"net/http"
)

type Checker struct {
	client *http.Client
	store  name.Store
}

func (c *Checker) Run(ctx context.Context) error {
	for {
		if ctx.Done() != nil {
			return nil
		}
		for hostAlias, hosts := range c.store.GetNamesToIps() {
			for _, host := range hosts {
				if err := c.checkJob(ctx, hostAlias, host); err != nil {
					return err
				}
			}
		}
	}
}

func (c *Checker) checkJob(ctx context.Context, hostName string, host string) error {
	status, err := c.check(ctx, host)
	if err != nil {
		return err
	}
	c.store.Put(hostName, host, status)
	return nil
}

func (c *Checker) check(ctx context.Context, host string) (name.Status, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getHealthAddr(host), nil)
	if err != nil {
		return name.Unknown, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return name.Unknown, err
	}
	defer resp.Body.Close()

	var healthResp Response
	if err = json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return name.Unknown, err
	}

	return healthResp.Status, nil
}

func getHealthAddr(host string) string {
	return fmt.Sprintf("http://%s/health", host)
}
