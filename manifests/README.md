# TL;DR
If you want to skip any customization, you can deploy Kubeflow Pipelines by running
```
export PIPELINE_VERSION=0.1.20
kubectl create -f https://raw.githubusercontent.com/kubeflow/pipelines/$PIPELINE_VERSION/manifests/namespaced-install.yaml
```

You might lack the permission to create role and command might partially fail. If so, bind your account as cluster admin.
(Or role creator in your namespace)
```
kubectl create clusterrolebinding your-binding --clusterrole=cluster-admin --user=[your-user-name]
```

When deployment is done, the UI is accessible by port-forwarding
```
kubectl port-forward -n kubeflow svc/ml-pipeline-ui 8080:80
```

# Customization
Customization can be done through Kustomize Overlay, and don't need to modify the base directory. 

## Change deploy namespace
This directory contains the Kustomize Manifest for deploying Kubeflow Pipelines. 
Kustomize allows you to easily customize your deployment.

To deploy Kubeflow Pipelines in namespace FOO
- Edit [kustomization.yaml](namespaced-install/kustomization.yaml) namespace section to FOO
- Then run 
```
kubectl kustomize . | kubectl apply -f -
```

## Reinstall with existing data
TODO

## Expose a IAM controlled public endpoint
By default, the deployment doesn't expose any public endpoint. 
If you don't want to port-forward every time to access UI, you could install an [invert proxy agent](https://github.com/google/inverting-proxy) that exposes a public endpoint.
To install, uncomment the proxy component in the [kustomization.yaml](base/kustomization.yaml).

When deployment is done, you can find the endpoint by describing
```
kubectl describe configmap inverse-proxy-config -n kubeflow
```
and check the Hostname section. The endpoint should have format like **1234567-dot-datalab-vm-us-west1.googleusercontent.com**


# Uninstall
You can uninstall everything by running
```
kubectl delete -f https://raw.githubusercontent.com/kubeflow/pipelines/$PIPELINE_VERSION/manifests/namespaced-install.yaml
```

Or if you deploy using kustomize
```
kubectl kustomize . | kubectl delete -f -
```
