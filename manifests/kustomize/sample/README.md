# Sample installation

1. Customize your values
- Edit **params.env** to your values
- Edit kustomization.yaml to set your namespace, e.x. "kubeflow"

2. Install

```
kubectl apply -k cluster-scoped-resources/

kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s

kubectl apply -k sample/
# If upper one action got failed, e.x. you used wrong value, try delete, fix and apply again
# kubectl delete -k sample/ # No surprise if saw msg says already deleted.

kubectl wait applications/mypipeline -n mykubeflow --for condition=Ready --timeout=1800s

kubectl describe configmap inverse-proxy-config -n kubeflow | grep googleusercontent.com
```

3. Post-installation configures

It depends on how you create the cluster, 
- if the cluster is created with **--scopes=cloud-platform**, no actions required
- if the cluster is on Workload Identity, please run **gcp-workload-identity-setup.sh**
