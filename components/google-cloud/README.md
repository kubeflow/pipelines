# Google Cloud Pipeline Components

Google Cloud Pipeline Components provides an SDK and a set of components that lets
you interact with Google Cloud services such as AI Platform (Unified). You can use 
the predefined components in this repository to build your pipeline using the
Kubeflow Pipelines SDK.

## Installation

### Requirements

-   Python >= 3.6
-   [A Google Cloud project with the AI Platform (Unified) API enabled.](https://cloud.google.com/ai-platform-unified/docs/start/cloud-environment)
-   An
    [authenticated GCP account](https://cloud.google.com/ai-platform/docs/getting-started-keras#authenticate_your_gcp_account)


### Install latest release

Use the following command to install Google Cloud Pipeline Components from PyPI.

```shell
pip install -U google-cloud-pipeline-components
```

### Install from source

Use the following commands to install Google Cloud Pipeline Components from GitHub.

```shell
git clone https://github.com/kubeflow/pipelines.git
pip install pipelines/components/google-cloud/.
```

### Build the package from source and install

Use the following commands build Google Cloud Pipeline Components from the source code and install the package.

```shell
source_root=$(pwd)

git clone https://github.com/kubeflow/pipelines.git
cd pipelines/components/google-cloud
python setup.py bdist_wheel clean

WHEEL_FILE=$(find "$source_root/pipelines/components/google-cloud/dist/" -name "google_cloud_pipeline_components*.whl")
pip3 install --upgrade $WHEEL_FILE
```

## Getting started with Google Cloud Pipeline Components

Google Cloud Pipeline Components provides an SDK that makes it easier to use
Google Cloud services in your pipeline. To get started, consider the following example.

```python
from google_cloud_pipeline_components as gcc
from kfp.v2 import dsl

@dsl.pipeline
def my_pipeline(
    dataset_resource_name: str = 'user_resouce_name_uri',
    ucaip_dataset_resource_name: str):

    # Import the dataset at the URI specified in the
    # dataset_resource_name pipeline parameter.
    importer = dsl.importer(artifact_uri=dataset_resource_name,
                            artifact_class=io_types.Dataset, reimport=False)
    
    # Use AutoML to train a regression model using tabular data
    training_step = gcc.aiplatform.AutoMLTabularTrainingJobRunner(
        display_name='train-housing-automl_1',
        optimization_prediction_type='regression',
        dataset = importer.output.outputs['dataset_metadata']
    )

    # Deploy the trained model for predictions
    model_deploy_step = gcc.aiplatform.ModelDeployer(
        model=training_step.outputs['model_metadata'])

```

In this example, the pipeline does the following.

1.  Imports a dataset, such as a CSV, from the URI specified in a
    pipeline parameter.
1.  Uses the dataset to train a regression model using tabular data.
1.  Deploys the model to AI Platform (Unified) for predictions.

## Components 
Following is the list of currently supported components. 
For API documentation please refer to the projects [ReadTheDocs page](https://google-cloud-pipeline-components.readthedocs.io).

```python
AutoMLImageTrainingJobRunOp(...)
    """Runs an AutoML Image training job and returns a model."""

AutoMLTabularTrainingJobRunOp(...)
    """Runs an AutoML Tabular training job and returns a model."""

AutoMLTextTrainingJobRunOp(...)
    """Runs an AutoML Text training job and returns a model."""

AutoMLVideoTrainingJobRunOp(...)
    """Runs an AutoML Image training job and returns a model."""

CustomContainerTrainingJobRunOp(...)
    """Runs a custom container training job."""

CustomPythonPackageTrainingJobRunOp(...)
    """Runs the custom training job."""

EndpointCreateOp(...)
    """Creates a new endpoint."""

ImageDatasetCreateOp(...)
    """Creates a new image dataset and optionally imports data into dataset when"""

ImageDatasetExportDataOp(...)
    """Exports data to output dir to GCS."""

ImageDatasetImportDataOp(...)
    """Upload data to existing managed dataset."""

ModelBatchPredictOp(...)
    """Creates a batch prediction job using this Model and outputs prediction"""

ModelDeployOp(...)
    """Deploys model to endpoint. Endpoint will be created if unspecified."""

ModelUploadOp(...)
    """Uploads a model and returns a Model representing the uploaded Model resource."""

TabularDatasetCreateOp(...)
    """Creates a new tabular dataset."""

TabularDatasetExportDataOp(...)
    """Exports data to output dir to GCS."""

TextDatasetCreateOp(...)
    """Creates a new text dataset and optionally imports data into dataset when"""

TextDatasetExportDataOp(...)
    """Exports Text data to output dir to GCS."""

TextDatasetImportDataOp(...)
    """Upload data to existing managed Text dataset."""

VideoDatasetCreateOp(...)
    """Creates a new video dataset and optionally imports data into dataset when"""

VideoDatasetExportDataOp(...)
    """Exports Video data to output dir to GCS."""

VideoDatasetImportDataOp(...)
    """Upload data to existing managed Video dataset."""
```
