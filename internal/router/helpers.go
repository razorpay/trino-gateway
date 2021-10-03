package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	"github.com/razorpay/trino-gateway/internal/provider"
)

func extractQueryIdFromServerResponse(ctx *context.Context, body string) string {
	provider.Logger(*ctx).Debugw(fmt.Sprint(LOG_TAG, "extracting queryId from server response"),
		map[string]interface{}{
			"body": body,
		})
	var resp struct{ Id string }
	json.Unmarshal([]byte(body), &resp)
	return resp.Id
}

func stringifyHttpRequest(ctx *context.Context, req *http.Request) string {
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		provider.Logger(*ctx).Errorw(
			fmt.Sprint(LOG_TAG, "Unable to stringify http request"),
			map[string]interface{}{
				"error": err.Error(),
			})
	}
	return string(requestDump)
}

func stringifyHttpResponse(ctx *context.Context, req *http.Response) string {
	responseDump, err := httputil.DumpResponse(req, true)
	if err != nil {
		provider.Logger(*ctx).Errorw(
			fmt.Sprint(LOG_TAG, "Unable to stringify http response"),
			map[string]interface{}{
				"error": err.Error(),
			})
	}
	return string(responseDump)
}

func parseBody(ctx *context.Context, body *io.ReadCloser) (string, error) {
	//b := req.Body

	bodyBytes, err := io.ReadAll(*body)
	if err != nil {
		return "", err
	}
	// since its a ReadCloser type, the stream will be empty after its read once
	// ensure it is restored in original request
	*body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes), nil
}
