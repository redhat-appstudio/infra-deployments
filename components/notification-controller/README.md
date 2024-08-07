---
title: Notification Controller
---

## Notification Controller

This controller sends push pipelineruns results to [AWS SNS service](https://aws.amazon.com/sns/).
It watches for `push pipelineruns`, extract the results from pipelineruns that ended successfully 
and sends them to a topic defined in `AWS SNS`.

Secrets and environment variables are needed to configure the `AWS SNS`

## Notification Controller secrets

| Name | Source | Description |
| -- | -- | -- |
| aws-sns-secret | appsre-stonesoup-vault | Secret containing `aws_access_key_id` and `aws_secret_access_key`

in the format:  
name: `credentials`  
value: 
```
[default]
aws_access_key_id=<AWS_ACCESS_KEY_ID>
aws_secret_access_key=<AWS_SECRET_ACCESS_KEY>
```

This secret will be used to connect to our AWS account and it is mounted in the created pod

## Environment Variables

List of environment variables:

| Name | Description |
| -- | -- |
| NOTIFICATION_REGION | define the AWS region to use
| NOTIFICATION_TOPIC_ARN | the topic arn the messages will be sent to

These environment variables will be used to define the `SNS topic` which the messages will be sent to 
and the `region` of the AWS account.

## Staging/Production deployment

To deploy the `Notification-Controller` in Staging/Production environments we are using `ExternalSecret`
defined in `vault` and mount it to the created pod.
In addition we supply the `NOTIFICATION_REGION` and `NOTIFICATION_TOPIC_ARN` environment variables.

## Development deployment  

By default, the controller will not be deployed in development environment.
However, deploying to development is possible by following these steps:

1. Obtain credentails for AWS.
2. Create SNS TOPIC and obtain its `region` and `topic arn`
3. Update the `NOTIFICATION_TOPIC_ARN` and `NOTIFICATION_REGION` in 
[development deployment patch](../notification-controller/development/topic_region_add.yaml) file
with the values you obtained         
4. Remove the `Notification-Controller` from the [delete-applications.yaml](../../argo-cd-apps/overlays/development/delete-applications.yaml) file
5. Bootstrap the cluster
6. Create a secret in `Notification-Controller` namespace with the AWS credentials you previously obtained, 
following the structure defined in the [Notification Controller secrets](#notification-controller-secrets) 
section.

