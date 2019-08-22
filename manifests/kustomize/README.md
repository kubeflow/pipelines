This folder contains Kubeflow Pipelines Kustomize manifests for a light weight deployment. You can follow the instruction and deploy Kubeflow Pipelines in an existing cluster.


# TL;DR

If you want to skip any customization, you can deploy Kubeflow Pipelines by running
```
export PIPELINE_VERSION=0.1.26
kubectl apply -f https://raw.githubusercontent.com/kubeflow/pipelines/$PIPELINE_VERSION/manifests/kustomize/namespaced-install.yaml
```

You might lack the permission to create role and command might partially fail. If so, bind your account as cluster admin and rerun the same command.
(Or role creator in your namespace)
```
kubectl create clusterrolebinding your-binding --clusterrole=cluster-admin --user=[your-user-name]
```

When deployment is complete, you can access Kubeflow Pipelines UI by an IAM controlled public endpoint, which can be found by
```
kubectl describe configmap inverse-proxy-config -n kubeflow
```
and check the Hostname section. The endpoint should have format like **1234567-dot-datalab-vm-us-west1.googleusercontent.com**

# Customization
Customization can be done through Kustomize [Overlay](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/glossary.md#overlay). 

Note - The instruction below assume you installed kubectl v1.14.0 or later, which has native support of kustomize.
To get latest kubectl, visit [here](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

## Change deploy namespace
To deploy Kubeflow Pipelines in namespace FOO,
- Edit [kustomization.yaml](env/dev/kustomization.yaml) namespace section to FOO
- Then run 
```
kubectl kustomize env/dev | kubectl apply -f -
```

## Disable the public endpoint
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



# Uninstall
You can uninstall Kubeflow Pipelines by running
```
kubectl delete -f https://raw.githubusercontent.com/kubeflow/pipelines/$PIPELINE_VERSION/manifests/kustomize/namespaced-install.yaml
```

Or if you deploy through kustomize
```
kubectl kustomize env/dev | kubectl delete -f -
```
# FAQ
If sample code requires a "user-gcp-sa" secret, you could create one by 
- First download the GCE VM service account token following this [instruction](https://cloud.google.com/kubernetes-engine/docs/tutorials/authenticating-to-cloud-platform#step_3_create_service_account_credentials)
- Run
```
kubectl create secret -n [your-namespace] generic user-gcp-sa --from-file=user-gcp-sa.json=[your-token-file].json
```
