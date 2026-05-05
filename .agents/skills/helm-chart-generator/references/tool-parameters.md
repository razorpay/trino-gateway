# Tool Parameter Reference

Complete parameter tables for all 7 Helm Chart Generator MCP tools.

## Table of Contents
- [generate_helm_chart_files](#generate_helm_chart_files)
- [create_helm_chart_pr](#create_helm_chart_pr)
- [create_pr_from_local_files](#create_pr_from_local_files)
- [onboard_service](#onboard_service)
- [add_component](#add_component)
- [onboard_gateway](#onboard_gateway)
- [monitor_helm_s3_push](#monitor_helm_s3_push)

---

## generate_helm_chart_files

Generate chart files in-memory. Returns file contents without creating a PR.

| Parameter | Type | Default | Required | Description |
|-----------|------|---------|----------|-------------|
| `app_name` | str | — | **YES** | Chart/service name |
| `environment` | str | — | **YES** | Target environment |
| `image` | str | — | **YES** | Full image URL (e.g. `c.rzp.io/razorpay/svc:v1.2`) |
| `port` | int | `8080` | no | Primary container port |
| `ingress_class` | str | `"traefik-concierge"` | no | Ingress class |
| `ingress_host` | str | auto-derived | no | Override ingress hostname |
| `service_account` | str | `None` | no | SA name (creates serviceaccount.yaml) |
| `service_account_annotations` | str (JSON) | `None` | no | SA annotations as JSON object |
| `secret_name` | str | `None` | no | K8s Secret name for envFrom |
| `env_vars` | str (JSON) | `None` | no | Env vars as JSON `{"KEY":"val"}` |
| `namespace` | str | `app_name` | no | K8s namespace |
| `ports` | list[int] | `[port]` | no | Additional ports |
| `min_replicas` | int | `1` | no | HPA min replicas |
| `max_replicas` | int | `3` | no | HPA max replicas |
| `include_hpa` | bool | `false` | no | Generate HPA template |
| `include_ingress` | bool | `true` | no | Generate IngressRoute |
| `include_service` | bool | `true` | no | Generate Service |
| `configmap_data` | str (JSON) | `None` | no | ConfigMap data as JSON `{"KEY":"val"}` |
| `is_cronjob` | bool | `false` | no | Generate CronJob instead of Deployment |
| `cronjob_schedule` | str | `"0 9 * * *"` | no | Cron schedule |
| `prometheus_port` | int | `0` | no | Prometheus scrape port (0=disabled) |
| `skip_namespace` | bool | `false` | no | Skip namespace.yaml |
| `security_context` | str (JSON) | `None` | no | Security context as JSON |

**Returns:** `{"status","message","chart_name","files":{"path":"content",...}}`

**Files generated (web):** Chart.yaml, values.yaml, templates/namespace.yaml, templates/deployment.yaml, templates/svc.yaml, templates/ing-v2.yaml

**Files generated (cronjob):** Chart.yaml, values.yaml, templates/namespace.yaml, templates/cronjob.yaml

**Optional files:** templates/hpa.yaml, templates/serviceaccount.yaml, templates/configmap.yaml

---

## create_helm_chart_pr

Same parameters as `generate_helm_chart_files`, plus:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `chart_name` | str | `None` | Alias for app_name |
| `base_image` | str | `None` | Image repo (combined with image_tag if `image` not set) |
| `image_tag` | str | `"latest"` | Tag (used only when `image` not set) |

**Returns:** `{"status","message","pr_url","branch","files","files_generated"}`

**kube-manifests layout:**
```
templates/{app_name}/Chart.yaml
templates/{app_name}/values.yaml
templates/{app_name}/templates/deployment.yaml  (+ svc, ing-v2, hpa, etc.)
{environment}/{app_name}/values.yaml
```

---

## create_pr_from_local_files

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `chart_name` | str | **YES** | Service name |
| `chart_yaml_content` | str | **YES** | Content of Chart.yaml |
| `values_yaml_content` | str | **YES** | Content of values.yaml |
| `environment` | str | `"stage"` | Target environment |
| `deployment_yaml_content` | str | no | Content of deployment.yaml |
| `service_yaml_content` | str | no | Content of service.yaml |
| `ingressroute_yaml_content` | str | no | Content of ingressroute.yaml |
| `namespace_yaml_content` | str | no | Content of namespace.yaml |
| `serviceaccount_yaml_content` | str | no | Content of serviceaccount.yaml |
| `hpa_yaml_content` | str | no | Content of hpa.yaml |
| `cronjob_yaml_content` | str | no | Content of cronjob.yaml |
| `configmap_yaml_content` | str | no | Content of configmap.yaml |
| `pr_title` | str | no | Override PR title |
| `pr_description` | str | no | Override PR body |

**Returns:** `{"status","message","pr_url","branch_name"}`

---

## onboard_service

| Parameter | Type | Default | Required | Description |
|-----------|------|---------|----------|-------------|
| `service_name` | str | — | **YES** | Chart name |
| `service_type` | str | — | **YES** | `"web"`, `"worker"`, `"cronjob"`, `"mixed"` |
| `namespace` | str | — | **YES** | K8s namespace |
| `image` | str | — | **YES** | Full image URL |
| `container_port` | int | `8080` | no | Primary port |
| `ingress_hosts` | str (JSON) | `{}` | no | `{"stage":"host","prod":"host"}` |
| `workers` | str (JSON) | `[]` | no | Array of WorkerConfig |
| `cronjobs` | str (JSON) | `[]` | no | Array of CronJobConfig |
| `stage_replicas` | int | `1` | no | Stage replica count |
| `prod_replicas` | int | `2` | no | Prod replica count |
| `request_cpu` | str | `"100m"` | no | CPU request |
| `request_memory` | str | `"256Mi"` | no | Memory request |
| `limit_cpu` | str | `"500m"` | no | CPU limit |
| `limit_memory` | str | `"512Mi"` | no | Memory limit |
| `enable_hpa` | bool | `true` | no | Include HPA |
| `enable_pdb` | bool | `false` | no | Include PodDisruptionBudget |
| `pdb_min_available` | str/int | `1` | no | PDB minAvailable |
| `enable_prometheus` | bool | `true` | no | Prometheus annotations |
| `prometheus_port` | int | `8080` | no | Prometheus scrape port |
| `create_namespace` | bool | `true` | no | Generate namespace.yaml |
| `add_to_helmfile` | bool | `false` | no | Patch helmfile.yaml |
| `ingress_class` | str | `"traefik-concierge"` | no | Ingress class |
| `include_ingress` | bool | `true` | no | Include IngressRoute |
| `include_service` | bool | `true` | no | Include Service |
| `env_vars` | str (JSON) | `None` | no | Static env vars |
| `secret_name` | str | `None` | no | K8s Secret for envFrom |
| `service_account` | str | `None` | no | Service account name |
| `configmap_data` | str (JSON) | `None` | no | ConfigMap data |

**WorkerConfig:** `{"name":"...","queue_type":"sqs|kafka","queue_name":"...","stage_replicas":1,"prod_replicas":2,"enable_keda":false}`

**CronJobConfig:** `{"name":"...","schedule":"0 2 * * *","command":["/bin/run"]}`

**Files generated:** templates/{svc}/, stage/{svc}/values.yaml, prod/{svc}/values.yaml, helmfile/charts/{svc}/values.yaml

---

## add_component

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service_name` | str | **YES** | Existing service name |
| `component_type` | str | **YES** | `"worker"`, `"cronjob"`, `"ingress"`, `"hpa"`, `"pdb"`, `"keda"`, `"configmap"` |
| `worker` | str (JSON) | for `worker` | WorkerConfig JSON |
| `cronjob` | str (JSON) | for `cronjob` | CronJobConfig JSON |
| `ingress_host` | str | for `ingress` | New ingress hostname |
| `ingress_env` | str | no | `"stage"`, `"prod"`, or `"both"` |
| `target_deployment` | str | no | Target deployment for HPA/KEDA |
| `min_replicas` | int | `1` | HPA/KEDA min replicas |
| `max_replicas` | int | `10` | HPA/KEDA max replicas |
| `target_cpu_utilization` | int | `70` | CPU target % |
| `keda_metric_query` | str | `None` | Prometheus query |
| `keda_threshold` | int | `None` | KEDA threshold |
| `configmap_data` | str (JSON) | `None` | ConfigMap data |

---

## onboard_gateway

| Parameter | Type | Default | Required | Description |
|-----------|------|---------|----------|-------------|
| `gateway_name` | str | — | **YES** | e.g. `"axis-v2"`, `"hdfc"` |
| `image` | str | — | **YES** | Full image URL |
| `gateway_type` | str | `"upi"` | no | Only `"upi"` supported |
| `http_port` | int | `8080` | no | HTTP port |
| `grpc_port` | int | `8081` | no | gRPC port |
| `metrics_port` | int | `8082` | no | Metrics port |
| `stage_http_host` | str | auto | no | Stage HTTP hostname |
| `stage_grpc_host` | str | auto | no | Stage gRPC hostname |
| `prod_http_host` | str | auto | no | Prod HTTP hostname |
| `prod_grpc_host` | str | auto | no | Prod gRPC hostname |
| `create_dark` | bool | `true` | no | Generate dark deployment |
| `create_canary` | bool | `false` | no | Generate canary manifests |
| `add_to_helmfile` | bool | `true` | no | Patch helmfile.yaml |
| `stage_replicas` | int | `1` | no | Stage replicas |
| `prod_replicas` | int | `2` | no | Prod replicas |
| `dark_replicas` | int | `1` | no | Dark replicas |
| `request_cpu` | str | `"100m"` | no | CPU request |
| `request_memory` | str | `"256Mi"` | no | Memory request |
| `limit_memory` | str | `"512Mi"` | no | Memory limit |

**Auto-derived hostnames:** `integrations-upi-{name}.int.stage.razorpay.in`, `integrations-upi-{name}.razorpay.in`

**Service name pattern:** Always `integrations-upi-{gateway_name}`

---

## monitor_helm_s3_push

| Parameter | Type | Default | Required | Description |
|-----------|------|---------|----------|-------------|
| `branch` | str | `None` | one of these | Branch name |
| `pr_url` | str | `None` | one of these | Full PR URL |
| `chart_name` | str | `None` | no | Helps filter S3 URLs |
| `timeout_minutes` | int | `15` | no | Max wait time |
| `poll_interval_seconds` | int | `20` | no | Polling interval |

**Returns:** `{"success":true,"helm_chart_s3_url":"s3://...","branch","chart_name","workflow_run_url","duration"}`
