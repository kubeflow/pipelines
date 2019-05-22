
This folder contains Kubeflow Pipelines Kustomize manifests for a light weight deployment. You can follow the instruction and deploy Kubeflow Pipelines in an existing cluster.


# TL;DR

If you want to skip any customization, you can deploy Kubeflow Pipelines by running
```
export PIPELINE_VERSION=4eeeb6e22432ece32c7d0efbd8307c15bfa9b6d3
kubectl apply -f https://raw.githubusercontent.com/kubeflow/pipelines/$PIPELINE_VERSION/manifests/namespaced-install.yaml
```

You might lack the permission to create role and command might partially fail. If so, bind your account as cluster admin and rerun the same command.
(Or role creator in your namespace)
```
kubectl create clusterrolebinding your-binding --clusterrole=cluster-admin --user=[your-user-name]
```

When deployment is done, the UI is accessible by port-forwarding
```
kubectl port-forward -n kubeflow svc/ml-pipeline-ui 8080:80
```
and open http://localhost:8080/

# Customization
Customization can be done through Kustomize [Overlay](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/glossary.md#overlay). You don't need to modify the base directory. 

Note - The instruction below assume you installed kubectl v1.14.0 or later, which has native support of kustomize.
To get latest kubectl, visit [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

## Change deploy namespace
To deploy Kubeflow Pipelines in namespace FOO
- Edit [kustomization.yaml](namespaced-install/kustomization.yaml) namespace section to FOO
- Then run 
```
kubectl kustomize . | kubectl apply -f -
```

## Reinstall with existing data
TODO

## Expose a IAM controlled public endpoint
By default, the deployment doesn't expose public endpoint.
If you don't want to port-forward every time to access UI, you could install an [invert proxy agent](https://github.com/google/inverting-proxy) that exposes a public URL.
To install, uncomment the proxy component in the [kustomization.yaml](base/kustomization.yaml).

When deployment is complete, you can find the endpoint by describing
```
kubectl describe configmap inverse-proxy-config -n kubeflow
```
and check the Hostname section. The endpoint should have format like **1234567-dot-datalab-vm-us-west1.googleusercontent.com**


# Uninstall
You can uninstall Kubeflow Pipelines by running
```
kubectl delete -f https://raw.githubusercontent.com/kubeflow/pipelines/$PIPELINE_VERSION/manifests/namespaced-install.yaml
```

Or if you deploy through kustomize
```
kubectl kustomize . | kubectl delete -f -
```
