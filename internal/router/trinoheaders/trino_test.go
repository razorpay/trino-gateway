package trinoheaders

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Get(t *testing.T) {
	trinoHttpReq := &http.Request{
		Header: map[string][]string{
			"X-Trino-User":                  {"user"},
			"X-Trino-Connection-Properties": {"connProps"},
		},
	}
	assert.Equal(t, Get("User", trinoHttpReq), "user")
	assert.Equal(t, Get("Connection-Properties", trinoHttpReq), "connProps")

	prestoHttpReq := &http.Request{
		Header: map[string][]string{
			"X-Presto-User":                  {"user"},
			"X-Presto-Connection-Properties": {"connProps"},
		},
	}
	assert.Equal(t, Get("User", prestoHttpReq), "user")
	assert.Equal(t, Get("Connection-Properties", prestoHttpReq), "connProps")
}
