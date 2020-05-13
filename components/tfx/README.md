## Versions of TFX components that can be used with KFP SDK

Disclaimer: The components in this directory are unofficial and are maintained by the KFP team not the TFX team.

If you experience any issues in this components please create a new issue in the Kubeflow Pipelines repo and assign it to Ark-kun.

These components were created to allow the users to use TFX components in their KFP pipelines, to be able to mix KFP and TFX components.

If your pipeline uses only TFX components, please use the official [TFX SDK](https://www.tensorflow.org/tfx/tutorials/tfx/cloud-ai-platform-pipelines).

See the [sample pipeline](_samples/TFX_pipeline.ipynb) which showcases most of the components.

The components come in two flavors - with file-based I/O and URI-based I/O. With file-based I/O the system takes care of storing output data and making it available to downstream components.
With URI-based I/O, only the URIs pointing to the data are passed between components and the pipeline author is responsible for providing unique URIs for all output artifacts of the components in the pipeline.

### Google Cloud Dataflow

The TFX components that use URI-based I/O allow specifying the `beam_pipeline_args` parameter that enables setting the runner to `DataflowRunner` which executes the components on Google Cloud Dataflow.
See the [sample TFX on Dataflow pipeline](_samples/TFX_Dataflow_pipeline.ipynb).


Aspects and limitations
* These components use the official TFX container image.
* These components run the executors and component classes of the official TFX components.
* These components do not execute TFX [drivers](https://www.tensorflow.org/tfx/api_docs/python/tfx/components/base/base_driver), so they do not log metadata themselves (the metadata is logged by the Metadata Writer service instead). The properties of artifacts are currently not logged.
