# Trino-Gateway

A load balancer / routing proxy / gateway for Trino Query engine 
(can support PrestoDB and other forks who follow same client protocol).

## Info

Golang based service which acts as a proxy for Trino servers.
The application comprises 3 components and they leverage gRPC for Inter process communication.

More details are in the design doc.

## Features

Horizontally scalable - Each instance of the service is stateless and a relational database is used for synchronization.

Cluster Monitoring - Periodic Trino Cluster healthchecks via a combination of SQL healthcheck queries, APIs.

Logical Grouping - Create multiple logical groups of Trino clusters, following routing strategies are available

- round robin
- least load
- random

Routing policies - Traffic can be routed to logical groups of Trino clusters based on following parameters:

- Incoming socket (controlled by deployment infrastructure)
- HTTP headers (controlled by client)

  - client-tags
  - connection-properties
  - host

GUI for monitoring queries (WIP)

swaggerUI for administering the service

Provides both REST APIs and gRPC for administration interface

## Not supported

Transactions - Handling transactions is half baked in the app, thus it is disabled and the application will throw an exception (HTTP500) if the client tries to initiate transactions.

Proxy entire query lifecycle - A design decision to only route the query submission requests to the backend, so subsequent communication between the server and client happens directly.
This removes data transfer overhead from gateway but as a tradeoff requires direct network connectivity between Trino clients and servers, which might be undesirable in certain scenarios.

## Gateway API

Check the protobuf file or the swaggerUI

## Prerequisites

Mysql Database
(Postgres is not tested, but should also work, but will require db bootstrap commands to be implemented from MySQL -> Postgres)

## Project structure

The project aims to loosely follow <golang/project/structure repo url> structure

## Development

A container based development environment can be setup in the project via docker-compose with only prerequisite being docker installation (or similar tools like podman + buildah etc).

Docker based env can be setup by invoking

```bash
make dev-docker-up
```

For more details check the `Makefile`

Rest of this section covers non-container based build environment.

### Setup build env

[Golang]()(It is recommended that the version matches exactly as defined in go.mod)
[Protobuf]()  

Run

```bash
go mod download
make setup
```

Mysql8 instance

### Build

```bash
make build build-frontend
```

note: building frontend takes time and can be skipped if there are no changes to frontend module.

### Setup app Config

Make required changes to app config

### Running the app

For bootstrapping the DB schemas

```bash
go run ./cmd/migration up
```

More available options can be found by running

```bash
go run ./cmd/migration
```

Once the DB schema is initialized, run the app

App uses uber/zap for logging with everything in single line json, use `jq` or similar cli json parsers to prettify the logs.
```bash
go run ./cmd/gateway | jq
```

## TODO

Integration tests

Support Transactions (need sticky routing on transaction id)

Proper GUI - scope would be limited but current implementation of using vecty + gopherjs is hard to maintain and deploy.

Tracing integration

Explore victoriaMetrics go client <https://github.com/VictoriaMetrics/metrics>

----

# Notes

## Observations

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

----

????? need to verify, trino doesnt start executing query unless the nextUri is hit first time

----

trino-dev - coincierge - is behind SSO
its connectivity is flaky via concierge from gateway, SSO keeps on redirecting to oauth page, breaking gateway

----

Swagger
swaggerui/index.html - contains location of loading swagger json file
to generate swagger to proto mapping
go run github.com/go-bridget/twirp-swagger-gen -in rpc/gateway/service.proto -out swaggerui/service.swagger.json -host "localhost:8000"

protoc --go_out=. --twirp_out=. ./rpc/gateway/service.proto

## Reasonings for reverse proxy flow

elazarl/goproxy
is a forward proxy

coreos/goproxy
is fork of previous one with reverse proxy support BUT

requests sent by all trino clients + normal curl requests
have
`http.Request.URL.Host` & `http.Request.URL.Scheme` set as nil
so goproxy cant work
<https://github.com/coreos/goproxy/blob/f8dc2d7ba04e38fa53c1a09be5fa92c62db6af62/proxy.go#L113>

solution:
inherit its type & override request URL

-- not sure how to do it in go

OR

fork it and remove this restriction

-- need to check how its built

OR
implement own basic reverse Proxy handler

-- lot of handling around request parsing, json marshaling unmarshaling etc.

## Build

go mod download
protoc --go_out=. --twirp_out=. ./rpc/gateway/service.proto
protoc --go_out=. --twirp_swagger_out=./swaggerui/ ./rpc/gateway/service.proto

protoc -I ./vendor/github.com/grpc-ecosystem/grpc-gateway/v2 -I . --openapiv2_out ./third_party/swaggerui --openapiv2_opt logtostderr=true rpc/gateway/service.proto

<!-- go run github.com/go-bridget/twirp-swagger-gen -in ./rpc/gateway/service.proto -out ./third_party/swaggerui/service.swagger.json -host localhost:8000 -->


### Notes for gopherjs

on changing go versions
rerun
go mod vendor

Not everything can be compiled
<https://github.com/gopherjs/gopherjs/issues/889>
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

## Notes for module version upgrades

check release notes of following modules as they are either sensitive to language version. env used or are in alpha state so their APIs can change

vecty
gorm
gopherjs
protoc-gen-validate
