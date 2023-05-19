package utils

import (
	"bytes"
	"compress/gzip"
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

func GetHttpBodyEncoding[T *http.Request | *http.Response](ctx *context.Context, r T) string {
	enc := ""
	headerKey := "Content-Encoding"
	switch v := any(r).(type) {
	case *http.Request:
		enc = v.Header.Get(headerKey)
	case *http.Response:
		enc = v.Header.Get(headerKey)
	}
	return enc
}

func StringifyHttpRequestOrResponse[T *http.Request | *http.Response](ctx *context.Context, r T) string {
	canDumpBody := GetHttpBodyEncoding(ctx, r) == ""
	if !canDumpBody {
		provider.Logger(*ctx).Debug(
			"Encoded body in http payload, assuming binary data and skipping dump of body")
	}
	var res []byte
	var err error
	switch v := any(r).(type) {
	case *http.Request:
		res, err = httputil.DumpRequest(v, canDumpBody)
	case *http.Response:
		res, err = httputil.DumpResponse(v, canDumpBody)
	}
	if err != nil {
		provider.Logger(*ctx).Errorw(
			"Unable to stringify http payload",
			map[string]interface{}{
				"error": err.Error(),
			})
	}
	return string(res)
}

func ParseHttpPayloadBody(ctx *context.Context, body *io.ReadCloser, encoding string) (string, error) {
	bodyBytes, err := io.ReadAll(*body)
	if err != nil {
		return "", err
	}
	// since its a ReadCloser type, the stream will be empty after its read once
	// ensure it is restored in original object
	*body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	switch encoding {
	case "gzip":
		var reader io.ReadCloser
		reader, err = gzip.NewReader(bytes.NewReader([]byte(bodyBytes)))
		if err != nil {
			provider.Logger(
				*ctx).WithError(err).Error(
				"Unable to decompress gzip encoded response")
		}
		defer reader.Close()
		bb, err := io.ReadAll(reader)
		if err != nil {
			return "", err
		}

		return string(bb), nil
	default:
		return string(bodyBytes), nil
	}
}
