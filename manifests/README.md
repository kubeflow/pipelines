# TL;DR
If you want to skip any customization, you can deploy Kubeflow Pipelines by running
```
export PIPELINE_VERSION=0.1.19
kubectl create -f https://raw.githubusercontent.com/kubeflow/pipelines/$PIPELINE_VERSION/manifests/namespaced-install.yaml
```

You might lack the permission to create role and command might partially fail. If so, bind your account as cluster admin.
(Or role creator in your namespace)
```
kubectl create clusterrolebinding your-binding --clusterrole=cluster-admin --user=[your-user-name]
```

# Customization

## Change deploy namespace
This directory contains the Kustomize Manifest for deploying Kubeflow Pipelines. 
Kustomize allows you to easily customize your deployment.

To deploy Kubeflow Pipelines in namespace FOO
- Edit [kustomization.yaml](namespaced-install/kustomization.yaml) namespace section to your namespace
- Then run 
```
kubectl kustomize . | kubectl create -f -
```

## Reinstall with existing data
TODO


# Uninstall
You can uninstall everything by running
```
kubectl delete -f https://raw.githubusercontent.com/kubeflow/pipelines/$PIPELINE_VERSION/manifests/namespaced-install.yaml
```

Or if you deploy using kustomize
```
kubectl kustomize . | kubectl delete -f -
```
