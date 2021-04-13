# Google Cloud Components

Google Cloud Components provides an sdk as well as a set of components for users
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
pip install -U google-cloud-components
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

WHEEL_FILE=$(find "$source_root/pipelines/components/google-cloud/dist/" -name "google_cloud_components*.whl")
pip3 install --upgrade $WHEEL_FILE
```

### High level overview

Google Cloud Components provides the an API for constructing your pipeline. To
start, let's walk through a simple workflow using this API.

1.  Let's begin with a Keras model training code such as the following, saved as
    `mnist_example.py`.

    ```python
    from google_cloud_components as gcc
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