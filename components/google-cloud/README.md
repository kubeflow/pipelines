# Google Cloud Pipeline Components

[![Python](https://img.shields.io/pypi/pyversions/google_cloud_pipeline_components.svg?style=plastic)](https://github.com/kubeflow/pipelines/tree/master/components/google-cloud)
[![PyPI](https://badge.fury.io/py/google-cloud-pipeline-components.svg)](https://badge.fury.io/py/google-cloud-pipeline-components.svg)

[Google Cloud Pipeline Components](https://cloud.google.com/vertex-ai/docs/pipelines/build-pipeline?hl=en#google-cloud-components) provides an SDK and a set of components that lets
you interact with Google Cloud services such as Vertex AI. You can use 
the predefined components in this repository to build your pipeline using the
Kubeflow Pipelines SDK.

## Documentation

### User Documentation

Please see the [Google_cloud_pipeline_components User Guide](https://cloud.google.com/vertex-ai/docs/pipelines/build-pipeline?hl=en#google-cloud-components).

### API documentation 
Please see the [Google_cloud_pipeline_components ReadTheDocs page](https://google-cloud-pipeline-components.readthedocs.io).

### Release Details

For detailed previous and upcoming changes, please [check here](https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/RELEASE.md)

## Examples
*   [Train an image classification model using Vertex AI AutoML](https://github.com/GoogleCloudPlatform/ai-platform-samples/blob/master/ai-platform-unified/notebooks/official/pipelines/google-cloud-pipeline-components_automl_images.ipynb).
*   [Train a classification model using tabular data and Vertex AI AutoML](https://github.com/GoogleCloudPlatform/ai-platform-samples/blob/master/ai-platform-unified/notebooks/official/pipelines/automl_tabular_classification_beans.ipynb).
*   [Train a linear regression model using tabular data and Vertex AI AutoML](https://github.com/GoogleCloudPlatform/ai-platform-samples/blob/master/ai-platform-unified/notebooks/official/pipelines/google-cloud-pipeline-components_automl_tabular.ipynb).
*   [Train a text classification model using Vertex AI AutoML](https://github.com/GoogleCloudPlatform/ai-platform-samples/blob/master/ai-platform-unified/notebooks/official/pipelines/google-cloud-pipeline-components_automl_text.ipynb).
*   [Use the Google Cloud pipeline components to upload and deploy a model](https://github.com/GoogleCloudPlatform/ai-platform-samples/blob/master/ai-platform-unified/notebooks/official/pipelines/google_cloud_pipeline_components_model_train_upload_deploy.ipynb).

## Installation

### Requirements

-   Python >= 3.6
-   [A Google Cloud project with the Vertex API enabled.](https://cloud.google.com/vertex-ai/docs/start/cloud-environment)
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
