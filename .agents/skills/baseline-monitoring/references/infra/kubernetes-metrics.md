# Kubernetes Metrics

Read this file when the task involves Kubernetes runtime health, pod behavior, deployment state, or cluster-level signals.

These metrics are platform-owned and should be loaded only when the service topology or debugging task actually depends on them.

**These metrics do not need to be instrumented in the application. They are auto-inserted by infra.**

## Metric Families

| Metric family | Metric or form | Description |
|---|---|---|
| Pod CPU usage | `container_cpu_usage_seconds_total` | CPU usage by pod |
| Pod memory usage | `container_memory_usage_bytes` | Memory consumption by pod |
| Pod network I/O | `container_network_receive_bytes_total` | Network bytes received or transmitted per pod |
| Restart count | `kube_pod_container_status_restarts_total` | Pod or container restart frequency |
| Pod status | `kube_pod_status_phase` | Pod phase such as Running or Pending |
| Deployment status | `kube_deployment_status_replicas` | Desired versus ready replica state |
| Service endpoints | `kube_endpoint_address_available` | Healthy endpoint availability |
| ConfigMap changes | Audit or config metrics | Configuration drift or change visibility |
| Secret access | RBAC or audit metrics | Secret usage or access visibility |
| Persistent volume usage | `kubelet_volume_stats_used_bytes` | PVC storage consumption |
| Node resource usage | `node_cpu_seconds_total` | CPU and memory usage by node |
| Namespace quotas | `kube_resourcequota` | Quota allocation and usage per namespace |
| HPA status | `kube_horizontalpodautoscaler_status_current_replicas` | Current autoscaler replica state |
| Ingress status | `kube_ingress_status` | Ingress object availability |
| Service mesh metrics | Custom proxy metrics | Istio, Linkerd, or equivalent mesh signals |
| RBAC monitoring | Custom RBAC usage metrics | Role or binding usage visibility |
| Cluster events | `kube_event` | Cluster events such as CrashLoopBackOff or evictions |
