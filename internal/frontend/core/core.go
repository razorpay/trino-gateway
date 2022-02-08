package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

type GatewayApiClient struct {
	Policy  gatewayv1.PolicyApi
	Backend gatewayv1.BackendApi
	Group   gatewayv1.GroupApi
	Query   gatewayv1.QueryApi
}

type Core struct {
	gatewayApiClient *GatewayApiClient
}

type ICore interface {
	GetQueries(count int, skip int, user string) ([]*gatewayv1.Query, error)
}

func NewCore(gatewayHost string) *Core {
	return &Core{
		gatewayApiClient: &GatewayApiClient{
			Backend: gatewayv1.NewBackendApiProtobufClient(gatewayHost, &http.Client{}),
			Group:   gatewayv1.NewGroupApiProtobufClient(gatewayHost, &http.Client{}),
			Policy:  gatewayv1.NewPolicyApiProtobufClient(gatewayHost, &http.Client{}),
			Query:   gatewayv1.NewQueryApiProtobufClient(gatewayHost, &http.Client{}),
		},
	}
}

func (c *Core) GetQueries(count int, skip int, user string) ([]*gatewayv1.Query, error) {
	req := gatewayv1.QueriesListRequest{
		Skip:     int32(skip),
		Count:    int32(count),
		Username: user,
	}
	queriesResp, err := c.gatewayApiClient.Query.ListQueries(context.Background(), &req)
	if err != nil {
		println(err.Error())
		return nil, errors.New(fmt.Sprint("Unable to Fetch list of queries", err.Error()))
	}

	return queriesResp.GetItems(), nil
}
