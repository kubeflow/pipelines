## Mock Implementation of Argo Workflow Executor Plugin

According to the [documentation](https://argo-workflows.readthedocs.io/en/latest/executor_plugins/), an Argo Workflow plugin is a sidecar container that runs in the agent pod. The agent pod is created once for each workflow.

### terminology
- kfp-driver-server — the KFP [driver](https://github.com/kubeflow/pipelines/tree/master/backend/src/v2/driver) component extracted from the workflow pod and deployed as a global HTTP/gRPC server.
- driver-plugin - our implementation of the Executor plugin

*One limitation is that the service account token is not mounted into the sidecar container, which means it cannot interact with the Kubernetes API. This is required for drivers.*
As a result, driver-plugin implementations should merely act as a proxy between the global kfp-driver-server and the Argo Workflow controller.

### Prerequisites
- According to the [configuration](https://argo-workflows.readthedocs.io/en/latest/executor_plugins/#configuration) ARGO_EXECUTOR_PLUGINS should be set to true 
- Add additional [workflow RBAC](https://argo-workflows.readthedocs.io/en/latest/http-template/#argo-agent-rbac) for the agent

1. Implement the driver plugin that simply proxies requests from the workflow controller to the kfp-driver-server and back. 
2. Build the image for the driver plugin.
3. Create the [yaml description](src/driver-plugin/plugin.yaml) of the plugin 
4. [Create](https://argo-workflows.readthedocs.io/en/latest/cli/argo_executor-plugin_build/) the configmap by executing  ```argo executor-plugin build .``` in the yaml description folder from the step 3 
5. Apply the created ConfigMap to the workflow-controller Kubernetes namespace.

After that, you will be able to reference the corresponding driver plugin in your Argo Workflow using:
```yaml
plugin:
    driver-plugin:
    ...
```

### Problem: Interaction With the Kubernetes API From a Sidecar Container
The driver [requires](https://github.com/kubeflow/pipelines/blob/master/backend/src/v2/driver/k8s.go#L68) access to the k8s API.
However, the required volume with the service account secret (/var/run/secrets/kubernetes.io/serviceaccount) is mounted only into the main (driver) container, but not into the sidecar container.
Below is a sample YAML snippet showing the container definitions in the agent pod.
```yaml
Containers:
  driver-plugin:
    Image:          .../kfp-driver-agent:2.4.1-63
    Port:           2948/TCP
    Host Port:      0/TCP
    Restart Count:  0
    Limits:
      cpu:     1
      memory:  1Gi
    Requests:
      cpu:     250m
      memory:  512Mi
    Environment:
      DRIVER_HOST:      http://ml-pipeline-kfp-driver.kubeflow.svc
      DRIVER_PORT:      2948
      SERVER_PORT:      2948
      TIMEOUT_SECONDS:  120
    Mounts:
      /etc/gitconfig from gitconfig (ro,path="gitconfig")
      /var/run/argo from var-run-argo (ro,path="driver-plugin")
  main:
    Image:           .../ml-platform/argoexec:v3.6.7
    Command:
      argoexec
    Args:
      agent
      main
      --loglevel
      info
      --log-format
      text
      --gloglevel
      0
    Ready:          True
    Restart Count:  2
    Limits:
      cpu:     100m
      memory:  256M
    Requests:
      cpu:     10m
      memory:  64M
    Environment:
      ARGO_WORKFLOW_NAME:     debug-component-pipeline-7bgps
      ARGO_WORKFLOW_UID:      03caea4e-70c1-4113-b700-b7183271f3b6
      ARGO_AGENT_PATCH_RATE:  10s
      ARGO_PLUGIN_ADDRESSES:  ["http://localhost:2948"]
      ARGO_PLUGIN_NAMES:      ["driver-plugin"]
    Mounts:
      /etc/gitconfig from gitconfig (ro,path="gitconfig")
      /var/run/argo from var-run-argo (rw)
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-z9nt6 (ro)
```
As a workaround, it makes sense to use the agent pod only as a proxy to the kfp-driver-server.