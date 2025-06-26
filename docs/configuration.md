# Configuration (WIP)

This document describes the configuration options available for Kubeflow Pipelines. This is a work in progress. For a
more complete view of configuration options, look at the default `backend/src/apiserver/config/config.json`
configuration file.

## Configuration File

Kubeflow Pipelines uses a JSON configuration file that can be specified when starting the API server. The default
configuration file is located at `backend/src/apiserver/config/config.json`.

### Setting the Configuration File Path

When launching the Kubeflow Pipelines API server, you can specify a custom configuration file path using the `--config`
flag:

```bash
# Use default config file
/bin/apiserver --config backend/src/apiserver/config/config.json

# Use custom config file
/bin/apiserver --config /path/to/your/custom-config.json
```

### Environment Variable Configuration

Configuration values can also be set using environment variables. The environment variable names follow Viper
conventions, where dots in the JSON configuration keys are replaced with underscores and the entire key is converted to
uppercase.

For example:

- `ArgoWorkflowsConfig.SemaphoreConfigMapName` can be set via the `ARGOWORKFLOWSCONFIG_SEMAPHORECONFIGMAPNAME`
  environment variable.

```bash
# Set semaphore ConfigMap name via environment variable
export ARGOWORKFLOWSCONFIG_SEMAPHORECONFIGMAPNAME="my-custom-semaphore-configmap"
/bin/apiserver --config backend/src/apiserver/config/config.json
```

Environment variables take precedence over values in the configuration file.

## Argo Workflows Configuration

Configuration specific to Argo Workflows integration is defined in the `ArgoWorkflowsConfig` section.

```json
{
  "ArgoWorkflowsConfig": {
    "SemaphoreConfigMapName": "kfp-argo-workflows-semaphore"
  }
}
```

### Options

- **SemaphoreConfigMapName** - Configures the name of the ConfigMap used for
  [semaphore synchronization](https://argo-workflows.readthedocs.io/en/latest/synchronization/) in Argo Workflows. This
  defaults to `kfp-argo-workflows-semaphore`.

## Related Documentation

- [Pipeline Configuration DSL](https://github.com/kubeflow/pipelines/blob/master/api/v2alpha1/pipeline_spec.proto#L1125) -
  For pipeline-level configuration options including semaphore_key
- [Argo Workflows Semaphores](https://argo-workflows.readthedocs.io/en/latest/synchronization/) - For information about
  semaphore synchronization
