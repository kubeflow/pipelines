The sample pipelines give you a quick start to build and deploy machine learning pipelines with Kubeflow Pipeline.
* Follow the guide to [deploy the Kubeflow pipelines service](https://www.kubeflow.org/docs/guides/pipelines/deploy-pipelines-service/).
* Build and deploy your pipeline [using the provided samples](https://www.kubeflow.org/docs/guides/pipelines/pipelines-samples/).

# Sample Structure
The samples are organized into the core set and the contrib set. 

**Core samples** demonstrate the full KFP functionality.
A selected set of these core samples will also be preloaded to the KFP during deployment. 
The core samples will also include intermediate samples that are 
more complex than basic samples such as flip coins but simpler than TFX samples. 
It serves to demonstrate a set of the outstanding features and offers users the next level KFP experience.

# Run Samples

## Compile the pipeline specification

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/guides/pipelines/build-pipeline/) to install the Kubeflow 
Pipelines SDK and compile the sample Python into a workflow specification. 
The specification takes one of the three forms: YAML file, YAML compressed into a `.tar.gz` file, and YAML compressed into a `.zip` file

For convenience, you can use the preloaded samples in the pipeline system. This saves you the steps required
to compile and compress the pipeline specification.

## Upload the pipeline to the Kubeflow Pipeline

Open the Kubeflow pipelines UI, and follow the prompts to create a new pipeline and upload the generated workflow
specification, `my-pipeline.zip` (example: `sequential.zip`).

## Run the pipeline

Follow the pipeline UI to create pipeline runs. 

Useful parameter values:

* For the "exit_handler" and "sequential" samples: `gs://ml-pipeline-playground/shakespeare1.txt`
* For the "parallel_join" sample: `gs://ml-pipeline-playground/shakespeare1.txt` and `gs://ml-pipeline-playground/shakespeare2.txt`

## Notes: component source codes

All samples use pre-built components. The command to run for each container is built into the pipeline file.

# Sample contribution
For better readability, samples are encouraged to adopt the following conventions.

* The sample file should be either `*.py` or `*.ipynb`, and its file name is consistent with its directory name.
* For `*.py` sample, it's recommended to have a main invoking `kfp.compiler.Compiler().compile()` to compile the 
pipeline function into pipeline yaml spec.
* For `*.ipynb` sample, parameters (e.g., `project_name`)
should be defined in a dedicated cell and tagged as parameter.
Detailed guideline is 
[here](https://github.com/nteract/papermill). Also, all the environment setup and 
preparation should be within the notebook, such as by `!pip install packages` 
