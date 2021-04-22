# Google Cloud Pipeline Components

Google Cloud Pipeline Components provides an sdk as well as a set of components for users
to interact with Google Cloud services such as AI Platform. You can use any of
the predefined components in this repo to construct your pipeline directly using
KFP DSL.

Additionally we are providing an SDK that allows users to dynamically generate
the components using python. In Preview release only a limited set of components
are available as a predefined component. For a full list of supported components
via the python SDK please refer to [usage guide #4](#usage-guide).

### Installation

#### Requirements

-   Python >= 3.6
-   [A Google Cloud project](https://cloud.google.com/ai-platform/docs/getting-started-keras#set_up_your_project)
-   An
    [authenticated GCP account](https://cloud.google.com/ai-platform/docs/getting-started-keras#authenticate_your_gcp_account)
-   [Google AI platform](https://cloud.google.com/ai-platform/) APIs enabled for
    your GCP account.

#### Install latest release

```shell
pip install -U google-cloud-pipeline-components
```

#### Install from source

```shell
git clone https://github.com/kubeflow/pipelines.git
pip install pipelines/components/google-cloud/.
```

#### Build the package from source and install

```shell
source_root=$(pwd)

git clone https://github.com/kubeflow/pipelines.git
cd pipelines/components/google-cloud
python setup.py bdist_wheel clean

WHEEL_FILE=$(find "$source_root/pipelines/components/google-cloud/dist/" -name "google_cloud_pipeline_components*.whl")
pip3 install --upgrade $WHEEL_FILE
```

### High level overview

Google Cloud Pipeline Components provides the an API for constructing your pipeline. To
start, let's walk through a simple workflow using this API.

1.  Let's begin with a Keras model training code such as the following, saved as
    `mnist_example.py`.

    ```python
    from google_cloud_pipeline_components as gcc
    from kfp.v2 import dsl

    @dsl.pipeline
    def my_pipeline(dataset_resource_name: str = 'user_resouce_name_uri')
    ucaip_dataset_resource_name: str):

    # create/get metadata node;
    importer = dsl.importer(artifact_uri=dataset_resource_name,
                            artifact_class=io_types.Dataset, reimport=False)
    # training pipeline
    training_step = gcc.aiplatform.AutoMLTabularTrainingJobRunner(
        display_name='train-housing-automl_1',
        optimization_prediction_type='regression',
        dataset = importer.output.outputs['dataset_metadata']
    )

    model_deploy_step = gcc.aiplatform.ModelDeployer(
        model=training_step.outputs['model_metadata'])

    #endpoint
    model_deploy_step.outputs['endpoint_metadata']
    ```


    ### Supported Components 
    Following is the list of currently supproted components. 

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
