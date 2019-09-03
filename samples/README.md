The sample pipelines give you a quick start to building and deploying machine learning pipelines with Kubeflow.
* Follow the guide to [deploy the Kubeflow pipelines service](https://www.kubeflow.org/docs/guides/pipelines/deploy-pipelines-service/).
* Build and deploy your pipeline [using the provided samples](https://www.kubeflow.org/docs/guides/pipelines/pipelines-samples/).




This page tells you how to use the _basic_ sample pipelines contained in the repo.

## Compile the pipeline specification

Follow the guide to [building a pipeline](https://www.kubeflow.org/docs/guides/pipelines/build-pipeline/) to install the Kubeflow Pipelines SDK and compile the sample Python into a workflow specification. The specification takes the form of a YAML file compressed into a `.tar.gz` file. 

For convenience, you can download a pre-compiled, compressed YAML file containing the
specification of the `core/sequential.py` pipeline. This saves you the steps required
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

## Sample conventions
For better readability and functions of sample test, samples are encouraged to adopt the following conventions.

* The sample should be either `*.py` or `*.ipynb`, and its file name is in consistence with its dir name.
* For `*.py` sample, it's recommended to have a main invoking `kfp.compiler.Compiler().compile()` to compile the 
pipeline function into pipeline yaml spec.
* For `*.ipynb` sample, parameters (e.g., experiment name and project name) should be defined in a dedicated cell and 
tagged as parameter. Detailed guideline is [here](https://github.com/nteract/papermill). Also, custom environment setup
should be done within the notebook itself by `!pip install`


## (Optional) Add sample test
For those samples that cover critical functions of KFP, possibly it should be picked up by KFP's sample test
so that it won't be broken by accidental PR merge. Contributors can do that by following these steps.

* Place the sample under the core sample directory `kubeflow/pipelines/samples/core`
* Make sure it follows the [sample conventions](#sample-conventions).
* If the test running requires specific values of pipeline parameters, they can be specified in a config yaml file
placed under `test/sample-test/configs`. See 
[`tfx_cab_classification.config.yaml`](https://github.com/kubeflow/pipelines/blob/master/test/sample-test/configs/tfx_cab_classification.config.yaml) as an example. The config yaml file will be validated 
according to `schema.config.yaml`. If no config yaml is provided, pipeline parameters will be substituted by their 
default values.
* Finally, add your test name (in consistency with the file name and dir name) in 
[`test/sample_test.yaml`](https://github.com/kubeflow/pipelines/blob/ecd93a50564652553260f8008c9a2d75ab907971/test/sample_test.yaml#L69)
* (Optional) Current sample test infra only checks successful runs, without any result/outcome validation. 
If those are needed, runtime checks should be included in the sample itself.
