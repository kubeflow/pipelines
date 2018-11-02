## Compile
Follow [README.md](https://github.com/kubeflow/pipelines/blob/master/samples/README.md) to install the compiler and 
compile the sample python into workflow yaml. 

"sequential.yaml" is pre-generated for referencing purpose.

## Deploy
Open Pipelines UI, Follow the wizard to create a new pipeline and upload the generated workflow yaml.

## Run
Follow the pipeline UI to create pipeline runs. The parameter value for "exit_handler" and "sequential" samples can 
be "gs://ml-pipeline/shakespeare1.txt". The parameter values for "parallel_join" can be "gs://ml-pipeline/shakespeare1.txt"
and "gs://ml-pipeline/shakespeare2.txt".

## Components Source
All samples use pre-built components. The command to run for each container is built into the pipeline file.
