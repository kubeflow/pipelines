# Compiling and Running the pipeline by uploading to Kubeflow Pipelines

This covers instructions on building the Bert and Cifar10 example pipelines locally using KFP sdk and uploading the file to Kubeflow Pipeline and starting a run out of that pipeline.

## Prerequisites

[KFP Python SDK](https://github.com/kubeflow/pipelines/tree/master/sdk/python)

### Generating component.yaml from templates

Follow the readme file for generating component.yaml files using templates

[generate component.yaml from templates](utils/template-generation.md)

### Building the pipeline

Run the below commands for building pipeline for the existing Cifar 10 and Bert examples

`python cifar10/pipeline.py`

or

`python bert/pipeline.py`

The output of the above script will generate a yaml file which can be uploaded to KFP for invoking a run.

### Uploading and invoking a run

  1. Create an Experiment from `kubeflow dashboard -> Experiments -> Create Experiment`.
  2. Upload the generated pipeline to `kubeflow dashboard -> Pipelines -> Upload Pipeline` 
  3. Create run from `kubeflow dashboard -> Pipelines -> {Pipeline Name} -> Create Run`

      **Pipeline params can be added while creating a run.**

  Refer: [Kubeflow Pipelines Quickstart](https://www.kubeflow.org/docs/components/pipelines/pipelines-quickstart/)
  
  4. Click on the visualization tab, select the custom tensorboard image from the dropdown (examples screenshot shown below) and click `Start Tensorboard`. Tensoboard UI will be loaded with the run details.

  ![](screenshots/tensorboard.png)


**For testing any code changes or adding new examples, use the build script**

  Refer: [Creating New examples](README.md##Adding-new-example)


