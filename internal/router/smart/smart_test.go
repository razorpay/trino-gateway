package smart

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestParsePreparedStatements_DecodesNameValuePairs(t *testing.T) {
	// Trino sends one or more "name=url-encoded-sql" entries, comma separated
	// within a header value and/or across repeated headers.
	got := ParsePreparedStatements([]string{
		"_trino_go=SELECT+id+FROM+t+WHERE+id+%3D+%3F,second=SELECT%201",
		"third=select%20%27a%2Cb%27",
	})
	want := map[string]string{
		"_trino_go": "SELECT id FROM t WHERE id = ?",
		"second":    "SELECT 1",
		"third":     "select 'a,b'",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestParsePreparedStatements_SkipsMalformedEntries(t *testing.T) {
	got := ParsePreparedStatements([]string{"noequals", "ok=SELECT%201", "bad=%zz"})
	if len(got) != 1 || got["ok"] != "SELECT 1" {
		t.Errorf("got %v", got)
	}
}

func routerStub(t *testing.T, status int, body string, delay time.Duration) (*httptest.Server, *Request) {
	t.Helper()
	var seen Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&seen)
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
}

const decisionJSON = `{"tier":"S","backend":"trino-srm","session":{"query_max_execution_time":"5m"},"watch":false,
  "decision":{"source":"history","fingerprint":"select ?","confidence":0.95,"reason":"seen 12×"}}`

func TestClient_RoutePostsRequestAndDecodesDecision(t *testing.T) {
	srv, seen := routerStub(t, 200, decisionJSON, 0)
	c := NewClient(srv.URL, 500*time.Millisecond)
	d, err := c.Route(context.Background(), Request{SQL: "SELECT 1", User: "u", Group: "adhoc",
		Tiers: map[string][]string{"S": {"trino-srm"}}})
	if err != nil {
		t.Fatal(err)
	}
	if seen.SQL != "SELECT 1" || seen.Group != "adhoc" || seen.Tiers["S"][0] != "trino-srm" {
		t.Errorf("router saw %+v", *seen)
	}
	if d.Backend != "trino-srm" || d.Tier != "S" || d.Session["query_max_execution_time"] != "5m" || d.Explain.Source != "history" {
		t.Errorf("decision=%+v", d)
	}
}

func TestClient_RejectIsADecisionNotAnError(t *testing.T) {
	srv, _ := routerStub(t, 422, `{"reject":{"reason":"always fails","override":"force-run"},"decision":{"source":"history"}}`, 0)
	d, err := NewClient(srv.URL, 500*time.Millisecond).Route(context.Background(), Request{SQL: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Reject == nil || d.Reject.Reason != "always fails" {
		t.Errorf("decision=%+v", d)
	}
}

func TestClient_TimeoutIsAnError(t *testing.T) {
	srv, _ := routerStub(t, 200, decisionJSON, 300*time.Millisecond)
	_, err := NewClient(srv.URL, 50*time.Millisecond).Route(context.Background(), Request{SQL: "x"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestClient_ServerErrorIsAnError(t *testing.T) {
	srv, _ := routerStub(t, 503, "no backend", 0)
	if _, err := NewClient(srv.URL, 500*time.Millisecond).Route(context.Background(), Request{SQL: "x"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestApply_ShadowKeepsCurrentBackend(t *testing.T) {
	d := &Decision{Backend: "trino-srm", Tier: "S"}
	got, err := Apply(ModeShadow, d, "trino-adhoc")
	if err != nil || got != "trino-adhoc" {
		t.Errorf("got %q err=%v", got, err)
	}
}

func TestApply_EnforceUsesDecisionBackend(t *testing.T) {
	d := &Decision{Backend: "trino-srm", Tier: "S"}
	got, err := Apply(ModeEnforce, d, "trino-adhoc")
	if err != nil || got != "trino-srm" {
		t.Errorf("got %q err=%v", got, err)
	}
}

func TestApply_EnforceRejectIsAnError(t *testing.T) {
	d := &Decision{Reject: &Reject{Reason: "always fails"}}
	if _, err := Apply(ModeEnforce, d, "trino-adhoc"); err == nil {
		t.Fatal("expected error")
	}
}

func TestApply_ShadowRejectIsNotAnError(t *testing.T) {
	d := &Decision{Reject: &Reject{Reason: "always fails"}}
	got, err := Apply(ModeShadow, d, "trino-adhoc")
	if err != nil || got != "trino-adhoc" {
		t.Errorf("got %q err=%v", got, err)
	}
}

func TestApply_EnforceWithoutBackendKeepsCurrent(t *testing.T) {
	got, err := Apply(ModeEnforce, &Decision{}, "trino-adhoc")
	if err != nil || got != "trino-adhoc" {
		t.Errorf("got %q err=%v", got, err)
	}
}

func TestNormalizeTiers_UppercasesTierKeys(t *testing.T) {
	// viper lowercases every TOML key, so tiers.adhoc.S arrives as "s".
	in := map[string]map[string][]string{"adhoc": {"s": {"trino-srm"}, "M": {"trino-dev"}, "l": {"trino-adhoc"}}}
	got := NormalizeTiers(in)
	want := map[string]map[string][]string{"adhoc": {"S": {"trino-srm"}, "M": {"trino-dev"}, "L": {"trino-adhoc"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestTiersFor_ConfiguredGroupUsesItsTiers(t *testing.T) {
	cfg := map[string]map[string][]string{"adhoc": {"S": {"trino-srm"}, "L": {"trino-adhoc"}}}
	got := TiersFor(cfg, "adhoc", "trino-adhoc")
	if !reflect.DeepEqual(got, cfg["adhoc"]) {
		t.Errorf("got %v", got)
	}
}

func TestTiersFor_UnknownGroupFallsBackToStrategyBackend(t *testing.T) {
	// Shadow mode must produce a prediction for every group, configured or
	// not; with one tier holding the strategy's backend the router can only
	// pick that backend, so enforce mode is a no-op for such groups.
	got := TiersFor(nil, "looker", "trino-looker-b")
	want := map[string][]string{"M": {"trino-looker-b"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestClient_WarmCallsHealthzAndReusesConnection(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	if err := NewClient(srv.URL, time.Second).Warm(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "/healthz" {
		t.Errorf("paths=%v", paths)
	}
}

func TestClient_WarmReportsUnreachableRouter(t *testing.T) {
	if err := NewClient("http://127.0.0.1:1", 100*time.Millisecond).Warm(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionLines_KeepsEveryClientLineAndEscapesValues(t *testing.T) {
	// Trino's Java client sends one header line per property.
	existing := []string{"join_distribution_type=BROADCAST", "query_max_execution_time=1h,task_concurrency=8"}
	d := &Decision{Session: map[string]string{"query_max_execution_time": "5m", "query_max_memory": "1GB", "odd": "a b,c"}}
	got := SessionLines(existing, d)
	want := []string{"odd=a+b%2Cc", "query_max_memory=1GB"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestSessionLines_NothingToAdd(t *testing.T) {
	if got := SessionLines(nil, &Decision{}); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestParsePreparedStatements_DecodesNames(t *testing.T) {
	got := ParsePreparedStatements([]string{"my%20stmt=SELECT%201"})
	if got["my stmt"] != "SELECT 1" {
		t.Errorf("got %v", got)
	}
}

func TestTiersFor_GroupIdIsCaseInsensitive(t *testing.T) {
	// viper lowercases config map keys; group ids in the database may not be.
	cfg := map[string]map[string][]string{"adhoc": {"S": {"trino-srm"}}}
	if got := TiersFor(cfg, "AdHoc", "x"); got["S"][0] != "trino-srm" {
		t.Errorf("got %v", got)
	}
}

func TestValidBackend_MustBeInATierAndActive(t *testing.T) {
	tiers := map[string][]string{"S": {"trino-srm"}, "L": {"trino-adhoc"}}
	active := map[string]bool{"trino-srm": true, "trino-adhoc": false}
	if !ValidBackend("trino-srm", tiers, active) {
		t.Error("in-tier active backend rejected")
	}
	if ValidBackend("trino-adhoc", tiers, active) {
		t.Error("inactive backend accepted")
	}
	if ValidBackend("trino-finance", tiers, map[string]bool{"trino-finance": true}) {
		t.Error("backend outside the tiers accepted")
	}
}

func TestNewClient_ZeroTimeoutIsClampedToDefault(t *testing.T) {
	if c := NewClient("http://x", 0); c.http.Timeout != DefaultTimeout {
		t.Errorf("timeout %v", c.http.Timeout)
	}
}

func TestNewClient_SharesAPooledTransport(t *testing.T) {
	a, b := NewClient("http://x", time.Second), NewClient("http://y", time.Second)
	ta, ok := a.http.Transport.(*http.Transport)
	if !ok || ta != b.http.Transport || ta.MaxIdleConnsPerHost < 32 {
		t.Errorf("transport not shared or pool too small: %#v", a.http.Transport)
	}
}

func TestLabel_MapsUnknownValuesToOther(t *testing.T) {
	if Label("history", SourceLabels) != "history" || Label("weird", SourceLabels) != "other" {
		t.Error("label allowlist")
	}
}
