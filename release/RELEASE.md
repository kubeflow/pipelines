# Current version under development

Kubeflow Pipelines 1.0

## Major features and improvements

### Features:

* CREATE/GET/LIST/DELETE/ARCHIVE the Kubeflow Pipelines resources of pipelines, pipeline versions, runs, jobs, experiments. []

* Upload the Kubeflow Pipelines resources of pipelines and pipeline versions.

* Separate the Kubeflow Pipelines resources with multiple user namespace support.

### Components:

* Added [Google Cloud Storage components](https://github.com/kubeflow/pipelines/tree/290fa55fb9c908be38cbe6bc4c3f477da6b0e97f/components/google-cloud/storage): [Download](https://github.com/kubeflow/pipelines/tree/290fa55fb9c908be38cbe6bc4c3f477da6b0e97f/components/google-cloud/storage/download), [Upload to explicit URI](https://github.com/kubeflow/pipelines/tree/290fa55fb9c908be38cbe6bc4c3f477da6b0e97f/components/google-cloud/storage/upload_to_explicit_uri), [Upload to unique URI](https://github.com/kubeflow/pipelines/tree/290fa55fb9c908be38cbe6bc4c3f477da6b0e97f/components/google-cloud/storage/upload_to_unique_uri) and [List](https://github.com/kubeflow/pipelines/tree/290fa55fb9c908be38cbe6bc4c3f477da6b0e97f/components/google-cloud/storage/list).

### Experimental Features

* Visualization of output artifacts in two ways: [built-in visualization of selected artifact types](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations/) and [python based custom visualization](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations/).

*



## Bug fixes and other changes

### Deprecations

## Breaking changes
