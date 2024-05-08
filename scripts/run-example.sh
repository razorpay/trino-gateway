#!/bin/bash

# Create dummy setup
curl --request POST \
  --url http://localhost:8000/twirp/razorpay.gateway.BackendApi/CreateOrUpdateBackend \
  --header 'Content-Type: application/json' \
  --header 'X-Auth-Key: test123' \
  --data '{
      "hostname": "docker.for.mac.localhost:37000",
      "scheme": "https",
      "id": "dev",
      "external_url": "docker.for.mac.localhost:37000",
      "is_enabled": true,
      "uptime_schedule": "* * * * *",
      "cluster_load": 0,
      "threshold_cluster_load": 0,
      "stats_updated_at": "0"
    }'

curl --request POST \
  --url http://localhost:8000/twirp/razorpay.gateway.GroupApi/CreateOrUpdateGroup \
  --header 'Content-Type: application/json' \
  --header 'X-Auth-Key: test123' \
  --data '{
        "id": "dev",
		"backends": ["dev"],
		"strategy": "RANDOM",
		"last_routed_backend": "dev",
		"is_enabled": true
    }'

curl --request POST \
  --url http://localhost:8000/twirp/razorpay.gateway.PolicyApi/CreateOrUpdatePolicy \
  --header 'Content-Type: application/json' \
  --header 'X-Auth-Key: test123' \
  --data '{
	    "id": "dev",
        "rule": {
            "type": "listening_port",
            "value": "8080"
        },
        "group": "dev",
        "fallback_group": "dev",
        "is_enabled": true,
        "is_auth_delegated": false,
        "set_request_source": "localDev"
    }'


