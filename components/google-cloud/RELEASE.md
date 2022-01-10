# Current Version 0.2.2.dev (Still in Development)
* Add notes for next release here.

# Release 0.2.1
* Add BigqueryQueryJobOp, BigqueryCreateModelJobOp, BigqueryExportModelJobOp and BigqueryPredictModelJobOp components
* Add ModelEvaluationOp component
* Accept UnmanagedContainerModel artifact in Batch Prediction component
* Add util components and fix YAML for HP Tuning Job component; delete lightweight python version
* Add generic custom training job component
* Fix Dataflow error log reporting and component sample

# Release 0.2.0
* Update custom job name to create_custom_training_job_op_from_component
* Remove special handling for "=" in remote runner.
* Bug fixes and documentation updates.

# Release 0.1.9
* Dataflow and wait components
* Bug fixes

# Release 0.1.8
* Update the CustomJob component interface, and rename to custom_training_job_op
* Define new artifact types for Google Cloud resources.
* Update the AI Platform components. Added the component YAML and uses the new Google artifact types
* Add Vertex notebook component
* Various doc updates

# Release 0.1.7
* Add support for labels in custom_job wrapper.
* Add a component that connects the forecasting preprocessing and training components.
* Write GCP_RESOURCE proto for the custom_job output.
* Expose Custom Job parameters Service Account, Network and CMEK via Custom Job wrapper.
* Increase KFP min version dependency.
* AUpdate documentations for GCPC components.
* Update typing checks to include Python3.6 deprecated types.

# Release 0.1.6
* Experimental component for Model Forecast.
* Fixed issue with parameter passing for Vertex AI components
* Simplify auto generated API docs
* Fix parameter passing for explainability on ModelUploadOp
* Update naming of project and location parameters for all for GCPC components

# Release 0.1.5
* Experimental component for vertex forecasting preprocessing and validation

# Release 0.1.4

* Experimental component for tfp_anomaly_detection.
* Experimental module for Custom Job Wrapper.
* Fix to include YAML files in PyPI package.
* Restructure the google_cloud_pipeline_components.

# Release 0.1.3

*   Use correct dataset type when passing dataset to CustomTraining.
*   Bump google-cloud-aiplatform to 1.1.1.

# Release 0.1.2

*   Add components for AutoMLForecasting.
*   Update API documentation.

# Release 0.1.1

*   Fix issue with latest version of KFP not accepting pipeline_root in kfp.compile.
*   Fix Compatibility with latest AI Platform name change to replace resource name class with Vertex AI

# Release 0.1.0

## First release

*   Initial release of the Python SDK with data and model managemnet operations for Image, Text, Tabular, and Video Data.
