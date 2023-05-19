package utils

import (
	"bytes"
	"compress/gzip"
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
