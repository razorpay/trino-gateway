package router

import (
	"fmt"

	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

type ClientRequest interface {
	isClientRequest()
	Validate() error
}

type UiRequest struct {
	ClientRequest
	queryId string
}

func (UiRequest) isClientRequest() {}

func (r UiRequest) Validate() error {
	if r.queryId == "" {
		tag := "ui"
		return fmt.Errorf("%s: %s", tag, "Missing query id")
	}
	return nil
}

type QueryRequest struct {
	ClientRequest
	headerConnectionProperties string
	headerClientTags           string
	incomingPort               int32
	transactionId              string
	clientHost                 string
	Query                      *gatewayv1.Query
}

func (QueryRequest) isClientRequest() {}
func (r QueryRequest) Validate() error {
	tag := "query submission"
	if r.Query.GetUsername() == "" {
		return fmt.Errorf("%s: %s", tag, "Missing Trino Username header")
	}
	if r.Query.GetText() == "" {
		return fmt.Errorf("%s: %s", tag, "Missing Query text")
	}

	// TODO: remove it once transaction support is added
	// Looker's Presto client sends `X-Presto-Transaction-Id: NONE`
	// whereas trino client doesnt send it if its not set
	if !(r.transactionId == "" || r.transactionId == "NONE") {
		return fmt.Errorf("%s: %s", tag, "Transactions are not supported in gateway.")
	}
	return nil
}

type ApiRequest struct {
	ClientRequest
}

func (ApiRequest) isClientRequest() {}
func (r ApiRequest) Validate() error {
	return nil
}
