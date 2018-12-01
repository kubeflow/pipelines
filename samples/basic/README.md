

This page tells you how to use the _basic_ sample pipelines contained in the repo.

## Compile the pipeline specification

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/guides/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK and compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.tar.gz` file. 

For convenience, you can download a pre-compiled, compressed YAML file containing the
specification of the `sequential.py` pipeline. This saves you the steps required
to compile and compress the pipeline specification:
[sequential.tar.gz](https://storage.googleapis.com/sample-package/sequential.tar.gz)

## Deploy

Open the Kubeflow pipelines UI, and follow the prompts to create a new pipeline and upload the generated workflow
specification, `my-pipeline.tar.gz` (example: `sequential.tar.gz`).

## Run

Follow the pipeline UI to create pipeline runs. 

Useful parameter values:

* For the "exit_handler" and "sequential" samples: `gs://ml-pipeline-playground/shakespeare1.txt`
* For the "parallel_join" sample: `gs://ml-pipeline-playground/shakespeare1.txt` and `gs://ml-pipeline-playground/shakespeare2.txt`

## Components source

All samples use pre-built components. The command to run for each container is built into the pipeline file.
