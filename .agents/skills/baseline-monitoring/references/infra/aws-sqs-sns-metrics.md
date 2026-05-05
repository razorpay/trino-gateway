# AWS SQS and SNS Metrics

Read this file when the service depends on SNS or SQS and those queueing or notification paths need observability coverage.

## SNS Metric Families

| Metric family | Metric | Description |
|---|---|---|
| Published messages | `aws_sns_number_of_messages_published` | Number of messages published to SNS |
| Delivered notifications | `aws_sns_number_of_notifications_delivered` | Number of notifications delivered by SNS |

## SQS Metric Families

| Metric family | Metric | Description |
|---|---|---|
| Messages sent | `aws_sqs_number_of_messages_sent` | Number of messages sent to the queue |
| Messages received | `aws_sqs_number_of_messages_received` | Number of messages received from the queue |
| Messages deleted | `aws_sqs_number_of_messages_deleted` | Number of messages deleted after processing |
| Approximate age of oldest message | `aws_sqs_approximate_age_of_oldest_message` | Queue backlog aging signal |
