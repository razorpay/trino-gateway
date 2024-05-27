package monitor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/trinodb/trino-go-client/trino"
)

type ITrinoClient interface {
	IsClusterUp(ctx *context.Context) (bool, error)
	RunQuery(ctx *context.Context, query string) (*sql.Rows, error)
	Teardown(ctx *context.Context) error
}

type TrinoClient struct {
	db   *sql.DB
	url  url.URL
	user string
	pass string
}

func NewTrinoClient(ctx *context.Context, url url.URL, user string) *TrinoClient {
	return &TrinoClient{
		url:  url,
		user: user,
	}
}

func (t *TrinoClient) httpClient(ctx *context.Context) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			MaxIdleConns:          3,
			IdleConnTimeout:       10 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 2 * time.Second,
		},
	}
}

func (t *TrinoClient) initTrinoDbClient(ctx *context.Context) error {
	provider.Logger(*ctx).Debug("Initializing Trinodb Client")
	httpClient := t.httpClient(ctx)

	const CustomClientKey = "custom_client"

	if err := trino.RegisterCustomClient(CustomClientKey, httpClient); err != nil {
		return err
	}

	dsn := t.generateDsn(ctx, CustomClientKey)

	client, err := sql.Open("trino", dsn)
	if err != nil {
		// boot.Logger(ctx).Error("could not create data-lake client")

		return err
	}

	t.db = client

	return nil
}

func (t *TrinoClient) generateDsn(ctx *context.Context, key string) string {
	return fmt.Sprintf("%s://%s:%s@%s?catalog=%s&schema=%s&custom_client=%s",
		t.url.Scheme,
		t.user,
		t.pass,
		t.url.Host,
		"system",
		"runtime",
		key)
}

// Checks whether cluster is up purely via healthcheck / clusterInfo Apis
func (t *TrinoClient) IsClusterUp(ctx *context.Context) (bool, error) {
	client := t.httpClient(ctx)

	path := fmt.Sprintf("%s://%s/v1/info", t.url.Scheme, t.url.Host)
	provider.Logger(*ctx).Debugw(
		"Fetching ClusterInfo from trino cluster api",
		map[string]interface{}{"path": path})
	req, err := http.NewRequestWithContext(*ctx, "GET", path, nil)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"error creating http request object",
			map[string]interface{}{"trinoHost": t.url.Host})
		return false, err
	}
	req.Header.Add("X-Trino-User", t.user)
	resp, err := client.Do(req)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"error sending http request to Trino api",
			map[string]interface{}{"trinoHost": t.url.Host})
		return false, err
	}

	if !(200 <= resp.StatusCode || resp.StatusCode <= 300) {
		err := errors.New("trino api returned non HTTP2xx status")
		provider.Logger(*ctx).WithError(err).Errorw(
			"error sending http request to Trino api",
			map[string]interface{}{"trinoHost": t.url.Host})
		return false, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"error parsing http response from Trino api",
			map[string]interface{}{"trinoHost": t.url.Host})
		return false, err
	}

	type Info struct {
		Starting    bool   `json:"starting"`
		Coordinator bool   `json:"coordinator"`
		Environment string `json:"environment"`
		Uptime      string `json:"uptime"`
		NodeVersion struct {
			Version string `json:"version"`
		} `json:"nodeVersion"`
	}

	info := &Info{Starting: true, Coordinator: false}
	json.Unmarshal(body, &info)

	if info.NodeVersion.Version == "" {
		err := errors.New("trino api returned invalid response")
		provider.Logger(*ctx).WithError(err).Errorw(
			"Malformed response returned from Trino Cluster Api",
			map[string]interface{}{"path": path, "body": info})
		return false, err
	}
	if !info.Coordinator || info.Starting {
		provider.Logger(*ctx).Debugw(
			"Trino cluster not ready",
			map[string]interface{}{"path": path, "body": info})
		return false, nil
	}

	return true, nil
}

func (t *TrinoClient) IsClusterHealthy(ctx *context.Context) (bool, error) {
	// Run healthcheck sql query
	q := boot.Config.Monitor.HealthCheckSql
	rows, err := t.RunQuery(ctx, q)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"error executing healthcheck trino query",
			map[string]interface{}{"query": q})
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		// The result of the healthcheck query is irrelevant
		// iterate over it fully so Server can mark query as Finished instead of
		// User Cancelled or User Abandoned.
	}
	if err = rows.Err(); err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"healthcheck query failed",
			map[string]interface{}{"query": q})
		return false, err
	}
	provider.Logger(*ctx).Debugw(
		"healthcheck query ran succesfully",
		map[string]interface{}{"query": q})
	return true, nil
}

func (t *TrinoClient) RunQuery(ctx *context.Context, q string) (*sql.Rows, error) {
	if t.db == nil {
		// log
		err := t.initTrinoDbClient(ctx)
		if err != nil {
			provider.Logger(*ctx).WithError(err).Errorw(
				"error initializing trino db client",
				map[string]interface{}{"query": q, "trinoHost": t.url.Host})
			return nil, err
		}
	}

	err := t.db.PingContext(*ctx)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"trino db ping failed",
			map[string]interface{}{"trinoHost": t.url.Host})
		return nil, err
	}

	// span
	rows, err := t.db.QueryContext(*ctx, q, sql.Named("X-Trino-User", t.user))
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"error executing query",
			map[string]interface{}{"query": q, "trinoHost": t.url.Host})
		return nil, err
	}

	provider.Logger(*ctx).Debugw(
		"trino query submission successful",
		map[string]interface{}{"query": q, "trinoHost": t.url.Host})

	return rows, nil
}

func (t *TrinoClient) Teardown(ctx *context.Context) error {
	if t.db != nil {
		// log
		err := t.db.Close()
		if err != nil {
			provider.Logger(*ctx).WithError(err).Errorw(
				"error closing trino db client",
				map[string]interface{}{"trinoHost": t.url.Host})
			return err
		}
	}
	return nil
}
