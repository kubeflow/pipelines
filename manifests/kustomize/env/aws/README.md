### Deploy on AWS (with S3 buckets as artifact store)

[https://github.com/e2fyi/kubeflow-aws](https://github.com/e2fyi/kubeflow-aws/tree/master/pipelines)
provides a community-maintained manifest for deploying kubeflow pipelines on AWS
(with S3 as artifact store instead of minio).

TL;DR
```bash
# create aws secret
kubectl -n kubeflow create secret generic ml-pipeline-aws-secret \
    --from-literal=accesskey=$AWS_ACCESS_KEY_ID \
    --from-literal=secretkey=$AWS_SECRET_ACCESS_KEY

# generate yaml
kubectl kustomize https://github.com/e2fyi/kubeflow-aws/pipelines/overlay/accesskey > pipelines-aws.yaml

# apply
kubectl apply -f pipelines-aws.yaml

# uninstall
kubectl delete -f pipelines-aws.yaml
```

For more details, please visit the [repo](https://github.com/e2fyi/kubeflow-aws/tree/master/pipelines).