# Install Kubeflow Pipelines
This folder contains Kubeflow Pipelines Kustomize manifests for a light weight deployment. You can follow the instruction and deploy Kubeflow Pipelines in an existing cluster.


## TL;DR

Deploy latest version of Kubeflow Pipelines
```
export PIPELINE_VERSION=0.1.40
kubectl apply -f https://storage.googleapis.com/ml-pipeline/pipeline-lite/$PIPELINE_VERSION/crd.yaml
kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io
kubectl apply -f https://storage.googleapis.com/ml-pipeline/pipeline-lite/$PIPELINE_VERSION/namespaced-install.yaml
```

Then get the Pipeline URL
```
kubectl describe configmap inverse-proxy-config -n kubeflow | grep googleusercontent.com
```

## Customization
Customization can be done through Kustomize [Overlay](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/glossary.md#overlay).

Note - The instruction below assume you installed kubectl v1.14.0 or later, which has native support of kustomize.
To get latest kubectl, visit [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

### Deploy on GCP with CloudSQL and GCS
See [here](env/gcp/README.md) for more details.

### Change deploy namespace
To deploy Kubeflow Pipelines in namespace FOO,
- Edit [dev/kustomization.yaml](env/dev/kustomization.yaml) or [gcp/kustomization.yaml](env/gcp/kustomization.yaml) namespace section to FOO
- Then run
```
kubectl kustomize base/crds | kubectl apply -f -
# then 
kubectl kustomize env/dev | kubectl apply -f -
# or
kubectl kustomize env/gcp | kubectl apply -f -
```

### Disable the public endpoint
By default, the deployment install an [invert proxy agent](https://github.com/google/inverting-proxy) that exposes a public URL. If you want to skip installing it,
- Comment out the proxy component in the [kustomization.yaml](base/kustomization.yaml).
- Then run
```
kubectl kustomize . | kubectl apply -f -
```

The UI is still accessible by port-forwarding
```
kubectl port-forward -n kubeflow svc/ml-pipeline-ui 8080:80
```
and open http://localhost:8080/

### Deploy on AWS (with S3 buckets as artifact store)

[https://github.com/e2fyi/kubeflow-aws](https://github.com/e2fyi/kubeflow-aws/tree/master/pipelines)
provides a community-maintained manifest for deploying kubeflow pipelines on AWS
(with S3 as artifact store instead of minio). More details can be found in the repo.

## Uninstall
You can uninstall Kubeflow Pipelines by running
```
export PIPELINE_VERSION=0.1.38
kubectl delete -f https://storage.googleapis.com/ml-pipeline/pipeline-lite/$PIPELINE_VERSION/namespaced-install.yaml
kubectl delete -f https://storage.googleapis.com/ml-pipeline/pipeline-lite/$PIPELINE_VERSION/crd.yaml
```

Or if you deploy through kustomize
```
kubectl kustomize env/dev | kubectl delete -f -
# or
kubectl kustomize env/gcp | kubectl delete -f -
# then
kubectl kustomize base/crds | kubectl delete -f -

```

## Troubleshooting

### Permission error installing Kubeflow Pipelines to a cluster
Run
```
kubectl create clusterrolebinding your-binding --clusterrole=cluster-admin --user=[your-user-name]
```

### Samples requires "user-gcp-sa" secret
If sample code requires a "user-gcp-sa" secret, you could create one by
- First download the GCE VM service account token [Document](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#creating_service_account_keys)
```
gcloud iam service-accounts keys create application_default_credentials.json \
  --iam-account [SA-NAME]@[PROJECT-ID].iam.gserviceaccount.com
```
- Run
```
kubectl create secret -n [your-namespace] generic user-gcp-sa --from-file=user-gcp-sa.json=application_default_credentials.json
```
