package router

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type HelpersSuite struct {
	suite.Suite
	ctx *context.Context
}

func (suite *HelpersSuite) SetupTest() {
	c := context.Background()
	suite.ctx = &c
}

func (suite *HelpersSuite) Test_stringifyHttpRequest() {
}

func (suite *HelpersSuite) Test_stringifyHttpResponse() {
}

func (suite *HelpersSuite) Test_parseBody() {
	str := "body"
	stringReader := strings.NewReader(str)
	stringReadCloser := io.NopCloser(stringReader)

	tst := func() string {
		s, _ := parseBody(suite.ctx, &stringReadCloser)
		return s
	}

	suite.Equalf(str, tst, "Failed to extract string from body")
	suite.Equalf(str, tst, "String extraction is not idempotent")
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(HelpersSuite))
}
