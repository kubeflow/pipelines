## Namespaced Installation

You can follow this guidance to have multiple Kubeflow pipeline installations in your cluster.

Beside multi-user Kubeflow Pipeline, there's another way to support multi-tenancy case. 
You can deploy KFP for every user/team in the shared cluster. Everyone will have their own KFP control plane.

Another thing user should be aware is this manifest doesn't contain KFP `cache` component
which sets up mutating webhook for execution cache. Since `MutatingWebhookConfiguration` is cluster scope resource,
`cache` component requires `ClusterRole` which is hard to limit all KFP in given namespace.
See [#4781](https://github.com/kubeflow/pipelines/issues/4781) for more details.


```
cd ${PIPELINE_PROJECT}/manifests/kustomize/env/namespaced-installation

1. Install cluster scoped resources
kubectl apply -k crd

2. Install KFP to one namespace
# Update `namespace` in env/namespaced-installation/kustomization.yaml
kustomize edit set namespace alice
kubectl apply -k .

3. Install KFP to a different namespace
# Update `namespace` in env/namespaced-installation/kustomization.yaml
kustomize edit set namespace bob
kubectl apply -k .
```
