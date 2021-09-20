# Contributing

## Adding New Example

### Steps for adding new examples:

1. Create new example

    1. Copy the new example to `pipelines/samples/contrib/pytorch-samples/` (Ex: `pipelines/samples/contrib/pytorch-samples/iris`)
    2. Add yaml files for all components in the pipeline
    3. Create a pipeline file (Ex: `pipelines/samples/contrib/pytorch-samples/iris/pipeline.py`)

2. Build image and compile pipeline

    ```./build.sh <example-directory> <dockerhub-user-name>```

    For example:

    ```./build.sh cifar10 johnsmith```

    The following actions are done in the build script

    1. Bundling the code changes into a docker image
    2. Pushing the docker image to dockerhub
    3. Changing image tag in component.yaml
    4. Run `pipeline.py` file to generate yaml file which can be used to invoke the pipeline.

3. Upload pipeline and create a run 

    1. Create an Experiment from `kubeflow dashboard -> Experiments -> Create Experiment`.
    2. Upload the generated pipeline to `kubeflow dashboard -> Pipelines -> Upload Pipeline` 
    3. Create run from `kubeflow dashboard -> Pipelines -> {Pipeline Name} -> Create Run`

        **Pipeline params can be added while creating a run.**

    Refer: [Kubeflow Pipelines Quickstart](https://www.kubeflow.org/docs/components/pipelines/pipelines-quickstart/)
