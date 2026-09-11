# MLflow Plugin Configuration

## KFP MLflow Integration

Kubeflow Pipelines supports an [MLflow](https://mlflow.org/docs/latest/ml/tracking/) plugin that enables automatic experiment tracking.
When enabled, the plugin registers each KFP run with MLflow, allowing users to view and analyze experiments and their pipeline runs in the MLflow UI.

## Prerequisites

* Administrative access to a Kubernetes cluster with KFP installed
* Kubectl configured to access your cluster
* An active MLflow deployment. Learn more about [deploying MLflow](https://mlflow.org/docs/latest/self-hosting/).

### MLflow Authentication

The MLflow plugin supports the following authentication methods:
- **kubernetes**: Kubernetes-native authentication using the [Kubeflow/MLflow Integration](https://github.com/kubeflow/mlflow-integration#mlflow-kubeflow-integration) extension. This method enforces request authorization via Kubernetes RBAC and enables workspace-based multi-tenancy.
- **basic-auth**: HTTP Basic Authentication with username and password credentials
- **bearer**: Bearer token authentication
- **none**: No authentication

For **basic-auth** and **bearer** authentication, credentials are provided via a Kubernetes secret. See [MLflow Plugin Configuration in Multi-User Mode](#mlflow-plugin-configuration-in-multi-user-mode) for an example.

## Security

The KFP MLflow plugin supports TLS to secure communication with the MLflow server. When using HTTPS endpoints, configure the following TLS settings in your [API server configuration](#configuring-the-kfp-mlflow-plugin):

- **caBundlePath**: Path to the CA certificate bundle used to verify the MLflow server's certificate
- **insecureSkipVerify**: Defaults to `false` to enforce certificate verification

For production deployments, it is strongly recommended to use HTTPS with valid TLS certificates and keep `insecureSkipVerify` set to `false`.

### CA Certificate Configuration

If your MLflow server uses a custom or internal CA certificate, you must configure trust in both the KFP API server and the driver/launcher pods that execute pipeline tasks:

1. Create a ConfigMap containing your CA certificate:

```bash
kubectl create configmap mlflow-ca-cert \
  --from-file=ca.crt=/path/to/your/ca-certificate.crt \
  -n kubeflow
```

2. Mount the ConfigMap in the API server pod by updating the API server deployment to include the volume and volumeMount:

```yaml
volumes:
  - name: mlflow-ca-cert
    configMap:
      name: mlflow-ca-cert
volumeMounts:
  - name: mlflow-ca-cert
    mountPath: /kfp/certs
    readOnly: true
```

3. Set `plugins.mlflow.tls.caBundlePath` to the mounted path (e.g., `/kfp/certs/ca.crt`) in your API server configuration.

4. For driver/launcher pods, ensure the CA certificate is trusted using one of these methods:
   - Use cluster-wide CA injection mechanisms via environment variables (e.g., `CABUNDLE_SECRET_NAME` or `CABUNDLE_CONFIGMAP_NAME`)
   - Mount the same ConfigMap in driver/launcher pods via platform-specific configuration
   - Use a base image that includes your organization's CA certificates

## MLflow Experiments

When the plugin is enabled, pipeline runs are logged to MLflow experiments:

- **Default experiment**: If no experiment is specified, runs are logged to an experiment with the name `"KFP-Default"`.
- **Custom experiments**: Users can specify a custom experiment name when submitting a run. KFP will create the experiment if it doesn't already exist

Note that if a name is specified for an experiment that does not exist, KFP will automatically create the experiment.

## MLflow Workspaces

[MLflow workspaces](https://mlflow.org/docs/latest/self-hosting/workspaces/) provide an optional organizational layer for multi-tenant deployments. When using the **kubernetes** authentication type with the [Kubeflow/MLflow Integration](https://github.com/kubeflow/mlflow-integration#mlflow-kubeflow-integration) extension, workspaces can be automatically mapped to Kubernetes namespaces for consistent multi-tenancy across KFP and MLflow.

**Workspace Configuration:**
- When `authType` is set to `kubernetes`, workspaces are **enabled by default**
- For other authentication types, workspaces are **disabled by default**
- You can explicitly control this behavior by setting `workspacesEnabled` to `true` or `false` in the plugin configuration

## Configuring the KFP MLflow Plugin

To enable the MLflow plugin, add the following configuration to your KFP API server `config.json` file:

```{code-block} json
:force:

{
  "plugins": {
    "mlflow": {
      "endpoint": "<scheme>://<mlflow-service>:<mlflow-port>",
      "timeout": "30s",
      "tls": {
        "insecureSkipVerify": <boolean>,
        "caBundlePath": "<path-to-ca-bundle>"
      },
      "settings": {}
    }
  }
}
```

### Plugin Configuration Values

Replace the placeholder values with your deployment-specific values:

- **endpoint**: The full URL of your MLflow server (e.g., `https://mlflow-service.mlflow.svc.cluster.local:8443`)
- **timeout**: Timeout duration string for MLflow API calls (e.g., `"30s"`, `"1m"`). Default: `"30s"`
- **tls**: TLS configuration for MLflow communication
  - **insecureSkipVerify**: Set to `true` to skip TLS certificate verification (not recommended for production)
  - **caBundlePath**: Path to the CA certificate bundle for your MLflow server. See [CA Certificate Configuration](#ca-certificate-configuration) for setup instructions.
- **settings**: See [Plugin Optional Settings](#plugin-optional-settings) below for a complete list of MLflow plugin settings

### Plugin Optional Settings

| Field               | Description                                                                                                                                                                                                                              |
|---------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| authType            | Authentication method for the MLflow server. Supported: `kubernetes`, `bearer`, `basic-auth`, `none`. Default: `kubernetes`                                                                                                                    |
| credentialSecretRef | When using `bearer` or `basic-auth`, create a Secret named `kfp-mlflow-credentials` in each run namespace. Set `credentialSecretRef.tokenKey` for bearer authentication, or `credentialSecretRef.usernameKey` and `credentialSecretRef.passwordKey` for basic authentication, to the corresponding Secret data keys. The Secret name is fixed and cannot be configured.                                                |
| workspacesEnabled    | Enable [MLflow workspaces](#mlflow-workspaces) for multi-tenancy in organizations. When `authType` is `kubernetes`, this defaults to `true`. Otherwise, it defaults to `false`. |
| defaultExperimentName | Default experiment name when none is specified for a pipeline run. Defaults to `"KFP-Default"`.                                                                                                                                          |
| experimentDescription | Default description for newly created experiments. Defaults to `"Created by Kubeflow Pipelines"`.                                                                                                                                        |
| kfpBaseURL | Base URL used to construct KFP run links stored in MLflow tags.                                                                                                                                                                          |
| kfpRunURLPathTemplate | Optional path template appended to `kfpBaseURL` when constructing run links. The placeholders `{run_id}` and `{namespace}` are replaced for each run.                                                                                    |
| mlflowBaseURL | Base URL for linking to the MLflow UI from pipeline runs.                                                                                                                                                                                |
| mlflowUIPathPrefix | Path prefix for constructing MLflow UI links.                                                                                                                                                                                            |
| injectUserEnvVars | When set to `true`, injects MLflow environment variables into user containers (e.g., `MLFLOW_TRACKING_URI`, `MLFLOW_TRACKING_AUTH`). Enables component code to interact with MLflow directly.                                            |

### MLflow Plugin Configuration in Multi-User Mode

For multi-user mode deployments, you can apply namespace-specific MLflow configuration using a `kfp-launcher` ConfigMap. This allows different namespaces to customize plugin settings. When using `basic-auth` or `bearer` authentication configured at the API server level, this ConfigMap is required to specify the credential secret reference so that the namespace has access to the MLflow auth credentials:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kfp-launcher
  namespace: kubeflow-user-example-com
data:
  defaultPipelineRoot: "minio://mlpipeline/v2/artifacts"
  plugins.mlflow: |
    {
      "settings": {
        "experimentDescription": "Custom experiment description for this namespace",
        "defaultExperimentName": "namespace-specific-experiment",
        "injectUserEnvVars": true,
        "credentialSecretRef": {
          "usernameKey": "username",
          "passwordKey": "password"
        }
      }
    }
```

The example above shows namespace-specific plugin settings. When using `basic-auth` or `bearer` authentication (configured at the API server level), you must create a secret named `kfp-mlflow-credentials` in the same namespace:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kfp-mlflow-credentials
  namespace: kubeflow-user-example-com
stringData:
  username: "<mlflow-username>"
  password: "<mlflow-password>"
```

For bearer token authentication, use:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kfp-mlflow-credentials
  namespace: kubeflow-user-example-com
stringData:
  token: "<bearer-token>"
```

### Applying Configuration Changes

**Note:** This restart is only required for changes to the API server's `config.json`. Changes to namespace-level `kfp-launcher` ConfigMaps take effect automatically without requiring an API server restart.

After updating the API server configuration, restart the KFP API server to apply the changes:

```bash
kubectl rollout restart deployment/ml-pipeline -n kubeflow
```

Verify the configuration by checking the API server logs for any MLflow-related errors:

```bash
kubectl logs -n kubeflow deployment/ml-pipeline | grep -i mlflow
```

If no errors appear, the plugin has been configured. You can confirm it's working by creating a pipeline run and checking that it appears in your MLflow UI.
