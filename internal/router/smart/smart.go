// Package smart is the gateway-side client for the smart router service
// (trino_router). The gateway sends the query plus the groups and backends
// its own policies already allow; the router answers with a tier, a backend,
// session-property guardrails and an explanation. Errors and timeouts are the
// caller's signal to fall back to the existing strategy, so the router can
// never take query submission down.
package smart

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Request mirrors router.Request in trino_router.
type Request struct {
	SQL                string              `json:"sql"`
	User               string              `json:"user"`
	Source             string              `json:"source"`
	ClientTags         []string            `json:"client_tags,omitempty"`
	Port               int                 `json:"port,omitempty"`
	Group              string              `json:"group"`
	Tiers              map[string][]string `json:"tiers"`
	PreparedStatements map[string]string   `json:"prepared_statements,omitempty"`
}

type Explain struct {
	Source      string  `json:"source"`
	Fingerprint string  `json:"fingerprint"`
	CPUClass    int     `json:"cpu_class,omitempty"`
	MemClass    int     `json:"mem_class,omitempty"`
	BytesClass  int     `json:"bytes_class,omitempty"`
	FailP       float64 `json:"fail_p"`
	Confidence  float64 `json:"confidence"`
	Reason      string  `json:"reason"`
}

type Reject struct {
	Reason   string `json:"reason"`
	Override string `json:"override"`
}

// Decision mirrors router.Decision in trino_router.
type Decision struct {
	Tier    string            `json:"tier,omitempty"`
	Backend string            `json:"backend,omitempty"`
	Session map[string]string `json:"session,omitempty"`
	Watch   bool              `json:"watch"`
	Reject  *Reject           `json:"reject,omitempty"`
	Explain Explain           `json:"decision"`
}

const (
	ModeShadow  = "shadow"  // log the decision, route as before
	ModeEnforce = "enforce" // follow the decision
)

// RejectedError carries the router's reason so the gateway can show it.
type RejectedError struct{ Reject Reject }

func (e *RejectedError) Error() string { return "rejected by smart router: " + e.Reject.Reason }

// Apply resolves the backend to use given the mode, the router's decision and
// the backend the gateway's own strategy picked.
func Apply(mode string, d *Decision, current string) (string, error) {
	if mode != ModeEnforce || d == nil {
		return current, nil
	}
	if d.Reject != nil {
		return "", &RejectedError{Reject: *d.Reject}
	}
	if d.Backend == "" {
		return current, nil
	}
	return d.Backend, nil
}

type Client struct {
	url  string
	http *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		url:  strings.TrimRight(baseURL, "/") + "/v1/route",
		http: &http.Client{Timeout: timeout},
	}
}

// Route asks the router for a decision. A 422 (reject) is a decision, not an
// error; any other non-200 status, transport failure or timeout is an error.
func (c *Client) Route(ctx context.Context, rq Request) (*Decision, error) {
	body, err := json.Marshal(rq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnprocessableEntity {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("router returned %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	var d Decision
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, fmt.Errorf("router response: %w", err)
	}
	return &d, nil
}

// ParsePreparedStatements decodes X-Trino-Prepared-Statement header values,
// each a comma-separated list of name=url-encoded-sql. Malformed entries are
// skipped so one bad statement cannot block routing.
func ParsePreparedStatements(headers []string) map[string]string {
	out := map[string]string{}
	for _, h := range headers {
		for _, entry := range strings.Split(h, ",") {
			name, enc, ok := strings.Cut(strings.TrimSpace(entry), "=")
			if !ok {
				continue
			}
			sql, err := url.QueryUnescape(enc)
			if err != nil {
				continue
			}
			out[name] = sql
		}
	}
	return out
}

// NormalizeTiers upper-cases tier names (S, M, L). viper lowercases every
// TOML key, so tiers.adhoc.S reaches the gateway as "s".
func NormalizeTiers(tiers map[string]map[string][]string) map[string]map[string][]string {
	out := make(map[string]map[string][]string, len(tiers))
	for group, byTier := range tiers {
		m := make(map[string][]string, len(byTier))
		for tier, backends := range byTier {
			m[strings.ToUpper(tier)] = backends
		}
		out[group] = m
	}
	return out
}

// TiersFor returns the configured tiers of group, or a single M tier holding
// the strategy's backend when the group has none. Shadow mode then gets a
// prediction for every query, while enforce mode can only re-pick the same
// backend for unconfigured groups.
func TiersFor(cfg map[string]map[string][]string, group, strategyBackend string) map[string][]string {
	if tiers, ok := cfg[group]; ok && len(tiers) > 0 {
		return tiers
	}
	return map[string][]string{"M": {strategyBackend}}
}

// SessionHeader merges the decision's session properties into an existing
// X-Trino-Session value. A property the client already set is kept, so a
// user's explicit limit is never tightened silently.
func SessionHeader(existing string, d *Decision) string {
	have := map[string]bool{}
	parts := []string{}
	for _, kv := range strings.Split(existing, ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		k, _, _ := strings.Cut(kv, "=")
		have[k] = true
		parts = append(parts, kv)
	}
	keys := make([]string, 0, len(d.Session))
	for k := range d.Session {
		if !have[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, k+"="+d.Session[k])
	}
	return strings.Join(parts, ",")
}
