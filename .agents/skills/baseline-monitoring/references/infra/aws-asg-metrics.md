# AWS ASG Metrics

Read this file when the service depends on AWS Auto Scaling Groups and compute-capacity health is part of the standard observability picture.

## Metric Families

| Metric family | Metric | Description |
|---|---|---|
| In-service instances | `aws_asg_group_in_service_instances` | Number of healthy instances currently serving traffic |
| Desired capacity | `aws_asg_group_desired_capacity` | Target number of instances the ASG wants to keep running |
| Min size | `aws_asg_group_min_size` | Lower bound for ASG capacity |
| Max size | `aws_asg_group_max_size` | Upper bound for ASG capacity |
| Pending instances | `aws_asg_group_pending_instances` | Instances still launching or not yet ready |
| Standby instances | `aws_asg_group_standby_instances` | Instances kept out of active service |
| Terminating instances | `aws_asg_group_terminating_instances` | Instances currently being removed |
| Total instances | `aws_asg_group_total_instances` | Total fleet size managed by the ASG |

## Baseline Expectation

When ASGs are part of the service path, instance health, desired-versus-actual capacity, and scaling state should be visible in the standard infra view.
