# Ephemeral Cache Templates

Templates for provisioning an ephemeral Redis cache for a devstack deployment.

## values.yaml additions

```yaml
ephemeral_cache: true

cache:
  namespace: <service-name>
  type: redis
  requests_cpu: 50m
  requests_memory: 50Mi
  dns_policy: ClusterFirst
  node_selector: node.kubernetes.io/worker-database-graviton
  version: "6.0"
```

---

## cache-configmap.yaml

```yaml
{{- if .Values.ephemeral_cache }}
apiVersion: v1
kind: ConfigMap
data:
  app.yaml: |
    name: redis-{{ .Values.devstack_label }}
    type: {{ .Values.cache.type }}
    imageTag: {{ .Values.cache.version | quote }}
    namespace: {{ .Values.cache.namespace }}
    ttl: {{ .Values.ttl }}
    requestsCpu: {{ .Values.cache.requests_cpu }}
    requestsMemory: {{ .Values.cache.requests_memory }}
    dnsPolicy: {{ .Values.cache.dns_policy }}
    nodeSelector: {{ .Values.cache.node_selector }}
metadata:
  labels:
    app: cc-{{ .Values.name }}-{{ .Values.devstack_label }}
  name: cc-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "2"
  namespace: cache-configurator
{{- end }}
```

---

## cache-configurator.yaml

```yaml
{{- if .Values.ephemeral_cache }}
apiVersion: batch/v1
kind: Job
metadata:
  name: cc-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "3"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: cache-configurator
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      labels:
        name: cc
    spec:
      containers:
        - image: 'c.rzp.io/razorpay/kube-manifests:cc'
          imagePullPolicy: Always
          name: cc
          resources:
            limits:
              cpu: 100m
              memory: 200Mi
            requests:
              cpu: 50m
              memory: 100Mi
          volumeMounts:
            - name: config-volume
              mountPath: /src/config
      imagePullSecrets:
        - name: registry
      nodeSelector:
        node.kubernetes.io/worker-configurators: ''
      volumes:
        - name: config-volume
          configMap:
            name: cc-{{ .Values.name }}-{{ .Values.devstack_label }}
      restartPolicy: Never
{{- end }}
```

---

## Connection Details

| Field | Value |
|---|---|
| Host | `redis-<devstack-label>.<namespace>.svc.cluster.local` |
| Port | `6379` |

## Hook Execution Order

| Hook | Weight | Resource |
|---|---|---|
| pre-install,pre-upgrade | 2 | cache-configmap.yaml |
| pre-install,pre-upgrade | 3 | cache-configurator.yaml |

> Also requires [secret-management.md](secret-management.md) — secret-cloner and sec-updater must be included to inject the Redis host into the label-specific secret.
