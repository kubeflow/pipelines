# Parameterized TFX pipeline sample

[Tensorflow Extended (TFX)](https://github.com/tensorflow/tfx) is a Google-production-scale machine
learning platform based on TensorFlow. It provides a configuration framework to express ML pipelines
consisting of TFX components. Kubeflow Pipelines can be used as the orchestrator supporting the 
execution of a TFX pipeline.

This sample demonstrates how to author a ML pipeline in TFX and run it on a KFP deployment. 
Please refer to inline comments for the purpose of each step.

In order to successfully compile this sample, you'll need to have a TFX installation at HEAD.
First, you can clone their repo and run `python setup.py install` from `tfx/tfx`. 
The image used in the pipeline is specified as `tfx_image` in the 
`KubeflowDagRunnerConfig`. Currently we're using our own patched version of TFX image containing visualization support.
List of officially released nightly build image available can be found [here](https://hub.docker.com/r/tensorflow/tfx/tags)).

After that, running 
`python3 chicago_taxi_pipeline_simple.py` compiles the TFX pipeline into KFP pipeline package.
This pipeline requires google storage permission to run. 


## Caveats

This sample uses pipeline parameters in a TFX pipeline, which is not yet fully supported. 
See [here](https://github.com/tensorflow/tfx/issues/362) for more details. In this sample, however,
the path to module file and path to data are parameterized. This is achieved by specifying those
objects `dsl.PipelineParam` and appending them to the `KubeflowDagRunner._params`. Then, 
KubeflowDagRunner can correctly identify those pipeline parameters and interpret them as Argo
placeholder correctly when compilation. However, this parameterization approach is a hack and 
we do not have plan for long-term support. Instead we're working with TFX team to support 
pipeline parameterization using their [RuntimeParameter](https://github.com/tensorflow/tfx/blob/46bb4f975c36ea1defde4b3c33553e088b3dc5b8/tfx/orchestration/data_types.py#L108).

### Known issues
* This approach only works for string-typed quantities. For example, you cannot parameterize 
`num_steps` of `Trainer` in this way.
* Name of parameters should be unique.
* By default pipeline root is always parameterized with the name `pipeline-root`.
* If the parameter is referenced at multiple places, the user should
make sure that it is correctly converted to the string-formatted placeholder by
calling `str(your_param)`.
* The best practice is to specify TFX pipeline root to an empty dir. In this sample Argo automatically do that by plugging in the 
workflow unique ID (represented `kfp.dsl.RUN_ID_PLACEHOLDER`) to the pipeline root path.
