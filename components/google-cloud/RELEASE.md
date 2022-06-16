# Current Version 1.0.10.dev (Still in Development)
* Add notes for next release here.

# Release 1.0.9
* FTE transform config passed as path to config file instead of directly as string to FTE
* Support BigQuery ML weights job component
* FTE now outputs training schema.
* Support BigQuery ML reconstruction loss and trial info job components
* Adding ML.TRAINING_INFO KFP and ML.EXPLAIN_PREDICT BQ Component.
* Add additional experiments in distillation pipeline.
* Support BigQuery ML advanced weights job component.
* Support BigQuery drop model job components.
* Support BigQuery ML centroids job components.
* Wide and Deep and Tabnet models both now use the Feature Transform Engine pipeline instead of the Transform component.
* Adding ML.CONFUSION_MATRIX KFP BQ Component.
* Adding ML.FEATURE_INFO KFP BQ Component.
* Merge distill_skip_evaluation and skip_evaluation pipelines with default pipeline using dsl.Condition
* Adding ML.ROC_CURVE KFP BQ Component.
* Adding ML.PRINCIPAL_COMPONENTS and ML.PRINCIPAL_COMPONENT_INFO KFP BQ component.
* Adding ML.FEATURE_IMPORTANCE KFP BQ Component.
* Add ML.ARIMA_COEFFICIENTS in component.yaml
* Adding ML.Recommend KFP BQ component.
* Add ML.ARIMA_EVALUATE in component.yaml
* KFP component for ml.explain_forecast
* KFP component for ml.forecast
* Add distill + evaluation pipeline for Tables
* Adding ML.GLOBAL_EXPLAIN KFP BQ Component.
* KFP component for ml.detect_anomalies
* Make stats-gen component to support running with example-gen only mode
* Fix AutoML Tables pipeline and builtin pipelines on VPC-SC environment.
* Preserve empty features in explanation_spec

# Release 1.0.8
* Use BigQuery batch queries in ARIMA pipeline after first 50 queries
* Stats Gen and Feature Transform Engine pipeline integration.
* Add window config to ARIMA pipeline
* Removed default location setting from AutoML components and documentation.
* Update default machine type to c2-standard-16 for built-in algorithms Custom and HyperparameterTuning Jobs
* Use float instead of int max windows, which caused ARIMA pipeline failure
* Renamed "Feature Transform Engine Transform Configuration" component to "Transform Configuration Planner" for clarity.
* Preserve empty features in explanation_spec
* Change json util to not remove empty primitives in a list.
* Add model eval component to built-in algorithm default pipelines
* Quick fix to Batch Prediction component input "bigquery_source_input_uri"

# Release 1.0.7
* Allow metrics and evaluated examples tables to be overwritten.
* Replace custom copy_table component with BQ first-party query component.
* Support vpc in feature selection.
* Add import eval metrics to model to AutoML Tables default pipeline.
* Add default Wide & Deep study_spec_parameters configs and add helper function to utils.py to get parameters.

# Release 1.0.6
* Update import evaluation metrics component.
* Support parameterized input for reserved_ip_range and other Vertex Training parameters in custom job utility.
* Generate feature selection tuning pipeline and test utils.
* Add retries to queries hitting BQ write quota on BQML Arima pipeline.
* Minor changes to the feature transform engine and transform configuration component specs to support their integration.
* Update Executor component for Pipeline to support kernel_spec.
* Add default TabNet study_spec_parameters_override configs for different dataset sizes and search space modes and helper function to get the parameters.

# Release 1.0.5
* Add VPC-SC and CMEK support for the experimental evaluation component
* Add an import evaluation metrics component
* Modify AutoML Tables template JSON pipeline specs
* Add feature transform engine AutoML Table component.

# Release 1.0.4
* Create alias for create_custom_training_job_op_from_component as create_custom_training_job_from_component
* Add support for env variables in Custom_Job component.

# Release 1.0.3
* Add API docs for Vertex Notification Email
* Add template JSON pipeline spec for running evaluation on a managed GCP Vertex model.
* Update documentation for Dataproc Serverless components v1.0.
* Use if:cond:then when specifying image name in built-in algorithm hyperparameter tuning job component and add separate hyperparameter tuning job default pipelines for TabNet and Wide & Deep
* Add gcp_resources in the eval component output
* Add downsampled_test_split_json to example_and_stats_gen component.

# Release 1.0.2
* Dataproc Serverless components v1.0 launch.
* Bump google-cloud-aiplatform version
* Fix HP Tuning documentation, fixes #7460
* Use feature ranking and selected features in AutoML Tables stage 1 tuning component.
* Update distill_skip_evaluation_pipeline for performance improvement.

# Release 1.0.1
* Add experimental email notification component
* add docs for create_custom_training_job_op_from_component
* Remove ForecastingTrainingWithExperimentsOp component.
* Use unmanaged_container_model for model_upload for AutoML Tables pipelines
* add nfs mount support for create_custom_training_job_op_from_component
* Implement cancellation for dataproc components
* bump google-api-core version to 2.0+
* Add retry for batch prediction component

# Release 1.0.0
* add enable_web_access for create_custom_training_job_op_from_component
* remove remove training_filter_split, validation_filter_split, test_filter_split from automl components
* Update the dataproc component docs

# Release 0.3.1
* Implement cancellation propagation
* Remove encryption key in input for BQ create model
* Add Dataproc Batch components
* Add AutoML Tables Wide & Deep trainer component and pipeline
* Create GCPC v1 and readthedocs for v1
* Fix bug when ExplanationMetadata.InputMetadata field is provided the batch prediction job component

# Release 0.3.0
* Update BQML export model input from string to artifact
* Move model/endpoint/job/bqml compoennts to 1.0 namespace
* Expose `enable_web_access` and `reserved_ip_ranges` for custom job component
* Add delete model and undeploy model components
* Add utility library for google artifacts

# Release 0.2.2
* Fixes for BQML components
* Add util functions for HP tuning components and update samples

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
