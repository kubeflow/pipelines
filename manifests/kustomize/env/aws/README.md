# Installing kubeflow pipelines for AWS infrastructure

This is a guide on installing kubeflow pipelines in a kubernetes cluster
(need not be EKS) hosted on AWS infrastructure (i.e. using EBS, S3, ...).

There are 2 approaches towards allowing kubeflow pipelines services to access
AWS resources:

- AWS access keys (see [here](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys)), or
- IAM (e.g. [kube2iam](https://github.com/jtblin/kube2iam))

The kustomize overlays are available in the respective folders:

- [accesskey](./accesskey): i.e. `kubectl kustomize accesskey`
- [iam](./iam): i.e. `kubectl kustomize iam`

## Quick start

`params.env` in the overlay folders are the parameters that you can set for your
deployment. Most of the defaults for parameters are fine except for:
- `awsIAMRole`: for IAM-based approach, you need to provide the IAM role for the service to assume
- `awsRegion`: your AWS region (this is needed for tensorboard viewer to work properly)

```bash
# generate the provided overlay variant
kubectl kustomize [overlay_folder] > kubeflow-pipelines-aws.yaml
# deploy
kubectl apply -f kubeflow-pipelines-aws.yaml
```

## Notable changes

> #### NOTE:
>
> The folder (defaults to `pipelines`) to save the pipeline templates cannot be
> configured until
> [#2080](https://github.com/kubeflow/pipelines/pull/2080) is merged in.

#### Archiving pod logs
`archiveLogs` is set to `true` so that pod logs are automatically archived into
the configured S3 bucket. `ml-pipeline-ui` will be able to retrieve the pod logs
even if the pod (and node) had been removed and purged.

#### MySQL
For simplicity, a generic mysql 5.6 service is provisioned. This can be replaced
with AWS Aurora or MYSQL RDS if needed (but not tested).

#### Access-key based access
A k8s secret with the AWS credential must be created. This secret will be referenced
by the various kfp services to access the S3 buckets.

```bash
kubectl -n kubeflow create secret generic ml-pipeline-aws-secret \
    --from-literal=accesskey=$AWS_ACCESS_KEY_ID \
    --from-literal=secretkey=$AWS_SECRET_ACCESS_KEY
```

#### IAM based access
An appropriate IAM role must be created for kfp services to access the S3 buckets.

This approach assumes that an IAM credential provisioning service
(e.g. [kube2iam](https://github.com/jtblin/kube2iam)) is deployed in the k8s cluster.

Alternatively, existing IAM role that is assigned to the k8s nodes can be updated to
permit access to the appropriate S3 buckets (not recommended).

