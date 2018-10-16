This is a temporary directory to host kubeflow deployment code including pipeline component. 

The code is forked from Kubeflow v0.3.0 release

Requirements:
- ksonnet version 0.11.0 or later.
- Kubernetes 1.8 or later
- kubectl

Instruction
1. Set environment variables
```
export KFAPP=[kf-app]
export PIPELINE_REPO=[pipeline-repo]
export KUBEFLOW_REPO=${PIPELINE_REPO}/kubeflow
```

2. Run the following commands to set up and deploy kubeflow:
```
cd ${KUBEFLOW_REPO} && ${KUBEFLOW_REPO}/scripts/kfctl.sh init ${KFAPP} --platform none
cd ${KFAPP} && ${KUBEFLOW_REPO}/scripts/kfctl.sh generate k8s
cd ${KFAPP} && ${KUBEFLOW_REPO}/scripts/kfctl.sh apply k8s
```

3. Expose endpoint
```
export NAMESPACE=kubeflow
kubectl port-forward -n ${NAMESPACE}  `kubectl get pods -n ${NAMESPACE} --selector=service=ambassador -o jsonpath='{.items[0].metadata.name}'` 8080:80
http://localhost:8080/
```



Reference:
- [Kubeflow quick start](https://www.kubeflow.org/docs/started/getting-started/)
- [Access Kubeflow UI](https://www.kubeflow.org/docs/guides/accessing-uis/#using-kubectl-and-port-forwarding)
