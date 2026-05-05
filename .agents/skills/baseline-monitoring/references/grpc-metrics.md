# gRPC Metrics

Read this file when the service exposes gRPC methods.

These metric families cover the expected ingress visibility for gRPC services.

The metric names below are the Razorpay baseline names for gRPC ingress coverage.

They are intentional baseline names for this standard, even if some upstream gRPC Prometheus libraries expose differently named defaults such as `grpc_server_started_total`.

## Metric Families

| Metric family | What it should measure | Canonical metric name | Code-owned labels |
|---|---|---|---|
| gRPC request rate | Incoming gRPC request volume | `server_requests_total` | `action` |
| gRPC response time | Method latency distribution | `grpc_server_handling_seconds_bucket` | `grpc_method` |
| gRPC status codes | Breakdown by gRPC status code | `grpc_server_handled_total` | `grpc_service`, `grpc_method`, `grpc_code` |
| gRPC error by action | Error count grouped by action and error type | `server_error_response_total` | `action`, `error` |

## Derived Views

- gRPC error rate should be derived from `grpc_server_handled_total`
- gRPC method usage should be derived from `server_requests_total` or `grpc_server_handled_total`
- gRPC stream metrics, authentication metrics, rate-limit metrics, and payload-size metrics should be emitted when those behaviors exist in the service

## Baseline Expectation

For gRPC services, request rate, response time, error rate, and status-code visibility are baseline. The remaining families should be emitted when those behaviors exist in the service.
