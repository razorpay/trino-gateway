package router

import (
	"fmt"

	"github.com/razorpay/trino-gateway/internal/router/smart"
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

type QueryApiRequest struct {
	ClientRequest
	headerConnectionProperties string
	headerClientTags           string
	incomingPort               int32
	transactionId              string
	clientHost                 string
	Query                      *gatewayv1.Query
}

func (QueryApiRequest) isClientRequest() {}
func (r QueryApiRequest) Validate() error {
	tag := "query api"
	if r.Query.GetUsername() == "" {
		return fmt.Errorf("%s: %s", tag, "Missing Trino Username header")
	}
	if r.Query.GetId() == "" {
		return fmt.Errorf("%s: %s", tag, "Missing Query Id")
	}

	// TODO: remove it once transaction support is added
	// Looker's Presto client sends `X-Presto-Transaction-Id: NONE`
	// whereas trino client doesnt send it if its not set
	if !(r.transactionId == "" || r.transactionId == "NONE") {
		return fmt.Errorf("%s: %s", tag, "Transactions are not supported in gateway.")
	}
	return nil
}

type QueryRequest struct {
	ClientRequest
	headerConnectionProperties string
	headerClientTags           string
	headerSource               string
	headerPreparedStatements   []string
	incomingPort               int32
	transactionId              string
	clientHost                 string
	Query                      *gatewayv1.Query
	// smart is the router's decision for this query, nil when the router is
	// disabled or unreachable. Logged with the query id once Trino answers.
	smart *smart.Decision
	// smartUnavailable records that the router failed, so the prediction log
	// still has a row for this query.
	smartUnavailable bool
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
