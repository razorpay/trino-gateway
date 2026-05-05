# Ephemeral Database Templates

Templates for provisioning an ephemeral MySQL/Postgres database for a devstack deployment.

## values.yaml additions

```yaml
ephemeral_db: true

database:
  type: mysql        # or postgres
  name: <service-name>
  namespace: <service-name>
  username: <service-name>
  password: <generated-password>
  requests_cpu: 50m
  requests_memory: 50Mi
  dns_policy: ClusterFirst
  node_selector: node.kubernetes.io/worker-database
  version: ""        # defaults to latest
  attach_volume: false
  volume_size: ""
```

---

## db-configmap.yaml

```yaml
{{- if .Values.ephemeral_db }}
apiVersion: v1
kind: ConfigMap
data:
  app.yaml: |
    name: {{ .Values.database.type }}-{{ .Values.devstack_label }}
    type: {{ .Values.database.type }}
    imageTag: {{ .Values.database.version | quote }}
    namespace: {{ .Values.database.namespace }}
    ttl: {{ .Values.ttl }}
    requestsCpu: {{ .Values.database.requests_cpu }}
    requestsMemory: {{ .Values.database.requests_memory }}
    dnsPolicy: {{ .Values.database.dns_policy }}
    nodeSelector: {{ .Values.database.node_selector }}
    rootPassword: {{ randAlphaNum 12 | lower }}
    attachVolume: {{ .Values.database.attach_volume | default false }}
    volumeSize: {{ .Values.database.volume_size | default "" }}
    databases:
      - dbName: {{ .Values.database.name }}
        username: {{ .Values.database.username }}
        password: {{ .Values.database.password }}
        seeding: false
        snapshotPath: ""
        configKey: db
metadata:
  labels:
    app: dbc-{{ .Values.name }}-{{ .Values.devstack_label }}
  name: dbc-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "2"
  namespace: db-configurator
{{- end }}
```

---

## db-configurator.yaml

```yaml
{{- if .Values.ephemeral_db }}
apiVersion: batch/v1
kind: Job
metadata:
  name: dbc-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "3"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: db-configurator
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 60
  template:
    metadata:
      annotations:
        iam.amazonaws.com/role: dev-serve-api
      labels:
        name: dbc
    spec:
      containers:
        - image: 'c.rzp.io/razorpay/kube-manifests:dbc'
          imagePullPolicy: Always
          name: dbc
          resources:
            limits:
              cpu: 200m
              memory: 500Mi
            requests:
              cpu: 100m
              memory: 150Mi
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
            name: dbc-{{ .Values.name }}-{{ .Values.devstack_label }}
      restartPolicy: Never
{{- end }}
```

---

## Connection Details

| Field | Value |
|---|---|
| Host / DB_URL | `<db-type>-<devstack-label>.<namespace>.svc.cluster.local` |
| Port | `3306` (MySQL) / `5432` (Postgres) |
| Database | as configured in `database.name` |
| Username | as configured in `database.username` |
| Password | as configured in `database.password` |

## Hook Execution Order

| Hook | Weight | Resource |
|---|---|---|
| pre-install,pre-upgrade | 2 | db-configmap.yaml |
| pre-install,pre-upgrade | 3 | db-configurator.yaml |

> Also requires [secret-management.md](secret-management.md) — secret-cloner and sec-updater must be included to inject DB credentials into the label-specific secret.
