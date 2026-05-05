# Ephemeral SQS Templates

Templates for provisioning ephemeral SQS queues on localstack for a devstack deployment.

## values.yaml additions

> **Both flags must be set** — `ephemeral_sqs` activates secret-cloner/sec-updater; `configurator.sqs` activates the queue provisioner job.

```yaml
ephemeral_sqs: true

configurator:
  sqs: true

queues:
  - name: devstack-<service-name>-jobs
    secretKey: JOBS_QUEUE_URL
  - name: devstack-<service-name>-tasks
    secretKey: TASKS_QUEUE_URL
```

---

## sqs-configmap.yaml

```yaml
{{- if .Values.configurator.sqs }}
apiVersion: v1
kind: ConfigMap
data:
  app.yaml: |
    queue:
      {{- range $idx, $queue := $.Values.queues }}
      q{{ add $idx 1 }}:
        name: {{ $queue.name }}-{{ $.Values.devstack_label }}
        secretKey: {{ $queue.secretKey }}
      {{- end }}
    updateSecret: true
    kubeSecret: {{ .Values.name }}-{{ .Values.devstack_label }}
    namespace: {{ .Values.namespace }}
    provider: localstack
    enableEndpointPrefix: false
metadata:
  labels:
    app: sqs-{{ .Values.name }}-{{ $.Values.devstack_label }}
  name: sqs-{{ .Values.name }}-{{ $.Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "2"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: sqs-configurator
{{- end }}
```

---

## sqs-configurator.yaml

```yaml
{{- if .Values.configurator.sqs }}
apiVersion: batch/v1
kind: Job
metadata:
  name: sqs-{{ .Values.name }}-{{ .Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "3"
    janitor/ttl: "{{ .Values.ttl }}"
  namespace: sqs-configurator
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 300
  template:
    metadata:
      labels:
        name: irc
    spec:
      containers:
        - image: 'c.rzp.io/razorpay/kube-manifests:sqsc'
          imagePullPolicy: IfNotPresent
          name: irc
          resources:
            limits:
              cpu: 100m
              memory: 100Mi
            requests:
              cpu: 100m
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
            name: sqs-{{ .Values.name }}-{{ .Values.devstack_label }}
      restartPolicy: Never
{{- end }}
```

---

## Connection Details

| Field | Value |
|---|---|
| Queue URL | `http://localstack.localstack.svc.cluster.local:4566/000000000000/<queue-name>-<devstack-label>` |
| Queue URLs injected via | `secretKey` field in values.yaml queues config |

## Hook Execution Order

| Hook | Weight | Resource |
|---|---|---|
| pre-install,pre-upgrade | 1 | secret-cloner.yaml |
| pre-install,pre-upgrade | 2 | sqs-configmap.yaml |
| pre-install,pre-upgrade | 3 | sqs-configurator.yaml |
| pre-install,pre-upgrade | 4 | sec-updater-cm.yaml |
| pre-install,pre-upgrade | 5 | sec-updater.yaml |

> Requires [secret-management.md](secret-management.md) — secret-cloner and sec-updater inject the localstack AWS credentials (`AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`) into the label-specific secret.
