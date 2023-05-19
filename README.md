# Trino-Gateway

Trino-Gateway is a load balancer / routing proxy / gateway primarily for [Trino](https://trino.io/) Query engine, written in Go and uses twirp framework.


## Features

- Horizontally scalable - Each instance of the service is stateless, and a relational database is used for synchronization.

- Cluster Monitoring - Periodic Trino Cluster healthchecks via a combination of SQL healthcheck queries, APIs.

- Logical Grouping - Create multiple logical groups of Trino clusters, available routing strategies:

  - round robin
  - least load
  - random

- Routing policies - Traffic can be routed to logical groups of Trino clusters based on the following parameters:

  - Incoming socket (controlled by deployment infrastructure)
  - HTTP headers (controlled by client)

    - client-tags
    - connection-properties
    - host

- GUI for monitoring queries (EXPERIMENTAL)

- swaggerUI for service administration

- Provides REST APIs, gRPC and swaggerUI for service administration

### Not supported

SQL Transactions - Handling SQL transactions is half-baked in the app. Therefore, it is disabled, and the application will throw an exception (HTTP500) if the client tries to initiate transactions.

Proxy entire query lifecycle - A design decision to only route the query submission requests to the backend, so subsequent communication between the server and client happens directly. This removes data transfer overhead from the gateway but requires direct network connectivitybetween Trino clients and servers, which might be undesirable in certain scenarios.

## Deployment

### Dependencies

#### Storage

The application requires a relational datastore to store the configs and query history.

Databases supported:
- MySQL 8+

  The DB user configured for the application currently needs full access to the Database.

### Deploy as a container

 Refer to [Docker Image configs](build/docker) and setup the deployment container properties accordingly.

### Deploy in a non container environment

Refer to [Docker Image configs](build/docker) and [Build scripts](scripts/setup.sh)

### Setup Application configs

Default app configs are stored [here](config/default.toml).


## Usage

Once the service is up and running head over to `<hostname>:<configured app.port>/admin/swaggerui` default would be `localhost:8000/admin/swaggerui`

Before it can serve traffic, routing policies need to be configured.

- Create atleast one Backend
- Create atleast one Group
- Create Policies as required

All available API endpoints, and more info on their request parameters are available in swaggerUI.
Proto file containing all the API contracts is present [here](rpc/gateway/service.proto)

## Development

### Application Architecture

The application comprises of 3 components and they leverage gRPC for Inter-process communication.

1. gatewayserver - Serves as the admin service and contains the logic for interfacing with the service's storage and business logic for selecting Trino cluster to route an incoming query to.
Uses [twirp](https://github.com/twitchtv/twirp) framework.

2. monitor - Performs periodic healthchecks of the configured `Backends`. Also tracks configured "uptime schedules" of the clusters and disables/enables them accordingly.

3. router - Contains logic to act as a reverse proxy for clients.

### Project structure

The project aims to loosely follow [golang-standards/project-layout](https://github.com/golang-standards/project-layout) structure


/third_party/swaggerui  -  Contains swaggerui distribution [files](https://github.com/swagger-api/swagger-ui/tree/v4.18.3/dist)

### Build Instructions

A container-based development environment can be set up in the project via docker-compose or similar tools with the only prerequisite being docker installation (or similar tools like podman + buildah etc).

Docker based env can be setup by invoking

```bash
make dev-docker-up
```

For more details check the [Makefile](Makefile)

Rest of this section covers non-container based build environment.

#### Setup build env

1. Install [Golang](https://go.dev)(It is recommended that the version matches exactly as defined in go.mod)

2. [Protobuf compiler](https://github.com/protocolbuffers/protobuf/releases)  

3. Install dependencies

Run

```bash
go mod download
make setup
```

4. Setup a Mysql8 instance

#### Build

```bash
make build build-frontend
```

note: building frontend takes time and can be skipped if there are no changes to frontend module.

#### Setup app Config

Make required changes to app config

#### Running the app

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

## Notes

#### Update swaggerUI
1. Extract https://github.com/swagger-api/swagger-ui/tree/<version>/dist -> third_party/swaggerui
2. Modify swagger-initializer.js to point to generated openApi spec

## TODO 

_rough notes_

Add DB query history purge logic.

Add initial setup configs

Integration tests

Fix GORM model regression in later version

Use https://github.com/samber/mo and https://github.com/samber/lo

Support Transactions (need sticky routing on transaction id), need caching layer or else performance is poor.

Setup cache layer for storing query_id -> backend_id mapping

Proper GUI - scope would be limited but current implementation of using vecty + gopherjs is hard to maintain and deploy.

Switch to Server side rendering for frontend eg:
    - https://hotwired.dev/
    - https://github.com/wolfeidau/hotwire-golang-website

Tracing integration

Explore victoriaMetrics go client <https://github.com/VictoriaMetrics/metrics>


Handle routing errors properly instead of returning HTTP500 in all cases, eg: sql transaction request must return HTTP400 instead of HTTP500
