package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

func main() {
	vecty.RenderBody(&MyComponent{})
}

type MyComponent struct {
	vecty.Core
}

func (mc *MyComponent) Render() vecty.ComponentOrHTML {
	return elem.Body(
		&MyChildComponent{},
		vecty.Text("some footer text"),
	)
}

type MyChildComponent struct {
	vecty.Core
}

func (mc *MyChildComponent) Render() vecty.ComponentOrHTML {
	queries, err := GetAllQueries()
	queryText := "BAMBOO"
	if err == nil {
		queryText = queries[0].GetText()
	}
	return elem.Div(
		vecty.Markup(vecty.Class("my-main-container")),
		// vecty.Text("Welcome to my site"),
		vecty.Text(queryText),
	)
}

func GetAllQueries() ([]*gatewayv1.Query, error) {
	queryClient := gatewayv1.NewQueryApiProtobufClient(fmt.Sprint("http://localhost:", "8080"), &http.Client{})

	req := gatewayv1.QueriesListRequest{}
	queriesResp, err := queryClient.ListQueries(context.Background(), &req)
	if err != nil {
		println(err.Error())
		return nil, errors.New(fmt.Sprint("Unable to Fetch list of queries", err.Error()))
	}

	queries := queriesResp.GetItems()

	return queries, nil
}
