
# Name
Component: Data processing by creating a cluster in Cloud Dataproc


# Label
Cloud Dataproc, Kubeflow

# Facets
<!--Make sure the asset has data for the following facets:
Use case
Technique
Input data type
ML workflow

The data must map to the acceptable values for these facets, as documented on the “taxonomy” sheet of go/aihub-facets
https://gitlab.aihub-content-external.com/aihubbot/kfp-components/commit/fe387ab46181b5d4c7425dcb8032cb43e70411c1
--->
Use case:
Other

Technique: 
Other

Input data type:
Tabular

ML workflow: 
Data preparation

# Summary
A Kubeflow pipeline component to create a cluster in Cloud Dataproc.

# Details
## Intended use

Use this component at the start of a Kubeflow pipeline to create a temporary Cloud Dataproc cluster to run Cloud Dataproc jobs as steps in the pipeline.

## Runtime arguments

| Argument | Description | Optional | Data type | Accepted values | Default |
|----------|-------------|----------|-----------|-----------------|---------|
| project_id | The Google Cloud Platform (GCP) project ID that the cluster belongs to. | No | GCPProjectID |  |  |
| region | The Cloud Dataproc region to create the cluster in. | No | GCPRegion |  |  |
| name | The name of the cluster. Cluster names within a project must be unique. You can reuse the names of deleted clusters. | Yes | String |  | None |
| name_prefix | The prefix of the cluster name. | Yes | String |  | None |
| initialization_actions | A list of Cloud Storage URIs identifying the executables on each node after the configuration is completed. By default, executables are run on the master and all the worker nodes. | Yes | List |  | None |
| config_bucket | The Cloud Storage bucket to use to stage the job dependencies, the configuration files, and the job driver console’s output. | Yes | GCSPath |  | None |
| image_version | The version of the software inside the cluster. | Yes | String |  | None |
| cluster | The full [cluster configuration](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters#Cluster). | Yes | Dict |  | None |
| wait_interval | The number of seconds to pause before polling the operation. | Yes | Integer |  | 30 |

## Output
Name | Description | Type
:--- | :---------- | :---
cluster_name | The name of the cluster. | String

Note: You can recycle the cluster by using the [Dataproc delete cluster component](https://github.com/kubeflow/pipelines/tree/master/components/gcp/dataproc/delete_cluster).


## Cautions & requirements

To use the component, you  must:
*   Set up the GCP project by following these [steps](https://cloud.google.com/dataproc/docs/guides/setup-project).
*   The component can authenticate to GCP. Refer to [Authenticating Pipelines to GCP](https://www.kubeflow.org/docs/gke/authentication-pipelines/) for details.
*   Grant the following types of access to the Kubeflow user service account:
    *   Read access to the Cloud Storage buckets which contain the initialization action files.
    *   The role, `roles/dataproc.editor`, on the project.

## Detailed description

This component creates a new Dataproc cluster by using the [Dataproc create cluster REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters/create). 

Follow these steps to use the component in a pipeline:

1.  Install the Kubeflow pipeline's SDK

    ```python
    %%capture --no-stderr

    !pip3 install kfp --upgrade
    ```

2. Load the component using the Kubeflow pipeline's SDK


    ```python
    import kfp.components as comp

    dataproc_create_cluster_op = comp.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/1.7.0-rc.3/components/gcp/dataproc/create_cluster/component.yaml')
    help(dataproc_create_cluster_op)
    ```

### Sample
The following sample code works in an IPython notebook or directly in Python code. See the sample code below to learn how to execute the template.

#### Set sample parameters

```python
# Required parameters
PROJECT_ID = '<Put your project ID here>'

# Optional parameters
EXPERIMENT_NAME = 'Dataproc - Create Cluster'
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import json
@dsl.pipeline(
    name='Dataproc create cluster pipeline',
    description='Dataproc create cluster pipeline'
)
def dataproc_create_cluster_pipeline(
    project_id = PROJECT_ID, 
    region = 'us-central1', 
    name='', 
    name_prefix='',
    initialization_actions='', 
    config_bucket='', 
    image_version='', 
    cluster='', 
    wait_interval='30'
):
    dataproc_create_cluster_op(
        project_id=project_id, 
        region=region, 
        name=name, 
        name_prefix=name_prefix, 
        initialization_actions=initialization_actions, 
        config_bucket=config_bucket, 
        image_version=image_version, 
        cluster=cluster, 
        wait_interval=wait_interval)
```

#### Compile the pipeline


```python
#Compile the pipeline
pipeline_func = dataproc_create_cluster_pipeline
pipeline_filename = pipeline_func.__name__ + '.zip'
import kfp.compiler as compiler
compiler.Compiler().compile(pipeline_func, pipeline_filename)
```

#### Submit the pipeline for execution


```python
#Specify values for the pipeline's arguments
arguments = {}

#Get or create an experiment
import kfp
client = kfp.Client()
experiment = client.create_experiment(EXPERIMENT_NAME)

#Submit a pipeline run
run_name = pipeline_func.__name__ + ' run'
run_result = client.run_pipeline(experiment.id, run_name, pipeline_filename, arguments)
```

## References
*   [Kubernetes Engine for Kubeflow](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts)
*   [Component Python code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/dataproc/_create_cluster.py)
*   [Component Docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
*   [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataproc/create_cluster/sample.ipynb)
*   [Dataproc create cluster REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters/create)

## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
