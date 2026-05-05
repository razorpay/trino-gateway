# Secret Management Templates

Templates for cloning and updating label-specific secrets. Required whenever any ephemeral resource is enabled (`ephemeral_db`, `ephemeral_cache`, `ephemeral_sqs`, or `ephemeral_sns`).

## How it works

1. **secret-cloner** clones the base secret (`<secret_name>`) into a label-specific copy (`<secret_name>-<devstack_label>`)
2. **sec-updater-cm** defines the key/value overrides to apply to that copy
3. **sec-updater** applies those overrides

All three are gated on `secret_cloner_enabled: true` (default) AND at least one ephemeral resource flag being set.

The deployment mounts:
- `<secret_name>` — when `devstack_label == "base"` or `secret_cloner_enabled: false`
- `<secret_name>-<devstack_label>` — otherwise

See [core.md](core.md) deployment.yaml for the conditional secret name.

---

## secret-cloner.yaml

```yaml
{{- if .Values.secret_cloner_enabled }}
apiVersion: batch/v1
kind: Job
metadata:
  name: sec-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "1"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: secret-cloner
spec:
  backoffLimit: 0
  template:
    metadata:
      labels:
        name: sec
    spec:
      containers:
        - env:
            - name: ACTION
              value: clone
            - name: NAMESPACE
              value: '{{ .Values.namespace }}'
            - name: SECRETNAME
              value: '{{ .Values.secret_name }}'
            - name: SECRETSUFFIX
              value: '{{ .Values.devstack_label }}'
          image: 'c.rzp.io/razorpay/kube-manifests:sec'
          imagePullPolicy: IfNotPresent
          name: sec
          resources:
            limits:
              cpu: 50m
              memory: 50Mi
            requests:
              cpu: 50m
              memory: 50Mi
      imagePullSecrets:
        - name: registry
      nodeSelector:
        node.kubernetes.io/worker-configurators: ''
      restartPolicy: OnFailure
{{- end }}
```

---

## sec-updater-cm.yaml

> ⚠️ **Always verify secret key names with the user before applying.**
> The `key:` values below are defaults. Ask the user what environment variable names their application actually reads and update the `key:` fields accordingly.
>
> Example: _"What env vars does `<service-name>` use for DB host, name, username, password, and URL? I'll update the keys to match."_

```yaml
{{- if .Values.secret_cloner_enabled }}
apiVersion: v1
kind: ConfigMap
data:
  app.yaml: |
    updateEntries:
{{- if .Values.ephemeral_db }}
      s1:
        key: DB_HOST
        value: {{ .Values.database.type }}-{{ .Values.devstack_label }}.{{ .Values.database.namespace }}.svc.cluster.local
      s2:
        key: DB_NAME
        value: {{ .Values.database.name }}
      s3:
        key: DB_USERNAME
        value: {{ .Values.database.username }}
      s4:
        key: DB_PASSWORD
        value: {{ .Values.database.password }}
      s5:
        key: DB_URL
        value: {{ .Values.database.type }}-{{ .Values.devstack_label }}.{{ .Values.database.namespace }}.svc.cluster.local
{{- end }}
{{- if or .Values.ephemeral_sqs .Values.ephemeral_sns }}
      aws1:
        key: AWS_REGION
        value: ap-south-1
      aws2:
        key: AWS_ACCESS_KEY_ID
        value: test
      aws3:
        key: AWS_SECRET_ACCESS_KEY
        value: test
{{- end }}
    action: update
    secretName: {{ .Values.secret_name }}-{{ .Values.devstack_label }}
    namespace: {{ .Values.namespace }}
metadata:
  labels:
    app: sec-updater-{{ .Values.name }}-{{ .Values.devstack_label }}
  name: sec-updater-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "4"
  namespace: secret-cloner
{{- end }}
```

---

## sec-updater.yaml

```yaml
{{- if .Values.secret_cloner_enabled }}
apiVersion: batch/v1
kind: Job
metadata:
  name: sec-updater-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "5"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: secret-cloner
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 300
  template:
    metadata:
      labels:
        name: sec-updater
    spec:
      containers:
        - image: 'c.rzp.io/razorpay/kube-manifests:sec'
          imagePullPolicy: Always
          name: sec
          resources:
            limits:
              cpu: 50m
              memory: 50Mi
            requests:
              cpu: 50m
              memory: 50Mi
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
            name: sec-updater-{{ .Values.name }}-{{ .Values.devstack_label }}
      restartPolicy: OnFailure
{{- end }}
```

---

## Hook Execution Order (full sequence)

| Weight | Hook | Resource |
|---|---|---|
| 1 | pre-install,pre-upgrade | secret-cloner.yaml |
| 2 | pre-install,pre-upgrade | db-configmap.yaml / sqs-configmap.yaml / sns-configmap.yaml |
| 3 | pre-install,pre-upgrade | db-configurator.yaml / sqs-configurator.yaml |
| 4 | pre-install,pre-upgrade | sec-updater-cm.yaml |
| 4 | post-install,post-upgrade | sns-configurator.yaml |
| 5 | pre-install,pre-upgrade | sec-updater.yaml |

## Keys injected per resource

| Resource | Keys added to secret |
|---|---|
| ephemeral_db | `DB_HOST`, `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD`, `DB_URL` |
| ephemeral_sqs | `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| ephemeral_sns | `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| ephemeral_cache | Redis host injected via separate sec-updater-cm entry (add as needed) |
