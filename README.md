# Trino-Gateway

[![Open in Dev Containers](https://img.shields.io/static/v1?label=Dev%20Containers&message=Open&color=blue&logo=visualstudiocode)](https://vscode.dev/redirect?url=vscode://ms-vscode-remote.remote-containers/cloneInVolume?url=https://github.com/razorpay/trino-gateway)

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

#### VS Code Dev Containers

If you already have VS Code and Docker installed, you can click the badge above or [here](https://vscode.dev/redirect?url=vscode://ms-vscode-remote.remote-containers/cloneInVolume?url=https://github.com/razorpay/trino-gateway) to get started. Clicking these links will cause VS Code to automatically install the Dev Containers extension if needed, clone the source code into a container volume, and spin up a dev container for use.

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

## References :sparkles:

--------------------
[Razorpay](https://engineering.razorpay.com/how-trino-and-alluxio-power-analytics-at-razorpay-803d3386daaf)

{{Add your org here}}

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


# RZP-Trino REST Client

## Overview

This module exposes an API to interact with Trino. It retrieves data using the Trino Go client, processes it, and returns it in JSON format to the client.

### Features:
- **Trino Integration:** Connects to a Trino instance and executes SQL queries.
- **Data Processing:** Applies business logic to the data before sending it to the client.
- **JSON API:** Communicates with clients using JSON format.
- **Request/Response Limitation:** Throws an error if response data exceeds 5,000 records.
- **Configurable:** Uses TOML files for different environments (development, staging, production).

## Base URL

The base URL for the API is: `http://rzp-trino-client:8000`

### 1. `GET /v1/health`

**Description:**  
Checks the health of the trino client API service.

**Response:**

```json
{
  "status": "string",  // Service health status
}
```

### 2. `POST /v1/query`

**Description:**  
Executes a SQL query against Trino.

**Request:**

```json
{
  "sql": "string"  // SQL query to be executed on trino
}
```

**Reponse:**

```json
{
  "status": "string",  // "Success/ Error/ Running"
  "columns": [
    {
      "name": "string",  // Column name
      "type": "string"   // Column type
    }
  ],
  "data": [
    [
      {
        "data": "string"  // Data for each cell
      }
    ]
  ],
  "error": {
    "message": "string",   // Error message (if any)
    "errorCode": "integer",// Error code (if any)
    "errorName": "string", // Error name (if any)
    "errorType": "string"  // Error type (if any)
  }
} 
```

**Reponse(Success example):**

```json
{
    "status": "Success",
    "columns": [
        {
            "name": "nationkey",
            "type": "BIGINT"
        },
        {
            "name": "name",
            "type": "VARCHAR"
        },
        {
            "name": "regionkey",
            "type": "BIGINT"
        },
        {
            "name": "comment",
            "type": "VARCHAR"
        }
    ],
    "data": [
        [
            {
                "data": 0
            },
            {
                "data": "ALGERIA"
            },
            {
                "data": 0
            },
            {
                "data": " haggle. carefully final deposits detect slyly agai"
            }
        ],
        [
            {
                "data": 1
            },
            {
                "data": "ARGENTINA"
            },
            {
                "data": 1
            },
            {
                "data": "al foxes promise slyly according to the regular accounts. bold requests alon"
            }
        ],
        [
            {
                "data": 2
            },
            {
                "data": "BRAZIL"
            },
            {
                "data": 1
            },
            {
                "data": "y alongside of the pending deposits. carefully special packages are about the ironic forges. slyly special "
            }
        ],
        [
            {
                "data": 3
            },
            {
                "data": "CANADA"
            },
            {
                "data": 1
            },
            {
                "data": "eas hang ironic, silent packages. slyly regular packages are furiously over the tithes. fluffily bold"
            }
        ],
        [
            {
                "data": 4
            },
            {
                "data": "EGYPT"
            },
            {
                "data": 4
            },
            {
                "data": "y above the carefully unusual theodolites. final dugouts are quickly across the furiously regular d"
            }
        ],
        [
            {
                "data": 5
            },
            {
                "data": "ETHIOPIA"
            },
            {
                "data": 0
            },
            {
                "data": "ven packages wake quickly. regu"
            }
        ],
        [
            {
                "data": 6
            },
            {
                "data": "FRANCE"
            },
            {
                "data": 3
            },
            {
                "data": "refully final requests. regular, ironi"
            }
        ],
        [
            {
                "data": 7
            },
            {
                "data": "GERMANY"
            },
            {
                "data": 3
            },
            {
                "data": "l platelets. regular accounts x-ray: unusual, regular acco"
            }
        ],
        [
            {
                "data": 8
            },
            {
                "data": "INDIA"
            },
            {
                "data": 2
            },
            {
                "data": "ss excuses cajole slyly across the packages. deposits print aroun"
            }
        ],
        [
            {
                "data": 9
            },
            {
                "data": "INDONESIA"
            },
            {
                "data": 2
            },
            {
                "data": " slyly express asymptotes. regular deposits haggle slyly. carefully ironic hockey players sleep blithely. carefull"
            }
        ]
    ]
}
```

**Reponse(error example):**

```json
{
    "status": "Error",
    "error": {
        "message": "Unable to query trino: trino: query failed (200 OK): \"USER_ERROR: line 1:31: mismatched input '10'. Expecting: ',', '.', 'AS', 'CROSS', 'EXCEPT', 'FETCH', 'FOR', 'FULL', 'GROUP', 'HAVING', 'INNER', 'INTERSECT', 'JOIN', 'LEFT', 'LIMIT', 'MATCH_RECOGNIZE', 'NATURAL', 'OFFSET', 'ORDER', 'RIGHT', 'TABLESAMPLE', 'UNION', 'WHERE', 'WINDOW', <EOF>, <identifier>\"",
        "errorCode": 500,
        "errorName": "",
        "errorType": ""
    }
}
```
## Project Structure

```plaintext
trino-api/
│
├── build/
│   ├── docker/
│       ├── dev/                   
│       │   ├── Dockerfile.api       
│       ├── entrypoint.sh  
│
├── cmd/
│   ├── api/
│       ├── main.go                  
│
├── config/
│   ├── default.toml                
│   ├── dev_docker.toml              
│   ├── stage.toml                   
│   ├── prod.toml                    
│
├── deployment/
│   ├── dev/
│       ├── docker-compose.yml       
│
├── internal/
│   ├── app/
│   │   ├── handler/
│   │   │    ├── handler.go          
│   │   │    ├── handler_test.go 
│   │   ├── process/
│   │   │    ├── processor.go        
│   │   ├── routes/
│   │   │    ├── routes.go           
│   │   ├── app.go                   
│   │
│   ├── config/
│   │   ├── config.go                
│   ├── model/
│   │   ├── model.go                 
│   ├── utils/
│   │   ├── response.go              
│   ├── services/
│   │   ├── trino/
│   │        ├── client.go           
│   └── boot/
│       ├── boot.go                  
│
├── pkg/
│   ├── config/
│       ├── config.go                
│       ├── config_test.go 
│
├── go.mod
│
├── go.sum
│
├── Makefile
│
├── README.md
│
