package router

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"time"

	"github.com/razorpay/trino-gateway/internal/boot"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

const LOG_TAG string = "GATEWAY_ROUTER"

type gatewayRouter struct {
	policyClient  gatewayv1.PolicyApi
	backendClient gatewayv1.BackendApi
	groupClient   gatewayv1.GroupApi
	queryClient   gatewayv1.QueryApi
	ctx           *context.Context
}

type clientRequest struct {
	username                   string
	host                       string
	headerConnectionProperties string
	headerClientTags           string
	queryId                    string
	incomingPort               int32
	queryText                  string
	clientIp                   string
}

func StartRoutingServer(ctx *context.Context, port int) *http.Server {

	router := gatewayRouter{
		groupClient:   gatewayv1.NewGroupApiProtobufClient(fmt.Sprint("http://localhost:", boot.Config.App.Port), &http.Client{}),
		policyClient:  gatewayv1.NewPolicyApiProtobufClient(fmt.Sprint("http://localhost:", boot.Config.App.Port), &http.Client{}),
		backendClient: gatewayv1.NewBackendApiProtobufClient(fmt.Sprint("http://localhost:", boot.Config.App.Port), &http.Client{}),
		queryClient:   gatewayv1.NewQueryApiProtobufClient(fmt.Sprint("http://localhost:", boot.Config.App.Port), &http.Client{}),
		ctx:           ctx,
	}

	reverseProxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			_, err := router.processRequest(req, port)
			if err != nil {
				log.Println(err.Error())
				req.URL.Host = "http://invalid:8080"
			}

		},
		Transport: nil,
		ErrorHandler: func(resp http.ResponseWriter, req *http.Request, err error) {
			resp.Write([]byte("Backend unavailable"))
		},
		ModifyResponse: func(resp *http.Response) error {
			err := router.processResponse(resp)
			if err != nil {
				log.Println(err.Error())
			}
			return err
		},
	}

	return &http.Server{
		Handler: &reverseProxy,
	}
}

func constructQueryFromReq(body string, preparedStmt string) string {
	// TODO
	return body
}

func parseClientRequest(r *http.Request, listeningPort int) *clientRequest {

	var body string
	// Assumption that HTTP spec is followed and body in GET is meaningless
	if r.Method == "GET" {
		body = ""
	} else {
		body = parseBody(r.Body)
	}
	query := constructQueryFromReq(body, r.Header.Get("X-Trino-Prepared-Statement"))

	return &clientRequest{
		username:                   r.Header.Get("X-Trino-User"),
		queryId:                    extractQueryId(query),
		incomingPort:               int32(listeningPort),
		host:                       r.Host,
		headerConnectionProperties: r.Header.Get("X-Trino-Connection-Properties"),
		headerClientTags:           r.Header.Get("X-Trino-Client-Tags"),
		queryText:                  query,
	}
}

func (b *gatewayRouter) processRequest(r *http.Request, listeningPort int) (*http.Request, error) {
	var err error = nil
	debugPrintReq(r)
	println("processing Request")
	println(listeningPort)

	clientRequest := parseClientRequest(r, listeningPort)
	querySaveReq := gatewayv1.Query{
		Id:         clientRequest.queryId,
		Text:       clientRequest.queryText,
		Username:   clientRequest.username,
		ClientIp:   clientRequest.clientIp,
		ReceivedAt: time.Now().Unix(),
	}

	var backend *gatewayv1.Backend
	backendFound := func() {
		r.URL.Host = backend.Hostname
		r.URL.Scheme = backend.Scheme.Enum().String()
		r.Host = backend.Hostname
		debugPrintReq(r)

		querySaveReq.BackendId = backend.Id
	}
	if clientRequest.queryId == "" {

		req := gatewayv1.EvaluateGroupRequest{
			IncomingPort:               clientRequest.incomingPort,
			Host:                       clientRequest.host,
			HeaderConnectionProperties: clientRequest.headerConnectionProperties,
			HeaderClientTags:           clientRequest.headerClientTags,
		}
		group, err := b.policyClient.EvaluateGroupForClient(*b.ctx, &req)
		if err != nil {
			err = errors.New(fmt.Sprint("Group Unresolvable for client id:", req, err.Error()))
		} else {
			querySaveReq.GroupId = group.Id
			backend, err = b.groupClient.EvaluateBackendForGroup(*b.ctx, &gatewayv1.EvaluateBackendRequest{
				GroupId: group.Id,
			})
			if err != nil {
				err = errors.New(fmt.Sprint("Backend Unresolvable for query id:", clientRequest.queryId, err.Error()))
			}
			backendFound()
		}
	} else {
		client := gatewayv1.NewQueryApiProtobufClient(fmt.Sprint("http://localhost:", boot.Config.App.Port), &http.Client{})

		req := gatewayv1.FindBackendForQueryRequest{QueryId: clientRequest.queryId}

		backend, err = client.FindBackendForQuery(*b.ctx, &req)
		if err != nil {
			err = errors.New(fmt.Sprint("Backend Unresolvable for query id:", clientRequest.queryId, err.Error()))
		} else {
			backendFound()
		}
	}

	_, err2 := b.queryClient.CreateOrUpdateQuery(*b.ctx, &querySaveReq)
	if err2 != nil {
		boot.Logger(*b.ctx).Errorw(LOG_TAG, map[string]interface{}{"msg": "Unable to save query", "query_id": querySaveReq.Id, "error": err.Error()})
	}

	return r, err
}

func (b *gatewayRouter) processResponse(resp *http.Response) error {
	// Handle Redirects
	// TODO: Clean it up
	regex := regexp.MustCompile(`\w+\:\/\/[^\/]*(.*)`)
	if resp.Header.Get("Location") != ("") {
		oldLoc := resp.Header.Get("Location")
		newLoc := fmt.Sprint("http://", boot.Config.App.ServiceExternalHostname, regex.ReplaceAllString(oldLoc, "$1"))
		resp.Header.Set("Location", newLoc)
	}

	go func() {
		isQuerySubmissionSuccessful := true
		if isQuerySubmissionSuccessful {
			queryId := extractQueryIdFromServerResponse(parseBody(resp.Body))
			req := gatewayv1.Query{
				// TODO
				Id:          queryId,
				SubmittedAt: time.Now().Unix(),
			}

			_, err := b.queryClient.CreateOrUpdateQuery(*b.ctx, &req)
			if err != nil {
				boot.Logger(*b.ctx).Errorw(LOG_TAG, map[string]interface{}{"msg": "Unable to save successfully submitted query", "query_id": req.Id, "error": err.Error()})
			}
		}
	}()

	return nil
}

func extractQueryIdFromServerResponse(body string) string {
	return ""
}

func debugPrintReq(r *http.Request) {
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Println(err)
	}
	log.Println(string(requestDump))
}

func parseBody(body io.ReadCloser) string {
	bodyBytes, _ := io.ReadAll(body)
	// since its a ReadCloser type, the stream will be empty after its read once
	// ensure a it is restored in original request
	body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes)
}

func extractQueryId(body string) string {
	// TODO
	return ""
}
