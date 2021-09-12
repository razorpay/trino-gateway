# trino-gateway
A load balancer / routing proxy / gateway for Trino Query engine (partitally supports PrestoDB and other forks).

## Info

Golang based microservice which acts as a proxy for Trino servers.
The application has 3 components and they use GRPC for communication.
More details are in this design doc.

## Features supported
Ability to create logical groups of Trino clusters.
Available routing strategies:
- round robin
- least load
- random

Supported parameters for routing policies:
- Incoming socket
- HTTP headers
    - client-tags
    - connection-properties
    - host

Periodic cluster healthcheck via SQL queries

## Not supported
Transactions - Handling transactions is half baked in the app, thus it is disabled and the application will throw an exception (HTTP500) if the client tries to initiate transactions.

Full Proxy - A design decision to only route the query submission requests to the backend and subsequent communication between the server and client happens directly.
This removes and data transfer overhead from gateway but as a tradeoff requires direct network connectivity between clients and servers, which might be undesirable in certain scenarios.


## TODO
Support Transactions (need sticky routing on transaction id)

Proper GUI - scope would be limited but current implementation of using vecty + gopherjs is hard to maintain and deploy.

Tracing integration

## Gateway API
Check the
protobuf file or the swaggerUI



##### Observations
Direct connectivity

NOTE: for curl the syntax is `query` instead of `"query"`  or `{query}` or `{"query"}`


python - y
go - busted -- throws a semantic error on trino side
curl - works

Via gateway
python - works
go - works
curl - works


Prepared statements:
partial - routing works but gateway doesnt save prepared stmt in its db
i.e.
for running `Select 1`
it will send `EXECUTE _trino_go USING 1`
with header `X-Trino-Prepared-Statement: _trino_go=SELECT+1`



-----
????? need to verify, trino doesnt start executing query unless the nextUri is hit first time


------

trino-dev - coincierge - is behind SSO
its connectivity is flaky via concierge from gateway, SSO keeps on redirecting to oauth page, breaking gateway

------------------
Swagger
swaggerui/index.html - contains location of loading swagger json file
to generate swagger to proto mapping
go run github.com/go-bridget/twirp-swagger-gen -in rpc/gateway/service.proto -out swaggerui/service.swagger.json -host "localhost:8000"

protoc --go_out=. --twirp_out=. ./rpc/gateway/service.proto

##### Reasonings for reverse proxy flow

elazarl/goproxy
is a forward proxy

coreos/goproxy
is fork of previous one with reverse proxy support BUT

requests sent by all trino clients + normal curl requests
have
`http.Request.URL.Host` & `http.Request.URL.Scheme` set as nil
so goproxy cant work
https://github.com/coreos/goproxy/blob/f8dc2d7ba04e38fa53c1a09be5fa92c62db6af62/proxy.go#L113



solution:
inherit its type & override request URL

-- not sure how to do it in go

OR

fork it and remove this restriction

-- need to check how its built

OR
implement own basic reverse Proxy handler

-- lot of handling around request parsing, json marshaling unmarshaling etc.




## Setup ENV
Golang
Twirp
https://twitchtv.github.io/twirp/docs/install.html

gopherJs


Setup a MySQL DB instance


## Build
go mod download
protoc --go_out=. --twirp_out=. ./rpc/gateway/service.proto
protoc --go_out=. --twirp_swagger_out=./swaggerui/ ./rpc/gateway/service.proto

protoc -I ./vendor/github.com/grpc-ecosystem/grpc-gateway/v2 -I . --openapiv2_out ./third_party/swaggerui --openapiv2_opt logtostderr=true rpc/gateway/service.proto


<!-- go run github.com/go-bridget/twirp-swagger-gen -in ./rpc/gateway/service.proto -out ./third_party/swaggerui/service.swagger.json -host localhost:8000 -->


## Dev
make build
go run ./cmd/migration up
App uses uber/zap for logging with everything in single line json, use `jq` or similar cli json parsers to prettify the logs.
go run ./cmd/gateway | jq

### Notes for gopherjs
on changing go versions
rerun
go mod vendor

Not everything can be compiled
https://github.com/gopherjs/gopherjs/issues/889
gopherjs build ./cmd/frontend -o ./web/frontend/frontend.js

It is necessary to hav a main package

as of gopherjs 1.17.1,
darwin arm64 is not fully supported for building as well as running
use a linux based container for building in this env


### Restriction
Policy can't be created unless atleast one group is created, <- this is to satisfy fallback group criteria


-----for UI testing
INSERT INTO `queries` (`id`,`created_at`,`updated_at`,`text`,`client_ip`,`group_id`,`backend_id`,`username`,`received_at`,`submitted_at`) VALUES ('20210930_051350_05544_n3mb3',1632978830,1632978830,'SELECT \'1\', \'abcd\'','','g1','b1','utk',1632978830,1632978830)


-----for meta query testing
"CALL system.runtime.kill_query(query_id => '20211215_184517_19053_wvv79', message => 'Looker query cancel.')"




## Sanity Testing
Create all backends
Create all groups
Create all policies


## Transaction support
Currently transactions are not supported
`X-Trino-Transaction-Id` should be empty or request will be rejected
this will require persisting transaction info in a separate table.
