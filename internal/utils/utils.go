package utils

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/robfig/cron/v3"
)

/*
Checks whether provided time object is in 1 minute window of cron expression
*/
func IsTimeInCron(ctx *context.Context, t time.Time, sched string) (bool, error) {
	s, err := cron.ParseStandard(sched)
	if err != nil {
		return false, err
	}
	nextRun := s.Next(t)

	provider.Logger(*ctx).Debugw(
		"Evaluated next valid ts from cron expression",
		map[string]interface{}{
			"providedTime": t,
			"nextRun":      nextRun,
		},
	)

	return nextRun.Sub(t).Minutes() <= 1, nil
}

func SliceContains[T comparable](collection []T, element T) bool {
	for _, item := range collection {
		if item == element {
			return true
		}
	}

	return false
}

// Finds intersection of 2 slices
func SimpleSliceIntersection[T comparable](list1 []T, list2 []T) []T {
	result := []T{}
	seen := map[T]struct{}{}

	for _, elem := range list1 {
		seen[elem] = struct{}{}
	}

	for _, elem := range list2 {
		if _, ok := seen[elem]; ok {
			result = append(result, elem)
		}
	}

	return result
}

func StringifyHttpRequest(ctx *context.Context, req *http.Request) string {
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		provider.Logger(*ctx).Errorw(
			"Unable to stringify http request",
			map[string]interface{}{
				"error": err.Error(),
			})
	}
	return string(requestDump)
}

func StringifyHttpResponse(ctx *context.Context, req *http.Response) string {
	responseDump, err := httputil.DumpResponse(req, true)
	if err != nil {
		provider.Logger(*ctx).Errorw(
			"Unable to stringify http response",
			map[string]interface{}{
				"error": err.Error(),
			})
	}
	return string(responseDump)
}

func ParseHttpPayloadBody(ctx *context.Context, body *io.ReadCloser) (string, error) {
	// b := req.Body

	bodyBytes, err := io.ReadAll(*body)
	if err != nil {
		return "", err
	}
	// since its a ReadCloser type, the stream will be empty after its read once
	// ensure it is restored in original request
	*body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes), nil
}
