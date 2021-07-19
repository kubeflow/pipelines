# PyTorch Kubeflow Pipeline Components

PyTorch Kubeflow Pipeline Components provides an SDK and a set of components that lets you build kubeflow pipelines using PyTorch. You can use the predefined components in this repository to build your pipeline using the Kubeflow Pipelines SDK.

## Installation
### Requirements
Python >= 3.6
Kubeflow cluster setuo (on-prem or in any of the Clouds)

### Install latest release
Use the following command to install PyTorch Pipeline Components from PyPI.

```
pip install -U pytorch-kfp-components
```

### Install from source
Use the following commands to install PyTorch Kubeflow Pipeline Components from GitHub.

```
git clone https://github.com/kubeflow/pipelines.git
pip install pipelines/components/PyTorch/pytorch_kfp_components/.
```

### Running the tests

Run the following command

```
pip install tox
cd ./components/PyTorch/pytorch-kfp-components/
tox -rvve py38
```

### Samples

For running the samples follow the instruction mentioned as below

[Samples README.md](../../../samples/contrib/pytorch-samples/README.md)