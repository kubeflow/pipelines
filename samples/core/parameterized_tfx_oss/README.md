# Overview

[Tensorflow Extended (TFX)](https://github.com/tensorflow/tfx) is a Google-production-scale machine
learning platform based on TensorFlow. It provides a configuration framework to express ML pipelines
consisting of TFX components. Kubeflow Pipelines can be used as the orchestrator supporting the 
execution of a TFX pipeline.

This directory contains two samples that demonstrate how to author a ML pipeline in TFX and run it 
on a KFP deployment. 
* `parameterized_tfx_oss.py` is a Python script that outputs a compiled KFP workflow, which you can
  submit to a KFP deployment to run;
* `parameterized_tfx_oss.ipynb` is a notebook version of `parameterized_tfx_oss.py`, and it also
  includes the guidance to setup its dependencies.

Please refer to inline comments for the purpose of each step in both samples.

# Compilation
* `parameterized_tfx_oss.py`: 
In order to successfully compile the Python sample, you'll need to have a TFX installation at 
version 0.15.0 by running `python3 -m pip install tfx==0.15.0`. After that, run
`python3 parameterized_tfx_oss.py` to compile the TFX pipeline into KFP pipeline package.

* `parameterized_tfx_oss.ipynb`:

# Permission

This pipeline requires Google Cloud Storage permission to run. 
If KFP was deployed through K8S marketplace, please follow instructions in 
[the guideline](https://github.com/kubeflow/pipelines/blob/master/manifests/gcp_marketplace/guide.md#gcp-service-account-credentials)
to make sure the service account has `storage.admin` role.

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
