package router

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/razorpay/trino-gateway/internal/router/smart"
)

func TestHandlePreRoutingError_SurfacesSmartRouterReject(t *testing.T) {
	ctx := context.Background()
	err := &smart.RejectedError{Reject: smart.Reject{Reason: "This query shape failed 57 of 57 recent runs with TABLE_NOT_FOUND.", Override: "set client tag force-run to bypass"}}
	status, msg := (&RouterServer{}).handlePreRoutingError(&ctx, err)
	if status != http.StatusBadRequest {
		t.Errorf("status=%d", status)
	}
	if !strings.Contains(msg, "TABLE_NOT_FOUND") || !strings.Contains(msg, "force-run") {
		t.Errorf("msg=%q", msg)
	}
}

func TestHandlePreRoutingError_GenericForOtherErrors(t *testing.T) {
	ctx := context.Background()
	status, msg := (&RouterServer{}).handlePreRoutingError(&ctx, errors.New("boom"))
	if status != http.StatusBadRequest || msg != "Gateway couldn't process this request" {
		t.Errorf("status=%d msg=%q", status, msg)
	}
}
