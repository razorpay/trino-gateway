package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/internal/utils"
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

func (r *RouterServer) ProcessResponse(
	ctx *context.Context,
	resp *http.Response,
	cReq ClientRequest,
) error {
	switch stCode := resp.StatusCode; true {
	case stCode >= 200 && stCode < 300:
		// TODO - fix redirect
		_ = r.handleRedirect(ctx, resp)
	case stCode >= 300 && stCode < 400:
		// http3xx -> server sent redirection, gateway doesn't need to modify anything here
		// Assuming Clients can directly connect to redirected Uri
	default:
		provider.Logger(*ctx).Errorw(
			fmt.Sprint(LOG_TAG, "Routing unsuccessful"),
			map[string]interface{}{
				"serverResponse": utils.StringifyHttpRequestOrResponse(ctx, resp),
			})
		return nil
	}

	provider.Logger(*ctx).Debug(LOG_TAG + "Routing successful")

	switch nt := cReq.(type) {
	case *ApiRequest:
		return nil
	case *UiRequest:
		return nil
	case *QueryRequest:
		req := nt.Query
		body, err := utils.ParseHttpPayloadBody(ctx, &resp.Body, utils.GetHttpBodyEncoding(ctx, resp))
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
			"resp": utils.StringifyHttpRequestOrResponse(ctx, resp),
		})

		return nil
	default:
		return nil
	}
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
