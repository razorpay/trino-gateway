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
	base string
	url  string
	http *http.Client
}

// DefaultTimeout applies when the configured timeout is zero or negative;
// http.Client treats 0 as "no timeout", which could stall query submission.
const DefaultTimeout = 50 * time.Millisecond

// transport is shared by every Client (one per gateway listener port) and kept
// apart from the reverse proxy's DefaultTransport. DefaultTransport keeps only
// 2 idle connections per host, so concurrent submissions would keep redialling
// the router inside the routing timeout.
var transport = func() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 256
	t.MaxIdleConnsPerHost = 64
	t.IdleConnTimeout = 5 * time.Minute
	return t
}()

func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	base := strings.TrimRight(baseURL, "/")
	return &Client{
		base: base,
		url:  base + "/v1/route",
		http: &http.Client{Timeout: timeout, Transport: transport},
	}
}

// Warm opens a kept-alive connection to the router so the first query does
// not pay for DNS (several search-domain lookups under ndots:5) and TCP setup
// inside its routing timeout. It uses its own, longer deadline.
func (c *Client) Warm(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Transport: c.http.Transport}).Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("router healthz returned %d", resp.StatusCode)
	}
	return nil
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
			encName, enc, ok := strings.Cut(strings.TrimSpace(entry), "=")
			if !ok {
				continue
			}
			// Trino URL-encodes both the statement name and its SQL.
			name, err1 := url.QueryUnescape(encName)
			sql, err2 := url.QueryUnescape(enc)
			if err1 != nil || err2 != nil {
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
	if tiers, ok := cfg[strings.ToLower(group)]; ok && len(tiers) > 0 {
		return tiers
	}
	return map[string][]string{"M": {strategyBackend}}
}

// SessionLines returns the X-Trino-Session lines to add for the decision's
// session properties: one "name=url-encoded-value" line per property the
// client has not already set in any of its existing lines (Trino's Java client
// sends one line per property). A user's explicit value is never overridden.
func SessionLines(existing []string, d *Decision) []string {
	have := map[string]bool{}
	for _, line := range existing {
		for _, kv := range strings.Split(line, ",") {
			if k, _, _ := strings.Cut(strings.TrimSpace(kv), "="); k != "" {
				have[k] = true
			}
		}
	}
	keys := make([]string, 0, len(d.Session))
	for k := range d.Session {
		if !have[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+url.QueryEscape(d.Session[k]))
	}
	return out
}

// ValidBackend reports whether the router's backend may be used: it must be
// one of the tiers the gateway sent and active (enabled and healthy) right now.
func ValidBackend(backend string, tiers map[string][]string, active map[string]bool) bool {
	if !active[backend] {
		return false
	}
	for _, bs := range tiers {
		for _, b := range bs {
			if b == backend {
				return true
			}
		}
	}
	return false
}

var (
	SourceLabels = []string{"metadata", "history", "model", "default"}
	TierLabels   = []string{"S", "M", "L", ""}
)

// Label maps a router-supplied value onto a fixed set for metric labels, so a
// misbehaving router cannot blow up label cardinality.
func Label(v string, allowed []string) string {
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	return "other"
}
