package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/razorpay/trino-gateway/internal/provider"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

func (r *RouterServer) handleRedirect(ctx *context.Context, resp *http.Response) error {
	// This needs more testing with looker

	// Handle Redirects
	// TODO: Clean it up
	// TODO - validate its working for all use cases
	regex := regexp.MustCompile(`\w+\:\/\/[^\/]*(.*)`)
	if resp.Header.Get("Location") != ("") {
		oldLoc := resp.Header.Get("Location")
		newLoc := fmt.Sprint("http://", r.routerHostname, regex.ReplaceAllString(oldLoc, "$1"))
		resp.Header.Set("Location", newLoc)
	}
	return nil
}

func (r *RouterServer) processResponse(
	ctx *context.Context,
	resp *http.Response,
	q *gatewayv1.Query,
) error {

	// TODO - fix redirect
	_ = r.handleRedirect(ctx, resp)

	isQuerySubmissionSuccessful := (resp.StatusCode >= 200) && (resp.StatusCode < 300)
	if !isQuerySubmissionSuccessful {
		provider.Logger(*ctx).Errorw(
			fmt.Sprint(LOG_TAG, "Query Submission unsuccessful"),
			map[string]interface{}{
				"serverResponse": stringifyHttpResponse(ctx, resp),
			})
		return nil
	}

	provider.Logger(*ctx).Debug(LOG_TAG + "Query Submission successful")

	req := q
	body, err := parseBody(ctx, &resp.Body)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Error(fmt.Sprint(LOG_TAG, "unable to parse body of server response"))
	}

	go func() {
		req.Id = extractQueryIdFromServerResponse(ctx, body)
		req.SubmittedAt = time.Now().Unix()

		_, err = r.gatewayApiClient.Query.CreateOrUpdateQuery(*ctx, req)
		if err != nil {
			provider.Logger(
				*ctx).WithError(err).Errorw(
				fmt.Sprint(LOG_TAG, "Unable to save query"),
				map[string]interface{}{
					"query_id": req.Id,
				})
		}
	}()

	provider.Logger(*ctx).Debugw("Server Response Processed", map[string]interface{}{
		"resp": stringifyHttpResponse(ctx, resp),
	})

	return nil
}

func extractQueryIdFromServerResponse(ctx *context.Context, body string) string {
	provider.Logger(*ctx).Debugw(fmt.Sprint(LOG_TAG, "extracting queryId from server response"),
		map[string]interface{}{
			"body": body,
		})
	var resp struct{ Id string }
	json.Unmarshal([]byte(body), &resp)
	return resp.Id
}
