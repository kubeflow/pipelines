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
3. Create the [yaml description](plugin.yaml) of the plugin 
4. [Create](https://argo-workflows.readthedocs.io/en/latest/cli/argo_executor-plugin_build/) the configmap by executing  ```argo executor-plugin build .``` in the yaml description folder from the step 3 
5. Apply the created ConfigMap to the workflow-controller Kubernetes namespace.
6. Create the service account driver-plugin-executor-plugin and set automountServiceAccountToken: true in the sidecar plugin ConfigMap (required for Kubernetes API access; see details below).

After that, you will be able to reference the corresponding driver plugin in your Argo Workflow using:
```yaml
plugin:
    driver-plugin:
    ...
```

### Interaction With the Kubernetes API From a Sidecar Container
The driver [requires](https://github.com/kubeflow/pipelines/blob/master/backend/src/v2/driver/k8s.go#L68) access to the k8s API.
However, by default, the Argo Workflow Controller does not mount the service account token in the executor plugin's sidecar container. Moreover, it [disabled](https://github.com/argoproj/argo-workflows/pull/8028) the ability to mount the Workflow's service account to the executor plugin.
As a result, to enable access to the Kubernetes API:
1. Create ServiceAccount in each profile namespace with the name `driver-plugin-executor-plugin`. Argo WF [expects](https://github.com/argoproj/argo-workflows/blob/main/workflow/controller/agent.go#L285) the format <plugin-name>-executor-plugin
2. Add a Role with appropriate Kubernetes API access and bind it to the service account.
3. Configure `sidecar.automountServiceAccountToken` see [example](plugin.yaml)