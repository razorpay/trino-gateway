package utils

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
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

func (suite *UtilsSuite) Test_stringifyHttpRequestOrResponse() {
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
