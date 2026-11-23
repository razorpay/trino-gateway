package utils

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/razorpay/trino-gateway/pkg/logger"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type UtilsSuite struct {
	suite.Suite
	ctx *context.Context
}

func (suite *UtilsSuite) SetupTest() {
	lgrConfig := logger.Config{
		LogLevel: logger.Warn,
	}

	l, err := logger.NewLogger(lgrConfig)
	if err != nil {
		panic("failed to initialize logger")
	}

	c := context.WithValue(context.Background(), logger.LoggerCtxKey, l)

	suite.ctx = &c
}

func (suite *UtilsSuite) Test_IsTimeInCron() {
	// func (c *Core) isCurrentTimeInCron(ctx *context.Context, sched string) (bool, error)

	tst := func(t time.Time, sched string) bool {
		s, _ := IsTimeInCron(suite.ctx, t, sched)
		return s
	}

	suite.Equalf(
		true,
		tst(time.Now(), "* * * * *"),
		"Failure",
	)
}

func (suite *UtilsSuite) Test_getHttpBodyEncoding() {
}

// randomSecret returns a random hex string. Test credentials are generated at
// runtime so no credential-like literals are committed to the source tree.
func randomSecret(suite *UtilsSuite) string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	suite.Require().NoError(err)
	return hex.EncodeToString(b)
}

func (suite *UtilsSuite) Test_stringifyHttpRequestOrResponse() {
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString(
		[]byte("user:"+randomSecret(suite)))
	trinoPass := randomSecret(suite)
	prestoPass := randomSecret(suite)
	authKey := randomSecret(suite)
	cookie := "session=" + randomSecret(suite)
	queryText := "SELECT cc_number FROM payments"
	respBody := `{"id":"20240101_000000_00000_abcde","rows":[{"cc_number":"4111111111111111"}]}`

	req := httptest.NewRequest("POST", "/v1/statement", strings.NewReader(queryText))
	req.Header.Set("Authorization", basicAuth)
	req.Header.Set("X-Trino-Password", trinoPass)
	req.Header.Set("X-Presto-Password", prestoPass)
	req.Header.Set("X-Auth-Key", authKey)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("X-Trino-User", "dev")

	reqDump := StringifyHttpRequestOrResponse(suite.ctx, req)

	// operational metadata is still logged
	suite.Containsf(reqDump, "POST /v1/statement", "request line not dumped")
	suite.Containsf(reqDump, "X-Trino-User: dev", "non sensitive header not dumped")

	// credentials & body never make it to the logs
	for _, secret := range []string{basicAuth, trinoPass, prestoPass, authKey, cookie} {
		suite.NotContainsf(reqDump, secret, "sensitive header value leaked into log")
	}
	for _, fragment := range []string{queryText, "SELECT", "payments"} {
		suite.NotContainsf(reqDump, fragment, "request body leaked into log")
	}

	// the request being forwarded is not mutated by logging
	suite.Equalf(basicAuth, req.Header.Get("Authorization"), "original request header was mutated")

	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{},
		Body:          io.NopCloser(strings.NewReader(respBody)),
		ContentLength: int64(len(respBody)),
	}
	resp.Header.Set("Set-Cookie", cookie)
	resp.Header.Set("Content-Type", "application/json")

	respDump := StringifyHttpRequestOrResponse(suite.ctx, resp)

	suite.Containsf(respDump, "HTTP/1.1 200 OK", "status line not dumped")
	suite.NotContainsf(respDump, cookie, "sensitive response header value leaked into log")
	for _, fragment := range []string{respBody, "20240101_000000_00000_abcde", "4111111111111111"} {
		suite.NotContainsf(respDump, fragment, "response body leaked into log")
	}
}

func (suite *UtilsSuite) Test_parseBody() {
	str := "body"
	stringReader := strings.NewReader(str)
	stringReadCloser := io.NopCloser(stringReader)

	tst := func() string {
		s, _ := ParseHttpPayloadBody(suite.ctx, &stringReadCloser, "")
		return s
	}

	suite.Equalf(str, tst(), "Failed to extract string from body")
	suite.Equalf(str, tst(), "String extraction is not idempotent")

	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(str)); err != nil {
		panic(err)
	}
	if err := gz.Close(); err != nil {
		panic(err)
	}

	strGzipped := b.String()
	stringReaderGzipped := strings.NewReader(strGzipped)
	stringReadCloserGzipped := io.NopCloser(stringReaderGzipped)

	tst_gzipped := func() string {
		s, _ := ParseHttpPayloadBody(suite.ctx, &stringReadCloserGzipped, "gzip")
		return s
	}

	suite.Equalf(str, tst_gzipped(), "Failed to extract string from body")
	suite.Equalf(str, tst_gzipped(), "String extraction is not idempotent")
}

func (suite *UtilsSuite) Test_InMemorySimpleCache_Get() {
	authCache := &InMemorySimpleCache{
		Cache: make(map[string]struct {
			Timestamp time.Time
			Value     string
		}),
	}
	key := "testKey"
	value := "testValue"
	authCache.Cache[key] = struct {
		Timestamp time.Time
		Value     string
	}{
		Timestamp: time.Now(),
		Value:     value,
	}

	entry, exists := authCache.Get(key)
	suite.Truef(exists, "Entry not found in cache.")
	if exists {
		suite.Equalf(value, entry, "Cached value doesn't match.")
	}
}

func (suite *UtilsSuite) Test_InMemorySimpleCache_Get_InfiniteExpiry() {
	authCache := &InMemorySimpleCache{
		Cache: make(map[string]struct {
			Timestamp time.Time
			Value     string
		}),
		ExpiryInterval: 0 * time.Second,
	}
	key := "testKey"
	value := "testValue"
	authCache.Cache[key] = struct {
		Timestamp time.Time
		Value     string
	}{
		Timestamp: time.Now().Add(-1000 * time.Hour),
		Value:     value,
	}

	entry, exists := authCache.Get(key)
	suite.Truef(exists, "Entry not found in cache.")
	if exists {
		suite.Equalf(value, entry, "Cached value doesn't match.")
	}
}

func (suite *UtilsSuite) Test_InMemorySimpleCache_Get_Expired() {
	expiryInterval := 2 * time.Second
	authCache := &InMemorySimpleCache{
		Cache: make(map[string]struct {
			Timestamp time.Time
			Value     string
		}),
		ExpiryInterval: expiryInterval,
	}
	key := "testKey"
	value := "testValue"
	authCache.Cache[key] = struct {
		Timestamp time.Time
		Value     string
	}{
		Timestamp: time.Now().Add(-1 * expiryInterval).Add(-1 * time.Second),
		Value:     value,
	}

	_, exists := authCache.Get(key)
	suite.False(exists, "Entry not expired.")
}

func (suite *UtilsSuite) Test_InMemorySimpleCache_Update() {
	authCache := &InMemorySimpleCache{
		Cache: make(map[string]struct {
			Timestamp time.Time
			Value     string
		}),
	}
	key := "testKey"
	value := "testValue"
	authCache.Update(key, value)

	entry, exists := authCache.Cache[key]
	suite.Truef(exists, "Entry not found in cache.")
	if exists {
		suite.Equalf(value, entry.Value, "Cached value doesn't match.")
	}
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
}
