# Ephemeral SNS Templates

Templates for provisioning ephemeral SNS topics on localstack for a devstack deployment.

## values.yaml additions

> **Both flags must be set** — `ephemeral_sns` activates secret-cloner/sec-updater; `configurator.sns` activates the topic provisioner job.

```yaml
ephemeral_sns: true

configurator:
  sns: true

topics:
  - prefix: devstack-<service-name>-event-processed
    secret_name: SNS_TOPICS_EVENT_PROCESSED_NAME
    subscriptions:
      - devstack-consumer-service-process-event
      - devstack-analytics-service-track-event
  - prefix: devstack-<service-name>-notification-sent
    secret_name: SNS_TOPICS_NOTIFICATION_SENT_NAME
    subscriptions:
      - devstack-audit-service-log-notification
```

> If enabling both SQS and SNS together, set all four flags and combine `queues` + `topics`:
> ```yaml
> ephemeral_sqs: true
> ephemeral_sns: true
> configurator:
>   sqs: true
>   sns: true
> ```

---

## sns-configmap.yaml

```yaml
{{- if .Values.configurator.sns }}
apiVersion: v1
kind: ConfigMap
data:
  app.yaml: |
    provider: localstack
    debug: true
    deleteExistingTopic: true
    multipleTopics:
      {{- range $pidx, $topic := $.Values.topics }}
      - name: {{ $topic.prefix }}-{{ $.Values.devstack_label }}
        subscriptions:
          {{- range $sidx, $subscription := $topic.subscriptions }}
          - protocol: sqs
            endpoint: http://localstack.localstack.svc.cluster.local:4566/000000000000/{{ $subscription }}-{{ $.Values.devstack_label }}
            dlqEndpoint: arn:aws:sqs:ap-south-1:000000000000:{{ $subscription }}-base
          {{- end }}
      {{- end }}
metadata:
  labels:
    app: sns-{{ .Values.name }}-{{ $.Values.devstack_label }}
  name: sns-{{ .Values.name }}-{{ $.Values.devstack_label }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "2"
  namespace: sns-configurator
{{- end }}
```

---

## sns-configurator.yaml

```yaml
{{- if .Values.configurator.sns }}
apiVersion: batch/v1
kind: Job
metadata:
  name: sns-{{ .Values.name }}-{{ $.Values.devstack_label }}
  annotations:
    "helm.sh/hook": post-install,post-upgrade
    "helm.sh/hook-weight": "4"
    janitor/ttl: "{{ $.Values.ttl }}"
  namespace: sns-configurator
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 0
  template:
    metadata:
      labels:
        name: sns-{{ .Values.name }}-{{ $.Values.devstack_label }}
    spec:
      containers:
        - image: 'c.rzp.io/razorpay/kube-manifests:snsc'
          imagePullPolicy: IfNotPresent
          name: snsc
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
            name: sns-{{ .Values.name }}-{{ $.Values.devstack_label }}
      restartPolicy: Never
{{- end }}
```

---

## Connection Details

| Field | Value |
|---|---|
| Topic ARN | `arn:aws:sns:ap-south-1:000000000000:<topic-prefix>-<devstack-label>` |
| Subscription endpoint | `http://localstack.localstack.svc.cluster.local:4566/000000000000/<subscription-queue>-<devstack-label>` |
| Topic ARNs injected via | `secret_name` field in values.yaml topics config |

## Hook Execution Order

| Hook | Weight | Resource |
|---|---|---|
| pre-install,pre-upgrade | 1 | secret-cloner.yaml |
| pre-install,pre-upgrade | 2 | sns-configmap.yaml |
| pre-install,pre-upgrade | 2 | sqs-configmap.yaml (if SQS also enabled — must run before SNS) |
| pre-install,pre-upgrade | 3 | sqs-configurator.yaml (SQS queues must exist before SNS subscribes) |
| post-install,post-upgrade | 4 | sns-configurator.yaml |
| pre-install,pre-upgrade | 4 | sec-updater-cm.yaml |
| pre-install,pre-upgrade | 5 | sec-updater.yaml |

> Requires [secret-management.md](secret-management.md) — secret-cloner and sec-updater inject localstack AWS credentials (`AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`) into the label-specific secret.
>
> If SNS subscribes to SQS queues, ensure SQS is set up first (include [ephemeral-sqs.md](ephemeral-sqs.md) templates).
