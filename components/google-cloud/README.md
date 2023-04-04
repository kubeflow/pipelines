# Google Cloud Pipeline Components

[![Python](https://img.shields.io/pypi/pyversions/google_cloud_pipeline_components.svg?style=plastic)](https://github.com/kubeflow/pipelines/tree/master/components/google-cloud)
[![PyPI](https://badge.fury.io/py/google-cloud-pipeline-components.svg)](https://badge.fury.io/py/google-cloud-pipeline-components.svg)

[Google Cloud Pipeline Components](https://cloud.google.com/vertex-ai/docs/pipelines/build-pipeline?hl=en#google-cloud-components) (GCPC) provides predefined [KFP](https://www.kubeflow.org/docs/components/pipelines/) components that can be run on Google Cloud Vertex AI Pipelines and other KFP-conformant pipeline execution backends. You can compose the components together into pipelines using the [Kubeflow Pipelines SDK](https://pypi.org/project/kfp/).

## Documentation

### User documentation

Please see the [Google Cloud Pipeline Components user guide](https://cloud.google.com/vertex-ai/docs/pipelines/components-introduction).

### API documentation
Please see the [Google Cloud Pipeline Components API reference documentation](https://google-cloud-pipeline-components.readthedocs.io/en/google-cloud-pipeline-components-1.0.41/).

### Release details

For details about previous and upcoming releases, please see the [release notes](https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/RELEASE.md).

## Examples
*   [Train an image classification model using Vertex AI AutoML](https://github.com/GoogleCloudPlatform/vertex-ai-samples/blob/master/notebooks/official/pipelines/google_cloud_pipeline_components_automl_images.ipynb).
*   [Train a classification model using tabular data and Vertex AI AutoML](https://github.com/GoogleCloudPlatform/vertex-ai-samples/blob/master/notebooks/official/pipelines/automl_tabular_classification_beans.ipynb).
*   [Train a linear regression model using tabular data and Vertex AI AutoML](https://github.com/GoogleCloudPlatform/vertex-ai-samples/blob/master/notebooks/official/pipelines/google_cloud_pipeline_components_automl_tabular.ipynb).
*   [Train a text classification model using Vertex AI AutoML](https://github.com/GoogleCloudPlatform/vertex-ai-samples/blob/master/notebooks/official/pipelines/google_cloud_pipeline_components_automl_text.ipynb).
*   [Use the Google Cloud pipeline components to upload and deploy a model](https://github.com/GoogleCloudPlatform/vertex-ai-samples/blob/master/notebooks/official/pipelines/google_cloud_pipeline_components_model_train_upload_deploy.ipynb).

## Installation

### Requirements

-   Python >= 3.7
-   [A Google Cloud project with the Vertex API enabled.](https://cloud.google.com/vertex-ai/docs/start/cloud-environment)
-   An
    [authenticated GCP account](https://cloud.google.com/ai-platform/docs/getting-started-keras#authenticate_your_gcp_account)


### Install latest release

Use the following command to install Google Cloud Pipeline Components from PyPI.

```shell
pip install -U google-cloud-pipeline-components
```

### Install from source

Use the following commands to install Google Cloud Pipeline Components from [GitHub](https://github.com/kubeflow/pipelines/tree/master/components/google-cloud).

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
