# Multi-user Isolation

Multi-user isolation for Kubeflow Pipelines is part of Kubeflow's overall profile and namespace isolation strategy.

:::{tip}
Kubeflow Pipelines multi-user isolation is only supported in [Kubeflow Community Distribution](https://www.kubeflow.org/docs/started/installing-kubeflow/#kubeflow-community-distribution) deployments.
Refer to [profiles and namespaces](https://www.kubeflow.org/docs/components/central-dash/profiles/) for common operations like [managing profile contributors](https://www.kubeflow.org/docs/components/central-dash/profiles/#manage-profile-contributors).
:::

## How are resources separated?

Kubeflow Pipelines separates resources using Kubernetes namespaces that are managed by [Kubeflow Profiles](https://www.kubeflow.org/docs/components/central-dash/profiles/).
Other users cannot see resources in your Profile/Namespace without permission, because the Kubeflow Pipelines API server
rejects requests for namespaces that the current user is not authorized to access.

"Experiments" belong to namespaces directly, runs and recurring runs belong to their parent experiment's namespace.

"Pipeline Runs" are executed in user namespaces, so that users can leverage Kubernetes namespace isolation.
For example, they can configure different secrets for other services in different namespaces.

:::{warning}
Kubeflow makes no hard security guarantees about Profile isolation.
<br>
User profiles have no additional isolation beyond what is provided by Kubernetes Namespaces.
:::

## When using the UI

When you visit the Kubeflow Pipelines UI from the Kubeflow Dashboard, it only shows "experiments", "runs", and "recurring runs" in your chosen namespace.
Similarly, when you create resources from the UI, they also belong to the namespace you have chosen.

:::{warning}
Pipeline definitions are not isolated right now, and are shared across all namespaces, see [Current Limitations](#current-limitations) for more details.
:::

## When using the SDK

How to connect Pipelines SDK to Kubeflow Pipelines will depend on __what kind__ of Kubeflow deployment you have, and __from where you are running your code__.

* [Full Kubeflow (from inside cluster)](../user-guides/core-functions/connect-api.md#kubeflow-community-distribution---inside-the-cluster)
* [Full Kubeflow (from outside cluster)](../user-guides/core-functions/connect-api.md#kubeflow-community-distribution---outside-the-cluster)
* [Standalone Kubeflow Pipelines (from inside cluster)](../user-guides/core-functions/connect-api.md#standalone-kfp---inside-the-cluster)
* [Standalone Kubeflow Pipelines (from outside cluster)](../user-guides/core-functions/connect-api.md#standalone-kfp---outside-the-cluster)

The following Python code will create an experiment (and associated run) from a Pod inside a full Kubeflow cluster.

```python
import kfp

# the namespace in which you deployed Kubeflow Pipelines
kubeflow_namespace = "kubeflow"

# the namespace of your pipelines user (where the pipeline will be executed)
user_namespace = "jane-doe"

# the KF_PIPELINES_SA_TOKEN_PATH environment variable is used when no `path` is set
# the default KF_PIPELINES_SA_TOKEN_PATH is /var/run/secrets/kubeflow/pipelines/token
credentials = kfp.auth.ServiceAccountTokenVolumeCredentials(path=None)

# create a client
client = kfp.Client(host=f"http://ml-pipeline-ui.{kubeflow_namespace}", credentials=credentials)

# create an experiment
client.create_experiment(name="<YOUR_EXPERIMENT_ID>", namespace=user_namespace)
print(client.list_experiments(namespace=user_namespace))

# create a pipeline run
client.run_pipeline(
    experiment_id="<YOUR_EXPERIMENT_ID>",  # the experiment determines the namespace
    job_name="<YOUR_RUN_NAME>",
    pipeline_id="<YOUR_PIPELINE_ID>"  # the pipeline definition to run
)
print(client.list_runs(experiment_id="<YOUR_EXPERIMENT_ID>"))
print(client.list_runs(namespace=user_namespace))
```

:::{tip}
* To set a default namespace for Pipelines SDK commands, use the [`kfp.Client().set_user_namespace()`](https://kubeflow-pipelines.readthedocs.io/en/stable/source/client.html#kfp.Client.set_user_namespace) method,
  this method stores your user namespace in a configuration file at `$HOME/.config/kfp/context.json`.
* Detailed documentation for `kfp.Client()` can be found in the [Kubeflow Pipelines SDK Reference](https://kubeflow-pipelines.readthedocs.io/en/stable/source/client.html).
:::

## When using the REST API

When calling the [Kubeflow Pipelines REST API](../reference/api/kubeflow-pipeline-api-spec.md), a namespace argument is required for experiment APIs.
<br>
The namespace is specified by a "resource reference" with `type` of `NAMESPACE` and `key.id` equal to the namespace name.

The following code uses the [generated python API client](https://kubeflow-pipelines.readthedocs.io/en/stable/source/kfp.server_api.html) to create an experiment and pipeline run.

```python
import kfp
from kfp_server_api import *

# the namespace in which you deployed Kubeflow Pipelines
kubeflow_namespace = "kubeflow"

# the namespace of your pipelines user (where the pipeline will be executed)
user_namespace = "jane-doe"

# the KF_PIPELINES_SA_TOKEN_PATH environment variable is used when no `path` is set
# the default KF_PIPELINES_SA_TOKEN_PATH is /var/run/secrets/kubeflow/pipelines/token
credentials = kfp.auth.ServiceAccountTokenVolumeCredentials(path=None)

# create a client
client = kfp.Client(host=f"http://ml-pipeline-ui.{kubeflow_namespace}", credentials=credentials)

# create an experiment
experiment: ApiExperiment = client._experiment_api.create_experiment(
    body=ApiExperiment(
        name="<YOUR_EXPERIMENT_ID>",
        resource_references=[
            ApiResourceReference(
                key=ApiResourceKey(
                    id=user_namespace,
                    type=ApiResourceType.NAMESPACE,
                ),
                relationship=ApiRelationship.OWNER,
            )
        ],
    )
)
print("-------- BEGIN: EXPERIMENT --------")
print(experiment)
print("-------- END: EXPERIMENT ----------")

# get the experiment by name (only necessary if you comment out the `create_experiment()` call)
# experiment: ApiExperiment = client.get_experiment(
#     experiment_name="<YOUR_EXPERIMENT_ID>",
#     namespace=user_namespace
# )

# create a pipeline run
run: ApiRunDetail = client._run_api.create_run(
    body=ApiRun(
        name="<YOUR_RUN_NAME>",
        pipeline_spec=ApiPipelineSpec(
            # replace <YOUR_PIPELINE_ID> with the UID of a pipeline definition you have previously uploaded
            pipeline_id="<YOUR_PIPELINE_ID>",
        ),
        resource_references=[ApiResourceReference(
            key=ApiResourceKey(
                id=experiment.id,
                type=ApiResourceType.EXPERIMENT,
            ),
            relationship=ApiRelationship.OWNER,
        )
        ],
    )
)
print("-------- BEGIN: RUN --------")
print(run)
print("-------- END: RUN ----------")

# view the pipeline run
runs: ApiListRunsResponse = client._run_api.list_runs(
    resource_reference_key_type=ApiResourceType.EXPERIMENT,
    resource_reference_key_id=experiment.id,
)
print("-------- BEGIN: RUNS --------")
print(runs)
print("-------- END: RUNS ----------")
```

## Current limitations

### Resources without isolation

The following resources do not currently support isolation and are shared without access control:

* Artifacts, Executions, and other metadata entities in [Machine Learning Metadata (MLMD)](https://www.tensorflow.org/tfx/guide/mlmd).
