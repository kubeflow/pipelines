# Notebooks as a Step

This solution builds a component that runs a notebook as a Kubeflow Pipelines step by using the Google Cloud [execution API](https://cloud.google.com/ai-platform/notebooks/docs/reference/rest/v1/projects.locations.executions).

## Labels

Vertex AI, Notebooks, Google Cloud, GCP, KubeFlow, Pipeline

## Summary

A Kubeflow Pipeline component to execute notebooks.

## [Optional] - Create a component.yaml file

Although this folder provides a component.yaml file, you can also create it from the source code of the component.

```source build_yaml.sh```

## Inputs

The Notebook-as-a-Step component takes the following inputs:

Name | Type | Default | Description
:--- | :--- | :------ | :----------
`project-id` | str | | Google Cloud Project ID (ex: `project-example`).
`region` | str | `'us-central1'`| Google Cloud Region (ex: `us-central1`).
`location` | str | `'us-central1-a'` | Google Cloud Zone (ex: `us-central1-a`).
`execution-id` | str | `'notebook_executor'` | Name of the execution to run. Must be unique amongst your project execution names.
`block` | str2bool | `'true'`| Whether to block the pipeline until this step is done.
`input-notebook-file` | str | | GCS path to notebook to execute. `gs://` is optional.
`output-notebook-folder` | str | | Cloud Storage folder under your working bucket for outputting result notebook. `gs://` is optional.
`master-type` | str | `'n1-standard-4'` | Specifies the type of virtual machine to use for the master node. You must specify this field when scaleTier is set to CUSTOM.
`scale-tier` | str | `'BASIC'` | Specifies the machine types, the number of replicas for workers and parameter servers.
`accelerator-type` | str | `'SCHEDULER_ACCELERATOR_TYPE_UNSPECIFIED'` | Accelerator type for hardware running notebook execution.
`accelerator-core-count` | str | `'0'`| Accelerator count for hardware running notebook execution.
`labels` | str | `'src=naas'` | Labels for execution of the form `k1=v1,k2=v2`.
`container-image-uri` | str | `'gcr.io/deeplearning-platform-release/base-cpu:latest'` | Container Image URI to a DLVM.
`params-yaml-file` | str | | Parameters used within the `input_notebook_file` notebook.
`parameters` | str | | Parameters file used within the `input_notebook_file` notebook.

## Outputs

Name| Type | Description
:---| :--- | :----------
state | `str` | State of the notebook execution.
output_notebook_file | `str` | Cloud Storage URI of the output executed notebook.

## Cautions & requirements

To use the component, you  must:

* Set up the GCP project by following these [steps](https://cloud.google.com/dataproc/docs/guides/setup-project).
* The component can authenticate to GCP. Refer to [Authenticating Pipelines to GCP](https://www.kubeflow.org/docs/gke/authentication-pipelines/) for details.
* Grant the following types of access to the Kubeflow user service account:
* The `notebooks.executions.create` IAM permission on the specified resource `parent`

## Use the component

From your client, clone the code of this repository

```bash
git clone https://github.com/kubeflow/pipelines.git
cd pipelines/components/gcp/notebook/execute_notebook/
```

### (Optional) Build the component

This step is optional but is useful if you customized the component code.

1. Set default variables

    `export PROJECT_ID="<YOUR_PROJECT_ID>"`

1. Build the image

    `chmod +x build_image.sh "${PROJECT_ID}"`

1. Update the `image:` in the [component.yaml](./component.yaml) file to match with your custom image.

    For example, you can replace `image: gcr.io/ml-pipeline/notebook-executor` with `image: gcr.io/<YOUR_PROJECT_ID>/notebook-executor` where `<YOUR_PROJECT_ID>` is the project ID string that you used in the "Set variables" section.

### Execute a notebook

Try the component by going through one of the notebooks in the [samples folder](./samples/).

We recommend using [sample-kfp-vertex.ipynb](./samples/sample-kfp-vertex.ipynb) which does not require you to set up a Kubeflow Pipelines environment on Kubernetes Engine.

## Reference

* Component python code
* Component docker file
* Sample notebook
* Notebook REST API

## License

By deploying or using this software you agree to comply with the AI Hub Terms of Service and the Google APIs Terms of Service. To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
