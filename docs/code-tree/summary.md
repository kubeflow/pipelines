# Codebase Knowledge Graph

Generated: 2026-02-13 19:07 UTC
Parser: AST + Regex

## Statistics

- **1996** files across **7** languages
- **17636** symbols extracted
  - 10634 methods
  - 3285 functions
  - 1354 structs
  - 1180 class
  - 743 interfaces
  - 347 constants
  - 93 enums
- **14938** import relationships
- **484** inheritance relationships

## Languages

| Language | Files |
|----------|-------|
| python | 908 |
| go | 693 |
| typescript | 273 |
| bash | 57 |
| protobuf | 54 |
| javascript | 10 |
| sql | 1 |

## Directory Structure

```
  v2alpha1/ (2 files, protobuf)
      cachekey/ (1 files, go)
      pipelinespec/ (1 files, go)
      protobuf/ (17 files, protobuf)
      rpc/ (1 files, protobuf)
    python/ (2 files, python)
        pipeline_spec/ (2 files, python)
backend/ (1 files, bash)
  api/ (1 files, bash)
    hack/ (1 files, bash)
    v1beta1/ (14 files, protobuf)
      go_client/ (33 files, go)
        experiment_client/ (1 files, go)
          experiment_service/ (13 files, go)
        experiment_model/ (9 files, go)
        healthz_client/ (1 files, go)
          healthz_service/ (3 files, go)
        healthz_model/ (3 files, go)
        job_client/ (1 files, go)
          job_service/ (13 files, go)
        job_model/ (16 files, go)
        pipeline_client/ (1 files, go)
          pipeline_service/ (25 files, go)
        pipeline_model/ (13 files, go)
        pipeline_upload_client/ (1 files, go)
          pipeline_upload_service/ (5 files, go)
        pipeline_upload_model/ (10 files, go)
        run_client/ (1 files, go)
          run_service/ (19 files, go)
        run_model/ (21 files, go)
        visualization_client/ (1 files, go)
          visualization_service/ (3 files, go)
        visualization_model/ (4 files, go)
      python_http_client/ (2 files, bash, python)
        kfp_server_api/ (5 files, python)
          api/ (7 files, python)
          models/ (39 files, python)
    v2beta1/ (11 files, protobuf)
      go_client/ (30 files, go)
        artifact_client/ (1 files, go)
          artifact_service/ (15 files, go)
        artifact_model/ (16 files, go)
        experiment_client/ (1 files, go)
          experiment_service/ (13 files, go)
        experiment_model/ (5 files, go)
        healthz_client/ (1 files, go)
          healthz_service/ (3 files, go)
        healthz_model/ (3 files, go)
        pipeline_client/ (1 files, go)
          pipeline_service/ (21 files, go)
        pipeline_model/ (9 files, go)
        pipeline_upload_client/ (1 files, go)
          pipeline_upload_service/ (5 files, go)
        pipeline_upload_model/ (5 files, go)
        recurring_run_client/ (1 files, go)
          recurring_run_service/ (13 files, go)
        recurring_run_model/ (12 files, go)
        run_client/ (1 files, go)
          run_service/ (27 files, go)
        run_model/ (32 files, go)
        visualization_client/ (1 files, go)
          visualization_service/ (3 files, go)
        visualization_model/ (4 files, go)
        rpc/ (1 files, protobuf)
      python_http_client/ (2 files, bash, python)
        kfp_server_api/ (5 files, python)
          api/ (11 files, python)
          models/ (69 files, python)
  conformance/ (1 files, bash)
      persistence/ (2 files, go)
        client/ (6 files, go)
          artifactclient/ (2 files, go)
          tokenrefresher/ (1 files, go)
        worker/ (4 files, go)
    apiserver/ (2 files, go)
      archive/ (1 files, go)
      auth/ (4 files, go)
      client/ (16 files, go)
      client_manager/ (2 files, go)
      common/ (4 files, go)
      config/ (1 files, go)
        proxy/ (1 files, go)
      filter/ (1 files, go)
      list/ (1 files, go)
      model/ (17 files, go)
      resource/ (3 files, go)
      server/ (15 files, go)
      storage/ (16 files, go)
      template/ (3 files, go)
      validation/ (2 files, go)
      visualization/ (3 files, bash, python)
        types/ (5 files, python)
      webhook/ (1 files, go)
    cache/ (2 files, go)
      client/ (4 files, go)
      deployer/ (3 files, bash)
      model/ (1 files, go)
      server/ (4 files, go)
      storage/ (3 files, go)
    common/ (1 files, go)
        api_server/ (1 files, go)
          v1/ (13 files, go)
          v2/ (12 files, go)
      util/ (25 files, go)
        scheduledworkflow/ (2 files, go)
          client/ (3 files, go)
          util/ (5 files, go)
        viewer/ (1 files, go)
          reconciler/ (1 files, go)
      hack/ (2 files, bash)
        v2beta1/ (4 files, go)
          scheduledworkflow/ (1 files, go)
            v1beta1/ (4 files, go)
          viewer/ (1 files, go)
            v1beta1/ (4 files, go)
            versioned/ (2 files, go)
              fake/ (3 files, go)
              scheme/ (2 files, go)
                  v1beta1/ (4 files, go)
                    fake/ (3 files, go)
            externalversions/ (2 files, go)
              internalinterfaces/ (1 files, go)
              scheduledworkflow/ (1 files, go)
                v1beta1/ (2 files, go)
              v1beta1/ (2 files, go)
      apiclient/ (2 files, go)
        kfpapi/ (2 files, go)
      cacheutils/ (1 files, go)
      client_manager/ (2 files, go)
        compiler/ (1 files, go)
        driver/ (2 files, go)
        launcher-v2/ (1 files, go)
      compiler/ (1 files, go)
        argocompiler/ (7 files, go)
        testdata/ (1 files, python)
      component/ (7 files, go)
      config/ (4 files, go)
      driver/ (7 files, go)
        common/ (1 files, go)
        resolver/ (4 files, go)
      expression/ (1 files, go)
      objectstore/ (2 files, go)
components/ (1 files, bash)
    Convert_to_OnnxModel_from_PyTorchScriptModule/ (1 files, python)
    Create_fully_connected_network/ (1 files, python)
      from_CSV/ (1 files, python)
    _samples/ (1 files, python)
    pytorch-kfp-components/ (1 files, python)
      pytorch_kfp_components/ (1 files, python)
          base/ (2 files, python)
          mar/ (2 files, python)
          minio/ (2 files, python)
          trainer/ (3 files, python)
          utils/ (2 files, python)
          visualization/ (2 files, python)
        types/ (1 files, python)
        src/ (1 files, python)
      common/ (1 files, python)
        src/ (1 files, python)
        src/ (1 files, python)
        src/ (1 files, python)
        src/ (1 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
          mnist-kmeans-training/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        scripts/ (3 files, bash, python)
      common/ (7 files, python)
      commonv2/ (5 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (2 files, python)
        src/ (3 files, python)
        src/ (2 files, python)
  google-cloud/ (1 files, python)
    docs/ (1 files, bash)
      source/ (1 files, python)
    google_cloud_pipeline_components/ (5 files, python)
      _implementation/ (1 files, python)
        llm/ (24 files, python)
        model/ (1 files, python)
          get_model/ (2 files, python)
        model_evaluation/ (3 files, python)
          chunking/ (3 files, python)
          data_sampler/ (2 files, python)
          dataset_preprocessor/ (2 files, python)
          endpoint_batch_predict/ (2 files, python)
          error_analysis_annotation/ (2 files, python)
          evaluated_annotation/ (2 files, python)
          feature_attribution/ (3 files, python)
          feature_extractor/ (2 files, python)
          import_evaluated_annotation/ (2 files, python)
          llm_classification_postprocessor/ (2 files, python)
          llm_embedding/ (2 files, python)
          llm_embedding_retrieval/ (2 files, python)
          llm_evaluation/ (2 files, python)
          llm_evaluation_preprocessor/ (2 files, python)
          llm_information_retrieval_preprocessor/ (2 files, python)
          llm_retrieval_metrics/ (2 files, python)
          llm_safety_bias/ (3 files, python)
          model_inference/ (2 files, python)
          model_name_preprocessor/ (2 files, python)
          target_field_data_remover/ (2 files, python)
          text2sql/ (2 files, python)
          text2sql_evaluation/ (2 files, python)
          text2sql_preprocess/ (2 files, python)
          text2sql_validate_and_process/ (2 files, python)
        starry_net/ (2 files, python)
          dataprep/ (2 files, python)
          evaluation/ (2 files, python)
          get_training_artifacts/ (2 files, python)
          maybe_set_tfrecord_args/ (2 files, python)
          set_dataprep_args/ (2 files, python)
          set_eval_args/ (2 files, python)
          set_tfrecord_args/ (2 files, python)
          set_train_args/ (2 files, python)
          train/ (2 files, python)
          upload_decomposition_plots/ (2 files, python)
          upload_model/ (2 files, python)
      container/ (1 files, python)
        _implementation/ (1 files, python)
          llm/ (1 files, python)
            templated_custom_job/ (3 files, python)
          model/ (1 files, python)
            get_model/ (2 files, python)
          model_evaluation/ (3 files, python)
        preview/ (1 files, python)
          custom_job/ (3 files, python)
          dataflow/ (1 files, python)
            flex_template/ (3 files, python)
          gcp_launcher/ (2 files, python)
        utils/ (4 files, python)
        v1/ (1 files, python)
          aiplatform/ (3 files, python)
          automl_training_job/ (1 files, python)
            image/ (3 files, python)
          batch_prediction_job/ (3 files, python)
          bigquery/ (1 files, python)
            create_model/ (3 files, python)
            detect_anomalies_model/ (3 files, python)
            drop_model/ (3 files, python)
            evaluate_model/ (3 files, python)
            explain_forecast_model/ (3 files, python)
            explain_predict_model/ (3 files, python)
            export_model/ (3 files, python)
            feature_importance/ (3 files, python)
            forecast_model/ (3 files, python)
            global_explain/ (3 files, python)
            ml_advanced_weights/ (3 files, python)
            ml_arima_coefficients/ (3 files, python)
            ml_arima_evaluate/ (3 files, python)
            ml_centroids/ (3 files, python)
            ml_confusion_matrix/ (3 files, python)
            ml_feature_info/ (3 files, python)
            ml_principal_component_info/ (3 files, python)
            ml_principal_components/ (3 files, python)
            ml_recommend/ (3 files, python)
            ml_reconstruction_loss/ (3 files, python)
            ml_roc_curve/ (3 files, python)
            ml_training_info/ (3 files, python)
            ml_trial_info/ (3 files, python)
            ml_weights/ (3 files, python)
            predict_model/ (3 files, python)
            query_job/ (3 files, python)
            utils/ (2 files, python)
          custom_job/ (3 files, python)
          dataflow/ (3 files, python)
          dataproc/ (1 files, python)
            create_pyspark_batch/ (3 files, python)
            create_spark_batch/ (3 files, python)
            create_spark_r_batch/ (3 files, python)
            create_spark_sql_batch/ (3 files, python)
            utils/ (2 files, python)
          endpoint/ (1 files, python)
            create_endpoint/ (3 files, python)
            delete_endpoint/ (3 files, python)
            deploy_model/ (3 files, python)
            undeploy_model/ (3 files, python)
          gcp_launcher/ (4 files, python)
            utils/ (5 files, python)
          hyperparameter_tuning_job/ (3 files, python)
          infra_validation_job/ (3 files, python)
          model/ (1 files, python)
            delete_model/ (3 files, python)
            export_model/ (3 files, python)
            get_model/ (3 files, python)
            upload_model/ (3 files, python)
          vertex_notification_email/ (2 files, python)
          wait_gcp_resources/ (3 files, python)
      preview/ (1 files, python)
        automl/ (1 files, python)
          forecasting/ (5 files, python)
          tabular/ (10 files, python)
          vision/ (3 files, python)
        custom_job/ (3 files, python)
        dataflow/ (1 files, python)
        llm/ (1 files, python)
          infer/ (2 files, python)
          rlaif/ (2 files, python)
          rlhf/ (2 files, python)
        model_evaluation/ (7 files, python)
        starry_net/ (2 files, python)
      proto/ (7 files, protobuf, python)
      types/ (2 files, python)
      v1/ (1 files, python)
        automl/ (1 files, python)
          forecasting/ (3 files, python)
          tabular/ (11 files, python)
          training_job/ (1 files, python)
            automl_forecasting_training_job/ (2 files, python)
            automl_image_training_job/ (2 files, python)
            automl_tabular_training_job/ (2 files, python)
            automl_text_training_job/ (2 files, python)
            automl_video_training_job/ (2 files, python)
        batch_predict_job/ (2 files, python)
        bigquery/ (1 files, python)
          create_model/ (2 files, python)
          detect_anomalies_model/ (2 files, python)
          drop_model/ (2 files, python)
          evaluate_model/ (2 files, python)
          explain_forecast_model/ (2 files, python)
          explain_predict_model/ (2 files, python)
          export_model/ (2 files, python)
          feature_importance/ (2 files, python)
          forecast_model/ (2 files, python)
          global_explain/ (2 files, python)
          ml_advanced_weights/ (2 files, python)
          ml_arima_coefficients/ (2 files, python)
          ml_arima_evaluate/ (2 files, python)
          ml_centroids/ (2 files, python)
          ml_confusion_matrix/ (2 files, python)
          ml_feature_info/ (2 files, python)
          ml_principal_component_info/ (2 files, python)
          ml_principal_components/ (2 files, python)
          ml_recommend/ (2 files, python)
          ml_reconstruction_loss/ (2 files, python)
          ml_roc_curve/ (2 files, python)
          ml_training_info/ (2 files, python)
          ml_trial_info/ (2 files, python)
          ml_weights/ (2 files, python)
          predict_model/ (2 files, python)
          query_job/ (2 files, python)
        custom_job/ (3 files, python)
        dataflow/ (1 files, python)
          flex_template/ (2 files, python)
          python_job/ (2 files, python)
        dataproc/ (1 files, python)
          create_pyspark_batch/ (2 files, python)
          create_spark_batch/ (2 files, python)
          create_spark_r_batch/ (2 files, python)
          create_spark_sql_batch/ (2 files, python)
        dataset/ (1 files, python)
          create_image_dataset/ (2 files, python)
          create_tabular_dataset/ (2 files, python)
          create_text_dataset/ (2 files, python)
          create_time_series_dataset/ (2 files, python)
          create_video_dataset/ (2 files, python)
          export_image_dataset/ (2 files, python)
          export_tabular_dataset/ (2 files, python)
          export_text_dataset/ (2 files, python)
          export_time_series_dataset/ (2 files, python)
          export_video_dataset/ (2 files, python)
          get_vertex_dataset/ (2 files, python)
          import_image_dataset/ (2 files, python)
          import_text_dataset/ (2 files, python)
          import_video_dataset/ (2 files, python)
        endpoint/ (1 files, python)
          create_endpoint/ (2 files, python)
          delete_endpoint/ (2 files, python)
          deploy_model/ (2 files, python)
          undeploy_model/ (2 files, python)
        forecasting/ (1 files, python)
          prepare_data_for_train/ (2 files, python)
          preprocess/ (2 files, python)
          validate/ (2 files, python)
        hyperparameter_tuning_job/ (3 files, python)
        model/ (1 files, python)
          delete_model/ (2 files, python)
          export_model/ (2 files, python)
          get_model/ (2 files, python)
          upload_model/ (2 files, python)
        model_evaluation/ (12 files, python)
          model_based_llm_evaluation/ (1 files, python)
            autosxs/ (2 files, python)
        vertex_notification_email/ (2 files, python)
        wait_gcp_resources/ (2 files, python)
    src/ (1 files, python)
    pytorch-launcher/ (1 files, python)
  snowflake/ (1 files, python)
  sdk/ (2 files, bash, python)
frontend/ (4 files, javascript)
  mock-backend/ (4 files, typescript)
        runtime/ (9 files, typescript)
  scripts/ (8 files, bash, javascript)
  server/ (11 files, typescript)
    handlers/ (9 files, typescript)
    helpers/ (2 files, typescript)
  src/ (9 files, typescript)
    __mocks__/ (1 files, javascript)
    __serializers__/ (1 files, javascript)
      experiment/ (4 files, typescript)
      filter/ (5 files, bash, typescript)
      job/ (4 files, typescript)
      pipeline/ (4 files, typescript)
      run/ (4 files, typescript)
      visualization/ (5 files, bash, typescript)
      experiment/ (5 files, bash, typescript)
      filter/ (5 files, bash, typescript)
      pipeline/ (5 files, bash, typescript)
      recurringrun/ (5 files, bash, typescript)
      run/ (5 files, bash, typescript)
      visualization/ (5 files, bash, typescript)
    atoms/ (10 files, typescript)
    components/ (35 files, typescript)
      graph/ (5 files, typescript)
      navigators/ (1 files, typescript)
      tabs/ (6 files, typescript)
      viewers/ (12 files, typescript)
    icons/ (7 files, typescript)
    lib/ (21 files, typescript)
      v2/ (5 files, typescript)
    mlmd/ (18 files, typescript)
    pages/ (45 files, typescript)
      functional_components/ (2 files, typescript)
      v2/ (1 files, typescript)
    stories/ (6 files, typescript)
      v2/ (4 files, typescript)
      mlmd/ (3 files, typescript)
hack/ (4 files, bash)
    kubernetesplatform/ (1 files, go)
  proto/ (1 files, protobuf)
  python/ (5 files, bash, python)
    docs/ (1 files, python)
      kubernetes/ (14 files, python)
    deployer/ (3 files, bash)
    hack/ (2 files, bash)
  kustomize/ (2 files, bash)
          pipelines-profile-controller/ (2 files, bash, python)
      platform-agnostic-multi-user-minio/ (1 files, python)
    hack/ (4 files, bash)
      seaweedfs/ (1 files, bash)
  12147-mlmd-removal/ (1 files, sql)
    protos/ (2 files, protobuf)
proxy/ (2 files, bash, python)
    XGBoost/ (1 files, python)
    caching/ (1 files, python)
    condition/ (2 files, python)
    execution_order/ (1 files, python)
    exit_handler/ (1 files, python)
    kubernetes_pvc/ (1 files, python)
    loop_output/ (1 files, python)
    loop_parallelism/ (1 files, python)
    loop_parameter/ (1 files, python)
    loop_static/ (1 files, python)
    output_a_directory/ (1 files, python)
    parallel_join/ (1 files, python)
    recursion/ (1 files, python)
    resource_spec/ (3 files, python)
    retry/ (1 files, python)
    secret/ (1 files, python)
    sequential/ (1 files, python)
      utils/ (1 files, python)
    train_until_good/ (1 files, python)
    DSL - Control structures/ (1 files, python)
    Data passing in python components/ (1 files, python)
  hack/ (1 files, bash)
  python/ (2 files, bash, python)
    kfp/ (2 files, python)
      cli/ (12 files, python)
        diagnose_me/ (5 files, python)
        utils/ (4 files, python)
      client/ (5 files, python)
      compiler/ (4 files, python)
      components/ (2 files, python)
      dsl/ (32 files, python)
        templates/ (2 files, python)
        types/ (5 files, python)
      local/ (18 files, python)
        orchestrator/ (4 files, python)
      registry/ (2 files, python)
        context/ (1 files, python)
      v2/ (4 files, python)
third_party/ (1 files, bash)
  argo/ (1 files, bash)
  metadata_envoy/ (1 files, python)
  minio/ (1 files, bash)
  ml-metadata/ (1 files, bash)
      ml_metadata/ (3 files, go)
      proto/ (2 files, protobuf)
tools/ (1 files, go)
  diagnose/ (1 files, bash)
  k8s-native/ (1 files, python)
  metadatastore-upgrade/ (1 files, go)
```

## Hub Files (most imported)

| File | Imported by |
|------|-------------|
| `tools/diagnose/kfp.sh` | 379 files |
| `backend/src/common/util/time.go` | 150 files |
| `components/google-cloud/google_cloud_pipeline_components/types/artifact_types.py` | 127 files |
| `sdk/python/kfp/local/io.py` | 124 files |
| `backend/api/v1beta1/python_http_client/kfp_server_api/configuration.py` | 110 files |
| `backend/api/v2beta1/python_http_client/kfp_server_api/configuration.py` | 110 files |
| `backend/src/common/util/json.go` | 101 files |
| `frontend/src/__mocks__/typestyle.js` | 89 files |
| `components/aws/sagemaker/common/common_inputs.py` | 47 files |
| `manifests/kustomize/base/installs/multi-user/pipelines-profile-controller/sync.py` | 36 files |
| `manifests/kustomize/env/platform-agnostic-multi-user-minio/sync.py` | 36 files |
| `components/aws/sagemaker/commonv2/common_inputs.py` | 34 files |
| `backend/api/v1beta1/python_http_client/kfp_server_api/exceptions.py` | 30 files |
| `backend/api/v2beta1/python_http_client/kfp_server_api/exceptions.py` | 30 files |
| `components/aws/sagemaker/common/sagemaker_component_spec.py` | 26 files |

## Key Types

| Type | Kind | File | Line | Bases |
|------|------|------|------|-------|
| `API` | interface | `backend/src/v2/apiclient/kfpapi/api.go` | 39 | - |
| `APICronSchedule` | struct | `backend/api/v1beta1/go_http_client/job_model/api_cron_schedule.go` | 20 | - |
| `APIExperiment` | struct | `backend/api/v1beta1/go_http_client/experiment_model/api_experiment.go` | 21 | - |
| `APIGetHealthzResponse` | struct | `backend/api/v1beta1/go_http_client/healthz_model/api_get_healthz_response.go` | 18 | - |
| `APIGetTemplateResponse` | struct | `backend/api/v1beta1/go_http_client/pipeline_model/api_get_template_response.go` | 18 | - |
| `APIJob` | struct | `backend/api/v1beta1/go_http_client/job_model/api_job.go` | 21 | - |
| `APIListExperimentsResponse` | struct | `backend/api/v1beta1/go_http_client/experiment_model/api_list_experiments_response.go` | 20 | - |
| `APIListJobsResponse` | struct | `backend/api/v1beta1/go_http_client/job_model/api_list_jobs_response.go` | 20 | - |
| `APIListPipelineVersionsResponse` | struct | `backend/api/v1beta1/go_http_client/pipeline_model/api_list_pipeline_versions_response.go` | 20 | - |
| `APIListPipelinesResponse` | struct | `backend/api/v1beta1/go_http_client/pipeline_model/api_list_pipelines_response.go` | 20 | - |
| `APIListRunsResponse` | struct | `backend/api/v1beta1/go_http_client/run_model/api_list_runs_response.go` | 20 | - |
| `APIParameter` | struct | `backend/api/v1beta1/go_http_client/job_model/api_parameter.go` | 18 | - |
| `APIParameter` | struct | `backend/api/v1beta1/go_http_client/pipeline_model/api_parameter.go` | 18 | - |
| `APIParameter` | struct | `backend/api/v1beta1/go_http_client/pipeline_upload_model/api_parameter.go` | 18 | - |
| `APIParameter` | struct | `backend/api/v1beta1/go_http_client/run_model/api_parameter.go` | 18 | - |
| `APIPeriodicSchedule` | struct | `backend/api/v1beta1/go_http_client/job_model/api_periodic_schedule.go` | 20 | - |
| `APIPipeline` | struct | `backend/api/v1beta1/go_http_client/pipeline_model/api_pipeline.go` | 21 | - |
| `APIPipeline` | struct | `backend/api/v1beta1/go_http_client/pipeline_upload_model/api_pipeline.go` | 21 | - |
| `APIPipelineRuntime` | struct | `backend/api/v1beta1/go_http_client/run_model/api_pipeline_runtime.go` | 18 | - |
| `APIPipelineSpec` | struct | `backend/api/v1beta1/go_http_client/job_model/api_pipeline_spec.go` | 20 | - |
| `APIPipelineSpec` | struct | `backend/api/v1beta1/go_http_client/run_model/api_pipeline_spec.go` | 20 | - |
| `APIPipelineVersion` | struct | `backend/api/v1beta1/go_http_client/pipeline_model/api_pipeline_version.go` | 21 | - |
| `APIPipelineVersion` | struct | `backend/api/v1beta1/go_http_client/pipeline_upload_model/api_pipeline_version.go` | 21 | - |
| `APIReportRunMetricsResponse` | struct | `backend/api/v1beta1/go_http_client/run_model/api_report_run_metrics_response.go` | 20 | - |
| `APIResourceKey` | struct | `backend/api/v1beta1/go_http_client/experiment_model/api_resource_key.go` | 19 | - |
| `APIResourceKey` | struct | `backend/api/v1beta1/go_http_client/job_model/api_resource_key.go` | 19 | - |
| `APIResourceKey` | struct | `backend/api/v1beta1/go_http_client/pipeline_model/api_resource_key.go` | 19 | - |
| `APIResourceKey` | struct | `backend/api/v1beta1/go_http_client/pipeline_upload_model/api_resource_key.go` | 19 | - |
| `APIResourceKey` | struct | `backend/api/v1beta1/go_http_client/run_model/api_resource_key.go` | 19 | - |
| `APIResourceReference` | struct | `backend/api/v1beta1/go_http_client/experiment_model/api_resource_reference.go` | 19 | - |
| `APIResourceReference` | struct | `backend/api/v1beta1/go_http_client/job_model/api_resource_reference.go` | 19 | - |
| `APIResourceReference` | struct | `backend/api/v1beta1/go_http_client/pipeline_model/api_resource_reference.go` | 19 | - |
| `APIResourceReference` | struct | `backend/api/v1beta1/go_http_client/pipeline_upload_model/api_resource_reference.go` | 19 | - |
| `APIResourceReference` | struct | `backend/api/v1beta1/go_http_client/run_model/api_resource_reference.go` | 19 | - |
| `APIRun` | struct | `backend/api/v1beta1/go_http_client/run_model/api_run.go` | 21 | - |
| `APIRunDetail` | struct | `backend/api/v1beta1/go_http_client/run_model/api_run_detail.go` | 19 | - |
| `APIRunMetric` | struct | `backend/api/v1beta1/go_http_client/run_model/api_run_metric.go` | 19 | - |
| `APIStatus` | struct | `backend/api/v1beta1/go_http_client/pipeline_upload_model/api_status.go` | 20 | - |
| `APITrigger` | struct | `backend/api/v1beta1/go_http_client/job_model/api_trigger.go` | 19 | - |
| `APIURL` | struct | `backend/api/v1beta1/go_http_client/pipeline_model/api_url.go` | 18 | - |
| `APIURL` | struct | `backend/api/v1beta1/go_http_client/pipeline_upload_model/api_url.go` | 18 | - |
| `APIVisualization` | struct | `backend/api/v1beta1/go_http_client/visualization_model/api_visualization.go` | 19 | - |
| `AWSConfigs` | interface | `frontend/server/configs.ts` | 237 | - |
| `AWSInstanceProfileCredentials` | class | `frontend/server/aws-helper.ts` | 61 | - |
| `AWSMetadataCredentials` | interface | `frontend/server/aws-helper.ts` | 17 | - |
| `Affinity` | interface | `frontend/src/third_party/mlmd/kubernetes.ts` | 27 | - |
| `AliasedPluralsGroup` | class | `sdk/python/kfp/cli/utils/aliased_plurals_group.py` | 20 | click.Group |
| `AllExperimentsAndArchive` | class | `frontend/src/pages/AllExperimentsAndArchive.tsx` | 40 | Page |
| `AllExperimentsAndArchiveProps` | interface | `frontend/src/pages/AllExperimentsAndArchive.tsx` | 32 | - |
| `AllExperimentsAndArchiveState` | interface | `frontend/src/pages/AllExperimentsAndArchive.tsx` | 36 | - |

*... and 3142 more types (see tags.json)*


## Inheritance Hierarchy

```
Artifact
  |- ClassificationMetrics
  |- Dataset
  |- HTML
  |- Markdown
  |- Metrics
  |- Model
  |- SlicedClassificationMetrics
BaseAPI
  |- ExperimentServiceApi
  |- JobServiceApi
  |- PipelineServiceApi
  |- RecurringRunServiceApi
  |- RunServiceApi
  |- VisualizationServiceApi
BaseComponent
  |- MarGeneration
  |- MinIO
  |- Trainer
  |- Visualization
BaseExecutor
  |- Executor
  |- GenericExecutor
Enum
  |- DebugRulesStatus
  |- TemplateType
Error
  |- RequiredError
ModelBase
  |- AndPredicate
  |- BinaryPredicate
    |- EqualsPredicate
    |- GreaterThanOrEqualPredicate
    |- GreaterThanPredicate
    |- LessThenOrEqualPredicate
    |- LessThenPredicate
    |- NotEqualsPredicate
  |- CachingStrategySpec
  |- ComponentReference
  |- ComponentSpec
  |- ConcatPlaceholder
  |- ContainerImplementation
  |- ContainerSpec
  |- ExecutionOptionsSpec
  |- ExecutorInputPlaceholder
  |- GraphImplementation
  |- GraphInputArgument
  |- GraphInputReference
  |- GraphSpec
  |- IfPlaceholder
  |- IfPlaceholderStructure
  |- InputMetadataPlaceholder
  |- InputOutputPortNamePlaceholder
  |- InputPathPlaceholder
  |- InputSpec
  |- InputUriPlaceholder
  |- InputValuePlaceholder
  |- IsPresentPlaceholder
  |- MetadataSpec
  |- NotPredicate
  |- OrPredicate
  |- OutputMetadataPlaceholder
  |- OutputPathPlaceholder
  |- OutputSpec
  |- OutputUriPlaceholder
  |- PipelineRunSpec
  |- RetryStrategySpec
  |- TaskOutputArgument
  |- TaskOutputReference
  |- TaskSpec
  |- TwoBooleanOperands
  |- TwoOperands
OpenApiException
  |- ApiException
  |- ApiKeyError
  |- ApiTypeError
  |- ApiValueError
Page
  |- AllExperimentsAndArchive
  |- AllRecurringRunsList
  |- AllRunsAndArchive
  |- AllRunsList
  |- ArchivedExperiments
  |- ArchivedRuns
  |- ArtifactDetails
  |- ArtifactList
  |- CompareV1
  |- ExecutionList
  |- ExperimentDetails
  |- ExperimentList
  |- GettingStarted
  |- NewExperiment
  |- NewPipelineVersion
  |- NewRun
  |- PipelineDetails
  |- PipelineList
  |- RecurringRunDetails
  |- RecurringRunDetailsV2
  |- RunDetails
PipelineChannel
  |- OneOfMixin
    |- OneOfArtifact
    |- OneOfParameter
  |- PipelineArtifactChannel
  |- PipelineParameterChannel
```

## Key Functions

| Function | File | Line | Params |
|----------|------|------|--------|
| `Accept` | `backend/src/v2/compiler/visitor.go` | 57 | *pipelinespec.PipelineJob, *pipelinespec.SinglePlatformSpec, Visitor |
| `AddRuntimeMetadata` | `backend/src/apiserver/template/argo_template.go` | 231 | *workflowapi.Workflow |
| `AdmitFuncHandler` | `backend/src/cache/server/admission.go` | 146 | admitFunc, ClientManagerInterface |
| `AnyStringPtr` | `backend/src/common/util/pointer.go` | 119 | interface{} |
| `ArchiveTgz` | `backend/src/common/util/tgz.go` | 30 | map[string]string |
| `ArtifactListSwitcher` | `frontend/src/pages/ArtifactListSwitcher.tsx` | 24 | PageProps |
| `ArtifactNode` | `frontend/src/components/graph/ArtifactNode.tsx` | 31 | id, ArtifactNodeProps |
| `ArtifactNodeDetail` | `frontend/src/components/tabs/RuntimeNodeDetailsV2.tsx` | 342 | execution, linkedArtifact, ArtifactNodeDetailProps |
| `ArtifactNodeDetail` | `frontend/src/components/tabs/StaticNodeDetailsV2.tsx` | 194 | pipelineSpec, element, ArtifactNodeDetailProps |
| `ArtifactTitle` | `frontend/src/components/tabs/ArtifactTitle.tsx` | 28 | ArtifactTitleProps |
| `BoolNilOrValue` | `backend/src/common/util/pointer.go` | 78 | *bool |
| `BoolPointer` | `backend/src/common/util/pointer.go` | 30 | bool |
| `BooleanPointer` | `backend/src/common/util/pointer.go` | 87 | bool |
| `ChevronRightSmallIcon` | `frontend/src/components/TwoLevelDropdown.tsx` | 78 | - |
| `CollapseButtonSingle` | `frontend/src/components/CollapseButtonSingle.tsx` | 45 | CollapseButtonSingleProps |
| `CompareTableSection` | `frontend/src/pages/CompareV2.tsx` | 194 | CompareTableSectionParams |
| `CompareV2` | `frontend/src/pages/CompareV2.tsx` | 238 | CompareV2Props |
| `Compile` | `backend/src/v2/compiler/argocompiler/argo.go` | 57 | *pipelinespec.PipelineJob, *pipelinespec.SinglePlatformSpec, *Options |
| `ComponentMetadata` | `components/aws/sagemaker/common/sagemaker_component.py` | 43 | name, description, spec |
| `ComponentMetadata` | `components/aws/sagemaker/commonv2/sagemaker_component.py` | 45 | name, description, spec |
| `Concat` | `components/google-cloud/google_cloud_pipeline_components/preview/automl/vision/json_utils.py` | 125 | - |
| `ConfigureCustomCABundle` | `backend/src/v2/compiler/argocompiler/common.go` | 88 | *wfapi.Template |
| `ConfusionMatrixSection` | `frontend/src/components/viewers/MetricsVisualizations.tsx` | 742 | ConfusionMatrixProps |
| `Container` | `backend/src/v2/driver/container.go` | 38 | context.Context, common.Options, client_manager.ClientManagerInterface |
| `CopyThisBinary` | `backend/src/v2/component/util.go` | 30 | string |
| `CreateArtifactPath` | `backend/src/apiserver/common/utils.go` | 39 | string, string, string |
| `CreateErrorCouldNotRecoverAPIStatus` | `backend/src/common/client/api_server/util.go` | 141 | error |
| `CreateErrorFromAPIStatus` | `backend/src/common/client/api_server/util.go` | 137 | string, int32 |
| `CreateKubernetesCoreOrFatal` | `backend/src/apiserver/client/kubernetes_core.go` | 43 | time.Duration, util.ClientParameters |
| `CreateKubernetesCoreOrFatal` | `backend/src/cache/client/kubernetes_core.go` | 42 | time.Duration, util.ClientParameters |
| `CreatePVC` | `kubernetes_platform/python/kfp/kubernetes/volume.py` | 26 | name, access_modes, size, pvc_name, pvc_name_suffix, ... |
| `CreateSubjectAccessReviewClientOrFatal` | `backend/src/apiserver/client/subject_access_review.go` | 41 | time.Duration, util.ClientParameters |
| `CreateTokenReviewClientOrFatal` | `backend/src/apiserver/client/token_review.go` | 41 | time.Duration, util.ClientParameters |
| `CurrentExecutionType` | `backend/src/common/util/execution_spec.go` | 52 | - |
| `CustomMarshaler` | `backend/src/apiserver/common/utils.go` | 153 | - |
| `DAG` | `backend/src/v2/driver/dag.go` | 33 | context.Context, common.Options, client_manager.ClientManagerInterface |
| `DateTimePointer` | `backend/src/common/util/pointer.go` | 38 | strfmt.DateTime |
| `DecompressPipelineTarball` | `backend/src/apiserver/server/util.go` | 76 | []byte |
| `DecompressPipelineZip` | `backend/src/apiserver/server/util.go` | 121 | []byte |
| `DefaultTransportConfig` | `backend/api/v1beta1/go_http_client/experiment_client/experiment_client.go` | 64 | - |
| `DefaultTransportConfig` | `backend/api/v1beta1/go_http_client/healthz_client/healthz_client.go` | 64 | - |
| `DefaultTransportConfig` | `backend/api/v1beta1/go_http_client/job_client/job_client.go` | 64 | - |
| `DefaultTransportConfig` | `backend/api/v1beta1/go_http_client/pipeline_client/pipeline_client.go` | 64 | - |
| `DefaultTransportConfig` | `backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_client.go` | 64 | - |
| `DefaultTransportConfig` | `backend/api/v1beta1/go_http_client/run_client/run_client.go` | 64 | - |
| `DefaultTransportConfig` | `backend/api/v1beta1/go_http_client/visualization_client/visualization_client.go` | 64 | - |
| `DefaultTransportConfig` | `backend/api/v2beta1/go_http_client/artifact_client/artifact_client.go` | 64 | - |
| `DefaultTransportConfig` | `backend/api/v2beta1/go_http_client/experiment_client/experiment_client.go` | 64 | - |
| `DefaultTransportConfig` | `backend/api/v2beta1/go_http_client/healthz_client/healthz_client.go` | 64 | - |
| `DefaultTransportConfig` | `backend/api/v2beta1/go_http_client/pipeline_client/pipeline_client.go` | 64 | - |

*... and 2296 more functions (see tags.json)*


## Module Dependencies

```
api/v2alpha1/go/cachekey -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio
api/v2alpha1/go/pipelinespec -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio
api/v2alpha1/python -> tools
backend/api/v1beta1/go_client -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/experiment_client/experiment_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/healthz_client/healthz_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/job_client/job_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/run_client/run_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/visualization_client/visualization_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/python_http_client -> tools
backend/api/v1beta1/python_http_client/kfp_server_api -> backend/api/v1beta1/python_http_client/kfp_server_api/api, backend/api/v1beta1/python_http_client/kfp_server_api/models, backend/api/v2beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api/api, backend/api/v2beta1/python_http_client/kfp_server_api/models, backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/python_http_client/kfp_server_api/api -> backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api/api
backend/api/v1beta1/python_http_client/kfp_server_api/models -> backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api/models
backend/api/v2beta1/go_client -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/artifact_client/artifact_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/experiment_client/experiment_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/healthz_client/healthz_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/run_client/run_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/visualization_client/visualization_service -> backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/python_http_client -> tools
backend/api/v2beta1/python_http_client/kfp_server_api -> backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v1beta1/python_http_client/kfp_server_api/api, backend/api/v1beta1/python_http_client/kfp_server_api/models, backend/api/v2beta1/python_http_client/kfp_server_api/api, backend/api/v2beta1/python_http_client/kfp_server_api/models, backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/python_http_client/kfp_server_api/api -> backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v1beta1/python_http_client/kfp_server_api/api, backend/api/v2beta1/python_http_client/kfp_server_api
backend/api/v2beta1/python_http_client/kfp_server_api/models -> backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v1beta1/python_http_client/kfp_server_api/models, backend/api/v2beta1/python_http_client/kfp_server_api
backend/src/agent/persistence -> backend/src/common/util
backend/src/agent/persistence/client -> backend/src/common/util
backend/src/agent/persistence/client/artifactclient -> backend/src/common/util, sdk/python/kfp/local
backend/src/agent/persistence/client/tokenrefresher -> backend/src/common/util, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio
backend/src/agent/persistence/worker -> backend/src/common/util
backend/src/apiserver -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio, sdk/python/kfp/local
backend/src/apiserver/archive -> backend/src/common/util, sdk/python/kfp/local
backend/src/apiserver/client -> backend/src/common/util
backend/src/apiserver/client_manager -> backend/src/common/util, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio
backend/src/apiserver/common -> backend/src/common/util
backend/src/apiserver/config -> backend/src/common/util
backend/src/apiserver/resource -> sdk/python/kfp/local
backend/src/apiserver/server -> backend/src/common/util, sdk/python/kfp/local
backend/src/apiserver/storage -> backend/src/common/util, sdk/python/kfp/local
backend/src/apiserver/template -> backend/src/common/util, sdk/python/kfp/local
backend/src/apiserver/visualization -> backend/src/common/util
backend/src/apiserver/visualization/types -> backend/src/common/util
backend/src/cache -> backend/src/apiserver/archive, backend/src/common/util
backend/src/cache/client -> backend/src/common/util
backend/src/cache/server -> backend/src/apiserver/archive, backend/src/common/util, sdk/python/kfp/local
backend/src/cache/storage -> backend/src/apiserver/archive
backend/src/common/client/api_server -> backend/src/common/util
backend/src/common/client/api_server/v2 -> sdk/python/kfp/local
backend/src/common/util -> sdk/python/kfp/local
backend/src/crd/controller/scheduledworkflow -> backend/src/common/util
backend/src/crd/controller/scheduledworkflow/client -> backend/src/common/util
backend/src/crd/controller/scheduledworkflow/util -> backend/src/common/util
backend/src/crd/controller/viewer -> backend/src/apiserver/archive
backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1 -> backend/src/common/util
backend/src/crd/pkg/client/informers/externalversions -> backend/src/common/util, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio
backend/src/crd/pkg/client/informers/externalversions/internalinterfaces -> backend/src/common/util
backend/src/crd/pkg/client/informers/externalversions/scheduledworkflow/v1beta1 -> backend/src/common/util
backend/src/v2/apiclient -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio
backend/src/v2/apiclient/kfpapi -> backend/src/common/util
backend/src/v2/compiler/testdata -> tools/diagnose
backend/src/v2/component -> backend/src/common/util, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio, sdk/python/kfp/local
backend/src/v2/objectstore -> sdk/python/kfp/local
components/PyTorch/_samples -> tools/diagnose
components/PyTorch/pytorch-kfp-components -> backend/src/common, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, tools
components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/mar -> components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/base
components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/minio -> backend/src/apiserver/client, backend/src/v2/config, components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/base
components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/trainer -> components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/base
components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/visualization -> backend/src/common/util, components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/base, components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/minio
components/aws/athena/query/src -> backend/src/common/util
components/aws/emr/common -> backend/src/common/util
components/aws/emr/create_cluster/src -> backend/src/apiserver/model, backend/src/v2/compiler/argocompiler, kubernetes_platform/python/kfp/kubernetes
components/aws/emr/delete_cluster/src -> backend/src/apiserver/model, backend/src/v2/compiler/argocompiler, kubernetes_platform/python/kfp/kubernetes
components/aws/emr/submit_pyspark_job/src -> backend/src/apiserver/model, backend/src/v2/compiler/argocompiler, kubernetes_platform/python/kfp/kubernetes
components/aws/emr/submit_spark_job/src -> backend/src/apiserver/model, backend/src/v2/compiler/argocompiler, kubernetes_platform/python/kfp/kubernetes
components/aws/sagemaker/DataQualityJobDefinition/src -> backend/src/common/util, components/aws/sagemaker/commonv2
components/aws/sagemaker/Endpoint/src -> backend/src/common/util, components/aws/sagemaker/commonv2
components/aws/sagemaker/EndpointConfig/src -> backend/src/common/util, components/aws/sagemaker/commonv2
components/aws/sagemaker/ModelBiasJobDefinition/src -> backend/src/common/util, components/aws/sagemaker/commonv2
components/aws/sagemaker/ModelExplainabilityJobDefinition/src -> backend/src/common/util, components/aws/sagemaker/commonv2
components/aws/sagemaker/ModelQualityJobDefinition/src -> backend/src/common/util, components/aws/sagemaker/commonv2
components/aws/sagemaker/Modelv2/src -> backend/src/common/util, components/aws/sagemaker/commonv2
components/aws/sagemaker/MonitoringSchedule/src -> backend/src/common/util, components/aws/sagemaker/commonv2
components/aws/sagemaker/TrainingJob/samples/mnist-kmeans-training -> sdk/python/kfp/local, tools/diagnose
components/aws/sagemaker/TrainingJob/src -> backend/src/common/util, components/aws/sagemaker/commonv2
components/aws/sagemaker/batch_transform/src -> components/aws/sagemaker/common
components/aws/sagemaker/common -> backend/src/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, components/aws/sagemaker/commonv2
components/aws/sagemaker/commonv2 -> backend/src/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, frontend/src/third_party/mlmd
components/aws/sagemaker/create_simulation_app/src -> components/aws/sagemaker/common
components/aws/sagemaker/delete_simulation_app/src -> components/aws/sagemaker/common
components/aws/sagemaker/deploy/src -> components/aws/sagemaker/common
components/aws/sagemaker/ground_truth/src -> components/aws/sagemaker/common
components/aws/sagemaker/hyperparameter_tuning/src -> components/aws/sagemaker/common, components/aws/sagemaker/train/src
components/aws/sagemaker/model/src -> components/aws/sagemaker/common
components/aws/sagemaker/process/src -> components/aws/sagemaker/common
components/aws/sagemaker/rlestimator/src -> components/aws/sagemaker/common
components/aws/sagemaker/simulation_job/src -> components/aws/sagemaker/common
components/aws/sagemaker/simulation_job_batch/src -> components/aws/sagemaker/common
components/aws/sagemaker/train/src -> components/aws/sagemaker/common
components/aws/sagemaker/workteam/src -> components/aws/sagemaker/common
components/google-cloud -> backend/src/common, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, tools
components/google-cloud/docs/source -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components -> backend/src/common/util, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/llm -> backend/src/common/util, tools, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model -> components/google-cloud/google_cloud_pipeline_components/_implementation/model/get_model
components/google-cloud/google_cloud_pipeline_components/_implementation/model/get_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation -> components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/chunking, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/data_sampler, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/dataset_preprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/endpoint_batch_predict, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/error_analysis_annotation, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/evaluated_annotation, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/feature_attribution, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/feature_extractor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/import_evaluated_annotation, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_classification_postprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_embedding_retrieval, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_evaluation, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_evaluation_preprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_information_retrieval_preprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_retrieval_metrics, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_safety_bias, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/model_name_preprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/target_field_data_remover
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/chunking -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/dataset_preprocessor -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/endpoint_batch_predict -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/evaluated_annotation -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/feature_attribution -> components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/data_sampler, components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/feature_extractor -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/import_evaluated_annotation -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_classification_postprocessor -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_embedding -> components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/endpoint_batch_predict, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_embedding_retrieval, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_information_retrieval_preprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_retrieval_metrics, components/google-cloud/google_cloud_pipeline_components/preview/model_evaluation, components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_evaluation -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_evaluation_preprocessor -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_safety_bias -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/text2sql -> components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/endpoint_batch_predict, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/text2sql_evaluation, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/text2sql_preprocess, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/text2sql_validate_and_process, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net -> components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/dataprep, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/evaluation, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/get_training_artifacts, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/maybe_set_tfrecord_args, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_dataprep_args, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_eval_args, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_tfrecord_args, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_train_args, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/train, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/upload_decomposition_plots, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/upload_model
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/dataprep -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/evaluation -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/get_training_artifacts -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/maybe_set_tfrecord_args -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_dataprep_args -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_eval_args -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_tfrecord_args -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_train_args -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/train -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/upload_decomposition_plots -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/upload_model -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/container/_implementation/llm/templated_custom_job -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/_implementation/model/get_model -> components/google-cloud/google_cloud_pipeline_components/proto, components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/container/_implementation/model_evaluation -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/proto
components/google-cloud/google_cloud_pipeline_components/container/preview/custom_job -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/preview/dataflow/flex_template -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/preview/gcp_launcher -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/proto
components/google-cloud/google_cloud_pipeline_components/container/utils -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/aiplatform -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/automl_training_job/image -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/batch_prediction_job -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/create_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/detect_anomalies_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/drop_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/evaluate_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/explain_forecast_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/explain_predict_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/export_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/feature_importance -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/forecast_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/global_explain -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_advanced_weights -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_arima_coefficients -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_arima_evaluate -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_centroids -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_confusion_matrix -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_feature_info -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_principal_component_info -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_principal_components -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_recommend -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_reconstruction_loss -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_roc_curve -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_training_info -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_trial_info -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/ml_weights -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/predict_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/bigquery/utils -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/container/v1/custom_job -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/dataflow -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/dataproc/utils -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/endpoint/create_endpoint -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/container/v1/endpoint/delete_endpoint -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/endpoint/deploy_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/endpoint/undeploy_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/gcp_launcher -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/proto
components/google-cloud/google_cloud_pipeline_components/container/v1/gcp_launcher/utils -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/hyperparameter_tuning_job -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/infra_validation_job -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/container/v1/custom_job
components/google-cloud/google_cloud_pipeline_components/container/v1/model/delete_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/model/export_model -> backend/src/common/util
components/google-cloud/google_cloud_pipeline_components/container/v1/model/upload_model -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/container/v1/wait_gcp_resources -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/proto, tools
components/google-cloud/google_cloud_pipeline_components/preview/automl/forecasting -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/preview/automl/tabular -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/preview/automl/vision -> backend/src/common/util, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/preview/custom_job -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/preview/llm -> components/google-cloud/google_cloud_pipeline_components/preview/llm/infer, components/google-cloud/google_cloud_pipeline_components/preview/llm/rlaif, components/google-cloud/google_cloud_pipeline_components/preview/llm/rlhf
components/google-cloud/google_cloud_pipeline_components/preview/llm/infer -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/preview/llm/rlaif -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/preview/llm/rlhf -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/preview/model_evaluation -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/preview/starry_net -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/types -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/automl/forecasting -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/automl/tabular -> backend/src/common/util, components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job -> components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job/automl_forecasting_training_job, components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job/automl_image_training_job, components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job/automl_tabular_training_job, components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job/automl_text_training_job, components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job/automl_video_training_job
components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job/automl_forecasting_training_job -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job/automl_image_training_job -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job/automl_tabular_training_job -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job/automl_text_training_job -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/automl/training_job/automl_video_training_job -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/batch_predict_job -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery -> components/google-cloud/google_cloud_pipeline_components/v1/bigquery/create_model, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/detect_anomalies_model, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/drop_model, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/evaluate_model, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/explain_forecast_model, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/explain_predict_model, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/export_model, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/feature_importance, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/forecast_model, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/global_explain, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_advanced_weights, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_arima_coefficients, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_arima_evaluate, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_centroids, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_confusion_matrix, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_feature_info, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_principal_component_info, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_principal_components, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_recommend, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_reconstruction_loss, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_roc_curve, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_training_info, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_trial_info, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_weights, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/predict_model, components/google-cloud/google_cloud_pipeline_components/v1/bigquery/query_job
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/create_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/detect_anomalies_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/drop_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/evaluate_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/explain_forecast_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/explain_predict_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/export_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/feature_importance -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/forecast_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/global_explain -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_advanced_weights -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_arima_coefficients -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_arima_evaluate -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_centroids -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_confusion_matrix -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_feature_info -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_principal_component_info -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_principal_components -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_recommend -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_reconstruction_loss -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_roc_curve -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_training_info -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_trial_info -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/ml_weights -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/predict_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/bigquery/query_job -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/custom_job -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataflow -> components/google-cloud/google_cloud_pipeline_components/v1/dataflow/flex_template, components/google-cloud/google_cloud_pipeline_components/v1/dataflow/python_job
components/google-cloud/google_cloud_pipeline_components/v1/dataproc -> components/google-cloud/google_cloud_pipeline_components/v1/dataproc/create_pyspark_batch, components/google-cloud/google_cloud_pipeline_components/v1/dataproc/create_spark_batch, components/google-cloud/google_cloud_pipeline_components/v1/dataproc/create_spark_r_batch, components/google-cloud/google_cloud_pipeline_components/v1/dataproc/create_spark_sql_batch
components/google-cloud/google_cloud_pipeline_components/v1/dataset -> components/google-cloud/google_cloud_pipeline_components/v1/dataset/create_image_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/create_tabular_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/create_text_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/create_time_series_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/create_video_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/export_image_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/export_tabular_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/export_text_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/export_time_series_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/export_video_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/get_vertex_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/import_image_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/import_text_dataset, components/google-cloud/google_cloud_pipeline_components/v1/dataset/import_video_dataset
components/google-cloud/google_cloud_pipeline_components/v1/dataset/create_image_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/create_tabular_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/create_text_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/create_time_series_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/create_video_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/export_image_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/export_tabular_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/export_text_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/export_time_series_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/export_video_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/get_vertex_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/import_image_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/import_text_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/dataset/import_video_dataset -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/endpoint -> components/google-cloud/google_cloud_pipeline_components/v1/endpoint/create_endpoint, components/google-cloud/google_cloud_pipeline_components/v1/endpoint/delete_endpoint, components/google-cloud/google_cloud_pipeline_components/v1/endpoint/deploy_model, components/google-cloud/google_cloud_pipeline_components/v1/endpoint/undeploy_model
components/google-cloud/google_cloud_pipeline_components/v1/endpoint/create_endpoint -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/endpoint/delete_endpoint -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/endpoint/deploy_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/endpoint/undeploy_model -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/forecasting -> components/google-cloud/google_cloud_pipeline_components/v1/forecasting/prepare_data_for_train, components/google-cloud/google_cloud_pipeline_components/v1/forecasting/preprocess, components/google-cloud/google_cloud_pipeline_components/v1/forecasting/validate
components/google-cloud/google_cloud_pipeline_components/v1/forecasting/prepare_data_for_train -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/forecasting/preprocess -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/forecasting/validate -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/model -> components/google-cloud/google_cloud_pipeline_components/v1/model/delete_model, components/google-cloud/google_cloud_pipeline_components/v1/model/export_model, components/google-cloud/google_cloud_pipeline_components/v1/model/get_model, components/google-cloud/google_cloud_pipeline_components/v1/model/upload_model
components/google-cloud/google_cloud_pipeline_components/v1/model/delete_model -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/model/export_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/v1/model/get_model -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/model/upload_model -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/model_evaluation -> components/google-cloud/google_cloud_pipeline_components/preview/model_evaluation, components/google-cloud/google_cloud_pipeline_components/types, components/google-cloud/google_cloud_pipeline_components/v1/model_evaluation/model_based_llm_evaluation/autosxs, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/model_evaluation/model_based_llm_evaluation -> components/google-cloud/google_cloud_pipeline_components/v1/model_evaluation/model_based_llm_evaluation/autosxs
components/google-cloud/google_cloud_pipeline_components/v1/model_evaluation/model_based_llm_evaluation/autosxs -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/v1/wait_gcp_resources -> tools/diagnose
components/kserve/src -> backend/src/common/util, frontend/src/third_party/mlmd
components/kubeflow/pytorch-launcher -> backend/src/common/util, frontend/src/third_party/mlmd, tools/diagnose
components/snowflake -> tools/diagnose
docs/sdk -> tools
frontend/server -> backend/src/apiserver/client, backend/src/v2/config
frontend/server/handlers -> backend/src/apiserver/client, backend/src/v2/config
frontend/src -> frontend/src/__mocks__
frontend/src/atoms -> frontend/src/__mocks__
frontend/src/components -> frontend/src/__mocks__, tools
frontend/src/components/graph -> frontend/src/__mocks__
frontend/src/components/viewers -> frontend/src/__mocks__, tools
frontend/src/lib -> frontend/src/__mocks__
frontend/src/lib/v2 -> frontend/src/__mocks__
frontend/src/mlmd -> frontend/src/__mocks__
frontend/src/pages -> frontend/src/__mocks__, tools
frontend/src/pages/functional_components -> frontend/src/__mocks__
kubernetes_platform/go/kubernetesplatform -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio
kubernetes_platform/python -> tools
kubernetes_platform/python/docs -> tools/diagnose
kubernetes_platform/python/kfp/kubernetes -> frontend/src/third_party/mlmd, sdk/python/kfp/compiler, tools/diagnose
manifests/kustomize/base/installs/multi-user/pipelines-profile-controller -> backend/src/common/util
manifests/kustomize/env/platform-agnostic-multi-user-minio -> backend/src/common/util
proxy -> backend/src/common/util
samples/core/XGBoost -> tools/diagnose
samples/core/caching -> backend/src/common/util, tools/diagnose
samples/core/condition -> tools/diagnose
samples/core/execution_order -> tools/diagnose
samples/core/exit_handler -> tools/diagnose
samples/core/kubernetes_pvc -> tools/diagnose
samples/core/loop_output -> tools/diagnose
samples/core/loop_parallelism -> tools/diagnose
samples/core/loop_parameter -> tools/diagnose
samples/core/loop_static -> tools/diagnose
samples/core/parallel_join -> tools/diagnose
samples/core/recursion -> tools/diagnose
samples/core/resource_spec -> tools/diagnose
samples/core/retry -> tools/diagnose
samples/core/secret -> tools/diagnose
samples/core/sequential -> tools/diagnose
samples/core/train_until_good -> tools/diagnose
samples/tutorials/DSL - Control structures -> tools/diagnose
samples/tutorials/Data passing in python components -> tools/diagnose
sdk/python -> tools
sdk/python/kfp/cli -> backend/src/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, sdk/python/kfp/compiler, sdk/python/kfp/dsl, tools, tools/diagnose
sdk/python/kfp/cli/diagnose_me -> backend/src/common/util
sdk/python/kfp/client -> backend/src/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, tools/diagnose
sdk/python/kfp/compiler -> backend/src/common/util, sdk/python/kfp/dsl, tools/diagnose
sdk/python/kfp/components -> sdk/python/kfp/dsl
sdk/python/kfp/dsl -> backend/src/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, sdk/python/kfp/dsl/types, sdk/python/kfp/local, tools, tools/diagnose
sdk/python/kfp/dsl/types -> backend/src/common/util, sdk/python/kfp/dsl, tools/diagnose
sdk/python/kfp/local -> api/v2alpha1/google/protobuf, backend/src/common/util, tools, tools/diagnose
sdk/python/kfp/registry -> backend/src/common/util
sdk/python/kfp/v2 -> tools/diagnose
third_party/metadata_envoy -> backend/src/common/util
third_party/ml-metadata/go/ml_metadata -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, manifests/kustomize/env/platform-agnostic-multi-user-minio
tools/metadatastore-upgrade -> backend/src/apiserver/archive, backend/src/common/util
```
