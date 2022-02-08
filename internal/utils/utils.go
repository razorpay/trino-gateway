package utils

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"reflect"
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

func SliceContains(a interface{}, e interface{}) bool {
	v := reflect.ValueOf(a)

	for i := 0; i < v.Len(); i++ {
		if v.Index(i).Interface() == e {
			return true
		}
	}
	return false
}

// Finds intersection of 2 slices via simple comparison approach O(n^2)
func SimpleSliceIntersection(a interface{}, b interface{}) []interface{} {
	set := make([]interface{}, 0)
	av := reflect.ValueOf(a)

	for i := 0; i < av.Len(); i++ {
		el := av.Index(i).Interface()
		if SliceContains(b, el) {
			set = append(set, el)
		}
	}

	return set
}

func SimpleStringSliceIntersection(a []string, b []string) []string {
	var res []string
	for _, i := range SimpleSliceIntersection(a, b) {
		_i := i.(string)
		res = append(res, _i)
	}
	return res
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
