# Codebase Knowledge Graph

Generated: 2026-03-18 20:23 UTC
Parser: AST + Regex

## Statistics

- **2922** files (876 test files) across **7** languages
- **23413** symbols extracted
  - 11539 methods
  - 7150 functions
  - 1789 class
  - 1285 structs
  - 1070 interfaces
  - 490 constants
  - 90 enums
- **92560** edges:
  - 39049 imports
  - 23413 contains
  - 20986 calls
  - 8236 tests
  - 876 inherits
- **935/2046** source files have test coverage

## Languages

| Language | Files |
|----------|-------|
| python | 1344 |
| go | 799 |
| typescript | 562 |
| bash | 100 |
| protobuf | 90 |
| javascript | 25 |
| sql | 2 |

## Directory Structure

```
    github-disk-cleanup/ (1 files, bash)
    junit-summary/ (1 files, python)
    scripts/ (4 files, bash)
      kfp-readiness/ (1 files, python)
    squid/ (1 files, bash)
  scripts/ (1 files, python)
  v2alpha1/ (2 files, protobuf)
      cachekey/ (1 files, go)
      pipelinespec/ (1 files, go)
      protobuf/ (54 files, 37 tests, protobuf)
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
        test/ (44 files, 44 tests, python)
    v2beta1/ (10 files, protobuf)
      go_client/ (27 files, go)
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
        recurring_run_model/ (13 files, go)
        run_client/ (1 files, go)
          run_service/ (17 files, go)
        run_model/ (15 files, go)
        visualization_client/ (1 files, go)
          visualization_service/ (3 files, go)
        visualization_model/ (4 files, go)
        rpc/ (1 files, protobuf)
      python_http_client/ (2 files, bash, python)
        kfp_server_api/ (5 files, python)
          api/ (10 files, python)
          models/ (44 files, python)
        test/ (52 files, 52 tests, python)
  conformance/ (1 files, bash)
  metadata_writer/ (1 files, bash)
    src/ (2 files, python)
      persistence/ (2 files, go)
        client/ (6 files, go)
          artifactclient/ (3 files, 1 tests, go)
          tokenrefresher/ (2 files, 1 tests, go)
        worker/ (8 files, 4 tests, go)
    apiserver/ (3 files, 1 tests, go)
      archive/ (2 files, 1 tests, go)
      auth/ (7 files, 3 tests, go)
      client/ (25 files, 9 tests, go)
      client_manager/ (4 files, 2 tests, go)
      common/ (6 files, 2 tests, go)
      config/ (2 files, 1 tests, go)
        proxy/ (2 files, 1 tests, go)
      filter/ (2 files, 1 tests, go)
      list/ (2 files, 1 tests, go)
      model/ (15 files, go)
      resource/ (6 files, 3 tests, go)
      server/ (31 files, 16 tests, go)
      storage/ (26 files, 12 tests, go)
      template/ (4 files, 1 tests, go)
      validation/ (4 files, 2 tests, go)
      visualization/ (5 files, 2 tests, bash, python)
        snapshots/ (1 files, 1 tests, python)
        types/ (5 files, python)
      webhook/ (2 files, 1 tests, go)
    cache/ (2 files, go)
      client/ (7 files, 3 tests, go)
      deployer/ (3 files, bash)
      model/ (1 files, go)
      server/ (6 files, 2 tests, go)
      storage/ (4 files, 1 tests, go)
    common/ (1 files, go)
        api_server/ (1 files, go)
          v1/ (13 files, go)
          v2/ (12 files, go)
      util/ (41 files, 19 tests, go)
        scheduledworkflow/ (2 files, go)
          client/ (4 files, 1 tests, go)
          util/ (9 files, 4 tests, go)
        viewer/ (1 files, go)
          reconciler/ (2 files, 1 tests, go)
      hack/ (2 files, bash)
        v2beta1/ (5 files, 1 tests, go)
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
      cacheutils/ (2 files, 1 tests, go)
      client_manager/ (2 files, go)
        compiler/ (1 files, go)
        driver/ (3 files, 1 tests, go)
        launcher-v2/ (1 files, go)
      compiler/ (1 files, go)
        argocompiler/ (14 files, 6 tests, go)
        testdata/ (1 files, python)
      component/ (5 files, 1 tests, go)
      config/ (5 files, 1 tests, go)
      driver/ (16 files, 8 tests, go)
      expression/ (2 files, 1 tests, go)
      metadata/ (6 files, 1 tests, go)
        testutils/ (1 files, 1 tests, go)
      objectstore/ (3 files, 1 tests, go)
      test/ (1 files, 1 tests, bash)
        scripts/ (2 files, 2 tests, bash)
  test/ (1 files, 1 tests, go)
    compiler/ (3 files, 3 tests, go)
      matchers/ (1 files, 1 tests, go)
      utils/ (1 files, 1 tests, go)
    config/ (1 files, 1 tests, go)
    constants/ (3 files, 3 tests, go)
    end2end/ (2 files, 2 tests, go)
      utils/ (1 files, 1 tests, go)
    initialization/ (2 files, 2 tests, go)
    integration/ (13 files, 13 tests, bash, go)
    logger/ (1 files, 1 tests, go)
    proto_tests/ (3 files, 3 tests, go)
    resources/ (1 files, 1 tests, python)
    testutil/ (10 files, 10 tests, go)
    v2/ (1 files, 1 tests, go)
      api/ (9 files, 9 tests, go)
        matcher/ (2 files, 2 tests, go)
      integration/ (8 files, 8 tests, go)
components/ (2 files, 1 tests, bash)
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
      tests/ (7 files, 7 tests, bash, python)
        iris/ (4 files, 4 tests, python)
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
        integration_tests/ (1 files, 1 tests, python)
          component_tests/ (14 files, 14 tests, python)
            definition/ (22 files, 22 tests, python)
          scripts/ (1 files, 1 tests, python)
          utils/ (8 files, 8 tests, python)
        unit_tests/ (4 files, 4 tests, bash)
            MonitoringSchedule/ (2 files, 2 tests, python)
            batch_transform/ (2 files, 2 tests, python)
            common/ (6 files, 6 tests, python)
            deploy/ (2 files, 2 tests, python)
            ground_truth/ (2 files, 2 tests, python)
            hyperparameter_tuning/ (2 files, 2 tests, python)
            model/ (2 files, 2 tests, python)
            process/ (2 files, 2 tests, python)
            rlestimator/ (2 files, 2 tests, python)
            robomaker/ (8 files, 8 tests, python)
            train/ (2 files, 2 tests, python)
            workteam/ (2 files, 2 tests, python)
        src/ (3 files, python)
        src/ (2 files, python)
  google-cloud/ (1 files, python)
    docs/ (1 files, bash)
      source/ (1 files, python)
    google_cloud_pipeline_components/ (5 files, python)
      _implementation/ (1 files, python)
        llm/ (25 files, 1 tests, python)
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
          set_test_set/ (2 files, 2 tests, python)
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
          tabular/ (8 files, python)
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
frontend/ (3 files, javascript, typescript)
  .storybook/ (2 files, typescript)
  mock-backend/ (4 files, typescript)
        runtime/ (9 files, typescript)
  scripts/ (15 files, 2 tests, bash, javascript, typescript)
    ui-smoke-test/ (8 files, 8 tests, javascript)
  server/ (21 files, 8 tests, typescript)
    handlers/ (10 files, 1 tests, typescript)
    helpers/ (2 files, typescript)
    integration-tests/ (4 files, 4 tests, typescript)
  src/ (11 files, 1 tests, typescript)
    __mocks__/ (1 files, javascript)
      experiment/ (2 files, typescript)
        apis/ (2 files, typescript)
        models/ (10 files, typescript)
      filter/ (2 files, typescript)
        models/ (9 files, typescript)
      job/ (2 files, typescript)
        apis/ (2 files, typescript)
        models/ (17 files, typescript)
      pipeline/ (2 files, typescript)
        apis/ (2 files, typescript)
        models/ (14 files, typescript)
      run/ (2 files, typescript)
        apis/ (2 files, typescript)
        models/ (22 files, typescript)
      visualization/ (2 files, typescript)
        apis/ (2 files, typescript)
        models/ (5 files, typescript)
      experiment/ (2 files, typescript)
        apis/ (2 files, typescript)
        models/ (6 files, typescript)
      filter/ (2 files, typescript)
        models/ (9 files, typescript)
      pipeline/ (2 files, typescript)
        apis/ (2 files, typescript)
        models/ (10 files, typescript)
      recurringrun/ (2 files, typescript)
        apis/ (2 files, typescript)
        models/ (14 files, typescript)
      run/ (2 files, typescript)
        apis/ (2 files, typescript)
        models/ (16 files, typescript)
      visualization/ (2 files, typescript)
        apis/ (2 files, typescript)
        models/ (5 files, typescript)
    atoms/ (18 files, 7 tests, typescript)
    components/ (71 files, 36 tests, typescript)
      graph/ (9 files, 4 tests, typescript)
      navigators/ (2 files, 1 tests, typescript)
      tabs/ (12 files, 6 tests, typescript)
      viewers/ (23 files, 11 tests, typescript)
    hooks/ (3 files, typescript)
    icons/ (7 files, typescript)
    lib/ (33 files, 12 tests, typescript)
      v2/ (9 files, 4 tests, typescript)
    mlmd/ (23 files, 6 tests, typescript)
    pages/ (85 files, 40 tests, typescript)
      functional_components/ (4 files, 2 tests, typescript)
      v2/ (1 files, typescript)
    stories/ (6 files, typescript)
      v2/ (4 files, typescript)
    testUtils/ (1 files, typescript)
      mlmd/ (3 files, typescript)
  vite-plugins/ (1 files, typescript)
hack/ (4 files, bash)
    kubernetesplatform/ (1 files, go)
  proto/ (1 files, protobuf)
  python/ (5 files, bash, python)
    docs/ (1 files, python)
      kubernetes/ (15 files, python)
      snapshot/ (1 files, 1 tests, python)
        data/ (16 files, 16 tests, python)
      unit/ (13 files, 13 tests, python)
    deployer/ (3 files, bash)
    hack/ (2 files, bash)
  kustomize/ (2 files, bash)
          pipelines-profile-controller/ (3 files, 1 tests, bash, python)
    hack/ (4 files, bash)
      seaweedfs/ (1 files, bash)
  12147-mlmd-removal/ (1 files, sql)
    protos/ (2 files, protobuf)
proxy/ (3 files, 1 tests, bash, python)
    XGBoost/ (2 files, 1 tests, python)
    caching/ (2 files, 1 tests, python)
    condition/ (4 files, 2 tests, python)
    execution_order/ (2 files, 1 tests, python)
    exit_handler/ (2 files, 1 tests, python)
    kubernetes_pvc/ (1 files, python)
    loop_output/ (2 files, 1 tests, python)
    loop_parallelism/ (2 files, 1 tests, python)
    loop_parameter/ (2 files, 1 tests, python)
    loop_static/ (2 files, 1 tests, python)
    multiple_outputs/ (1 files, 1 tests, python)
    output_a_directory/ (2 files, 1 tests, python)
    parallel_join/ (1 files, python)
    recursion/ (1 files, python)
    resource_spec/ (5 files, 2 tests, python)
    retry/ (2 files, 1 tests, python)
    secret/ (2 files, 1 tests, python)
    sequential/ (1 files, python)
      utils/ (1 files, python)
    train_until_good/ (1 files, python)
    DSL - Control structures/ (1 files, python)
    Data passing in python components/ (1 files, python)
  hack/ (1 files, bash)
  python/ (2 files, bash, python)
    kfp/ (3 files, 1 tests, python)
      cli/ (15 files, 3 tests, python)
        diagnose_me/ (9 files, 4 tests, python)
        utils/ (7 files, 3 tests, python)
      client/ (9 files, 4 tests, python)
      compiler/ (7 files, 3 tests, python)
      components/ (3 files, 1 tests, python)
      dsl/ (47 files, 15 tests, python)
        templates/ (2 files, python)
        types/ (9 files, 4 tests, python)
      local/ (33 files, 15 tests, python)
        orchestrator/ (4 files, python)
      registry/ (3 files, 1 tests, python)
        context/ (1 files, python)
      v2/ (4 files, python)
    test/ (1 files, 1 tests, python)
      client/ (1 files, 1 tests, python)
      compilation/ (1 files, 1 tests, python)
      local_execution/ (1 files, 1 tests, python)
      runtime/ (1 files, 1 tests, python)
      test_utils/ (2 files, 2 tests, python)
test/ (21 files, 21 tests, bash)
  artifact-proxy/ (1 files, 1 tests, bash)
  frontend-integration-test/ (5 files, 5 tests, bash, javascript)
  gcpc-tests/ (1 files, 1 tests, python)
  imagebuilder/ (1 files, 1 tests, bash)
  kfp-kubernetes-native-migration-tests/ (5 files, 5 tests, python)
  release/ (4 files, 4 tests, bash)
  scripts/ (1 files, 1 tests, bash)
  seaweedfs/ (2 files, 2 tests, bash, python)
  server-integration-test/ (2 files, 2 tests, javascript)
    clean-mysql-runs/ (3 files, 3 tests, bash, python, sql)
    project-cleaner/ (2 files, 2 tests, bash, go)
    valid/ (76 files, 76 tests, python)
      critical/ (34 files, 34 tests, python)
        modelcar/ (1 files, 1 tests, python)
      essential/ (22 files, 22 tests, python)
      failing/ (3 files, 3 tests, python)
      integration/ (1 files, 1 tests, python)
  sdk_uncompiled_pipelines/ (7 files, 7 tests, python)
    v1/ (2 files, 2 tests, python)
third_party/ (1 files, bash)
  argo/ (1 files, bash)
  metadata_envoy/ (1 files, python)
  minio/ (1 files, bash)
  ml-metadata/ (1 files, bash)
      ml_metadata/ (3 files, go)
      proto/ (2 files, protobuf)
tools/ (1 files, go)
  code-tree/ (4 files, 2 tests, python)
  diagnose/ (1 files, bash)
  k8s-native/ (1 files, python)
  metadatastore-upgrade/ (1 files, go)
```

## Hub Files (most imported)

| File | Imported by |
|------|-------------|
| `tools/diagnose/kfp.sh` | 1326 files |
| `backend/src/common/util/time.go` | 380 files |
| `backend/src/common/util/json.go` | 324 files |
| `api/v2alpha1/google/protobuf/unittest.proto` | 225 files |
| `backend/api/v1beta1/python_http_client/kfp_server_api/rest.py` | 186 files |
| `backend/api/v2beta1/python_http_client/kfp_server_api/rest.py` | 186 files |
| `backend/src/apiserver/common/utils.go` | 173 files |
| `backend/src/common/util/uuid.go` | 166 files |
| `backend/src/common/util/string.go` | 162 files |
| `backend/src/common/util/client_parameters.go` | 158 files |
| `backend/src/common/util/consts.go` | 158 files |
| `backend/src/common/util/consts_test.go` | 158 files |
| `backend/src/common/util/error.go` | 158 files |
| `backend/src/common/util/error_test.go` | 158 files |
| `backend/src/common/util/execution_client.go` | 158 files |

## Most-Called Symbols

| Symbol | Called by |
|--------|----------|
| `assertEqual` | 721 callers |
| `OutputArtifact.Marshal` | 406 callers |
| `V2beta1CreatePipelineAndVersionRequest.pipeline` | 221 callers |
| `ContainerSpec` | 181 callers |
| `NewFakeTimeForEpoch` | 179 callers |
| `Options.info` | 169 callers |
| `NewWorkflow` | 148 callers |
| `OutputPath` | 145 callers |
| `component` | 136 callers |
| `Wrap` | 126 callers |
| `NewInvalidInputError` | 124 callers |
| `NewFakeUUIDGeneratorOrFatal` | 123 callers |
| `assertTrue` | 123 callers |
| `ArtifactAndType.Reset` | 114 callers |
| `ArtifactAndType.String` | 114 callers |

## Key Types

| Type | Kind | File | Line | Bases |
|------|------|------|------|-------|
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
| `AWSConfigs` | interface | `frontend/server/configs.ts` | 239 | - |
| `AWSInstanceProfileCredentials` | class | `frontend/server/aws-helper.ts` | 59 | - |
| `AWSMetadataCredentials` | interface | `frontend/server/aws-helper.ts` | 15 | - |
| `ActivationLayer` | class | `components/PyTorch/Create_fully_connected_network/component.py` | 17 | torch.nn.Module |
| `Affinity` | interface | `frontend/src/third_party/mlmd/kubernetes.ts` | 27 | - |
| `AliasedPluralsGroup` | class | `sdk/python/kfp/cli/utils/aliased_plurals_group.py` | 20 | click.Group |
| `AllExperimentsAndArchive` | class | `frontend/src/pages/AllExperimentsAndArchive.tsx` | 39 | Page |
| `AllExperimentsAndArchiveProps` | interface | `frontend/src/pages/AllExperimentsAndArchive.tsx` | 31 | - |
| `AllExperimentsAndArchiveState` | interface | `frontend/src/pages/AllExperimentsAndArchive.tsx` | 35 | - |

*... and 3367 more types (see tags.json)*


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
BaseComponent
  |- MarGeneration
  |- MinIO
  |- Trainer
  |- Visualization
BaseExecutor
  |- Executor
  |- GenericExecutor
BaseTestCase
  |- RLEstimatorComponentTestCase
DockerMockTestCase
  |- TestDockerTaskHandler
  |- TestE2E
  |- TestRunDockerContainer
Enum
  |- DebugRulesStatus
  |- TemplateType
Error
  |- FetchError
  |- RequiredError
  |- ResponseError
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
    |- TestCompare
  |- ExecutionList
  |- ExperimentDetails
  |- ExperimentList
    |- ExperimentListTest
  |- GettingStarted
  |- NewExperiment
  |- NewPipelineVersion
    |- TestNewPipelineVersion
  |- NewRun
    |- TestNewRun
  |- PipelineDetails
  |- PipelineList
  |- RecurringRunDetails
  |- RecurringRunDetailsV2
  |- RunDetails
```

## Key Functions

| Function | File | Line | Params |
|----------|------|------|--------|
| `Accept` | `backend/src/v2/compiler/visitor.go` | 57 | *pipelinespec.PipelineJob, *pipelinespec.SinglePlatformSpec, Visitor |
| `AddRuntimeMetadata` | `backend/src/apiserver/template/argo_template.go` | 231 | *workflowapi.Workflow |
| `AdmitFuncHandler` | `backend/src/cache/server/admission.go` | 146 | admitFunc, ClientManagerInterface |
| `AnyStringPtr` | `backend/src/common/util/pointer.go` | 115 | interface{} |
| `ApiCronScheduleFromJSON` | `frontend/src/apis/job/models/ApiCronSchedule.ts` | 49 | any |
| `ApiCronScheduleToJSON` | `frontend/src/apis/job/models/ApiCronSchedule.ts` | 67 | any |
| `ApiExperimentFromJSON` | `frontend/src/apis/experiment/models/ApiExperiment.ts` | 83 | any |
| `ApiExperimentFromJSONTyped` | `frontend/src/apis/experiment/models/ApiExperiment.ts` | 87 | any, boolean |
| `ApiExperimentStorageStateFromJSON` | `frontend/src/apis/experiment/models/ApiExperimentStorageState.ts` | 38 | any |
| `ApiExperimentStorageStateToJSON` | `frontend/src/apis/experiment/models/ApiExperimentStorageState.ts` | 49 | null |
| `ApiExperimentToJSON` | `frontend/src/apis/experiment/models/ApiExperiment.ts` | 107 | any |
| `ApiFilterFromJSON` | `frontend/src/apis/filter/models/ApiFilter.ts` | 84 | any |
| `ApiFilterFromJSONTyped` | `frontend/src/apis/filter/models/ApiFilter.ts` | 88 | any, boolean |
| `ApiFilterToJSON` | `frontend/src/apis/filter/models/ApiFilter.ts` | 100 | any |
| `ApiGetTemplateResponseFromJSON` | `frontend/src/apis/pipeline/models/ApiGetTemplateResponse.ts` | 38 | any |
| `ApiGetTemplateResponseToJSON` | `frontend/src/apis/pipeline/models/ApiGetTemplateResponse.ts` | 54 | any |
| `ApiIntValuesFromJSON` | `frontend/src/apis/filter/models/ApiIntValues.ts` | 37 | any |
| `ApiIntValuesFromJSONTyped` | `frontend/src/apis/filter/models/ApiIntValues.ts` | 41 | any, boolean |
| `ApiIntValuesToJSON` | `frontend/src/apis/filter/models/ApiIntValues.ts` | 50 | any |
| `ApiJobFromJSON` | `frontend/src/apis/job/models/ApiJob.ts` | 154 | any |
| `ApiJobFromJSONTyped` | `frontend/src/apis/job/models/ApiJob.ts` | 158 | any, boolean |
| `ApiJobToJSON` | `frontend/src/apis/job/models/ApiJob.ts` | 185 | any |
| `ApiListExperimentsResponseFromJSON` | `frontend/src/apis/experiment/models/ApiListExperimentsResponse.ts` | 59 | any |
| `ApiListExperimentsResponseToJSON` | `frontend/src/apis/experiment/models/ApiListExperimentsResponse.ts` | 80 | any |
| `ApiListJobsResponseFromJSON` | `frontend/src/apis/job/models/ApiListJobsResponse.ts` | 52 | any |
| `ApiListJobsResponseToJSON` | `frontend/src/apis/job/models/ApiListJobsResponse.ts` | 70 | any |
| `ApiListPipelineVersionsResponseToJSON` | `frontend/src/apis/pipeline/models/ApiListPipelineVersionsResponse.ts` | 82 | any |
| `ApiListPipelinesResponseFromJSON` | `frontend/src/apis/pipeline/models/ApiListPipelinesResponse.ts` | 59 | any |
| `ApiListPipelinesResponseToJSON` | `frontend/src/apis/pipeline/models/ApiListPipelinesResponse.ts` | 80 | any |
| `ApiListRunsResponseFromJSON` | `frontend/src/apis/run/models/ApiListRunsResponse.ts` | 52 | any |
| `ApiListRunsResponseToJSON` | `frontend/src/apis/run/models/ApiListRunsResponse.ts` | 70 | any |
| `ApiLongValuesFromJSON` | `frontend/src/apis/filter/models/ApiLongValues.ts` | 37 | any |
| `ApiLongValuesFromJSONTyped` | `frontend/src/apis/filter/models/ApiLongValues.ts` | 41 | any, boolean |
| `ApiLongValuesToJSON` | `frontend/src/apis/filter/models/ApiLongValues.ts` | 50 | any |
| `ApiParameterFromJSON` | `frontend/src/apis/job/models/ApiParameter.ts` | 43 | any |
| `ApiParameterFromJSON` | `frontend/src/apis/pipeline/models/ApiParameter.ts` | 43 | any |
| `ApiParameterFromJSON` | `frontend/src/apis/run/models/ApiParameter.ts` | 43 | any |
| `ApiParameterFromJSONTyped` | `frontend/src/apis/job/models/ApiParameter.ts` | 47 | any, boolean |
| `ApiParameterFromJSONTyped` | `frontend/src/apis/pipeline/models/ApiParameter.ts` | 47 | any, boolean |
| `ApiParameterFromJSONTyped` | `frontend/src/apis/run/models/ApiParameter.ts` | 47 | any, boolean |
| `ApiParameterToJSON` | `frontend/src/apis/job/models/ApiParameter.ts` | 57 | any |
| `ApiParameterToJSON` | `frontend/src/apis/pipeline/models/ApiParameter.ts` | 57 | any |
| `ApiParameterToJSON` | `frontend/src/apis/run/models/ApiParameter.ts` | 57 | any |
| `ApiPeriodicScheduleFromJSON` | `frontend/src/apis/job/models/ApiPeriodicSchedule.ts` | 49 | any |
| `ApiPeriodicScheduleToJSON` | `frontend/src/apis/job/models/ApiPeriodicSchedule.ts` | 67 | any |
| `ApiPipelineFromJSON` | `frontend/src/apis/pipeline/models/ApiPipeline.ts` | 116 | any |
| `ApiPipelineFromJSONTyped` | `frontend/src/apis/pipeline/models/ApiPipeline.ts` | 120 | any, boolean |
| `ApiPipelineRuntimeFromJSON` | `frontend/src/apis/run/models/ApiPipelineRuntime.ts` | 45 | any |
| `ApiPipelineRuntimeToJSON` | `frontend/src/apis/run/models/ApiPipelineRuntime.ts` | 62 | any |
| `ApiPipelineSpecFromJSON` | `frontend/src/apis/job/models/ApiPipelineSpec.ts` | 84 | any |

*... and 2842 more functions (see tags.json)*


## Module Dependencies

```
.github/actions/junit-summary -> backend/src/common/util
.github/resources/scripts/kfp-readiness -> backend/src/common/util, frontend/src/third_party/mlmd
api/v2alpha1/go/cachekey -> api/v2alpha1/go/pipelinespec, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller
api/v2alpha1/go/pipelinespec -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller
api/v2alpha1/python -> tools
backend/api/v1beta1/go_client -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/experiment_client -> backend/api/v1beta1/go_http_client/experiment_client/experiment_service
backend/api/v1beta1/go_http_client/experiment_client/experiment_service -> backend/api/v1beta1/go_http_client/experiment_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/healthz_client -> backend/api/v1beta1/go_http_client/healthz_client/healthz_service
backend/api/v1beta1/go_http_client/healthz_client/healthz_service -> backend/api/v1beta1/go_http_client/healthz_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/job_client -> backend/api/v1beta1/go_http_client/job_client/job_service
backend/api/v1beta1/go_http_client/job_client/job_service -> backend/api/v1beta1/go_http_client/job_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/pipeline_client -> backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service
backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service -> backend/api/v1beta1/go_http_client/pipeline_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/pipeline_upload_client -> backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service
backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service -> backend/api/v1beta1/go_http_client/pipeline_upload_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/run_client -> backend/api/v1beta1/go_http_client/run_client/run_service
backend/api/v1beta1/go_http_client/run_client/run_service -> backend/api/v1beta1/go_http_client/run_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/go_http_client/visualization_client -> backend/api/v1beta1/go_http_client/visualization_client/visualization_service
backend/api/v1beta1/go_http_client/visualization_client/visualization_service -> backend/api/v1beta1/go_http_client/visualization_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/python_http_client -> tools
backend/api/v1beta1/python_http_client/kfp_server_api -> backend/api/v1beta1/python_http_client/kfp_server_api/api, backend/api/v1beta1/python_http_client/kfp_server_api/models, backend/api/v2beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api/api, backend/api/v2beta1/python_http_client/kfp_server_api/models, backend/src/common/util, sdk/python/kfp/local
backend/api/v1beta1/python_http_client/kfp_server_api/api -> backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api/api
backend/api/v1beta1/python_http_client/kfp_server_api/models -> backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api/models
backend/api/v1beta1/python_http_client/test -> api/v2alpha1/google/protobuf, backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v1beta1/python_http_client/kfp_server_api/api, backend/api/v1beta1/python_http_client/kfp_server_api/models, backend/api/v2beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api/api, backend/api/v2beta1/python_http_client/kfp_server_api/models
backend/api/v2beta1/go_client -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/experiment_client -> backend/api/v2beta1/go_http_client/experiment_client/experiment_service
backend/api/v2beta1/go_http_client/experiment_client/experiment_service -> backend/api/v2beta1/go_http_client/experiment_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/healthz_client -> backend/api/v2beta1/go_http_client/healthz_client/healthz_service
backend/api/v2beta1/go_http_client/healthz_client/healthz_service -> backend/api/v2beta1/go_http_client/healthz_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/pipeline_client -> backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service
backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service -> backend/api/v2beta1/go_http_client/pipeline_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/pipeline_upload_client -> backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service
backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service -> backend/api/v2beta1/go_http_client/pipeline_upload_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/recurring_run_client -> backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service
backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service -> backend/api/v2beta1/go_http_client/recurring_run_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/run_client -> backend/api/v2beta1/go_http_client/run_client/run_service
backend/api/v2beta1/go_http_client/run_client/run_service -> backend/api/v2beta1/go_http_client/run_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/go_http_client/visualization_client -> backend/api/v2beta1/go_http_client/visualization_client/visualization_service
backend/api/v2beta1/go_http_client/visualization_client/visualization_service -> backend/api/v2beta1/go_http_client/visualization_model, backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/python_http_client -> tools
backend/api/v2beta1/python_http_client/kfp_server_api -> backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v1beta1/python_http_client/kfp_server_api/api, backend/api/v1beta1/python_http_client/kfp_server_api/models, backend/api/v2beta1/python_http_client/kfp_server_api/api, backend/api/v2beta1/python_http_client/kfp_server_api/models, backend/src/common/util, sdk/python/kfp/local
backend/api/v2beta1/python_http_client/kfp_server_api/api -> backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v1beta1/python_http_client/kfp_server_api/api, backend/api/v2beta1/python_http_client/kfp_server_api
backend/api/v2beta1/python_http_client/kfp_server_api/models -> backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v1beta1/python_http_client/kfp_server_api/models, backend/api/v2beta1/python_http_client/kfp_server_api
backend/api/v2beta1/python_http_client/test -> api/v2alpha1/google/protobuf, backend/api/v1beta1/python_http_client/kfp_server_api, backend/api/v1beta1/python_http_client/kfp_server_api/api, backend/api/v1beta1/python_http_client/kfp_server_api/models, backend/api/v2beta1/python_http_client/kfp_server_api, backend/api/v2beta1/python_http_client/kfp_server_api/api, backend/api/v2beta1/python_http_client/kfp_server_api/models
backend/metadata_writer/src -> backend/src/common/util, frontend/src/third_party/mlmd, tools
backend/src/agent/persistence -> backend/src/agent/persistence/client, backend/src/agent/persistence/client/tokenrefresher, backend/src/agent/persistence/worker, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow, backend/src/crd/pkg/client/clientset/versioned, backend/src/crd/pkg/client/clientset/versioned/scheme, backend/src/crd/pkg/client/informers/externalversions
backend/src/agent/persistence/client -> backend/api/v1beta1/go_client, backend/src/agent/persistence/client/artifactclient, backend/src/agent/persistence/client/tokenrefresher, backend/src/common/util, backend/src/crd/pkg/client/informers/externalversions/scheduledworkflow/v1beta1
backend/src/agent/persistence/client/artifactclient -> backend/src/agent/persistence/client/tokenrefresher, backend/src/common/util, sdk/python/kfp/local
backend/src/agent/persistence/client/tokenrefresher -> backend/src/apiserver/archive, backend/src/common/util, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller
backend/src/agent/persistence/worker -> backend/api/v1beta1/go_client, backend/src/agent/persistence/client, backend/src/agent/persistence/client/artifactclient, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1
backend/src/apiserver -> backend/api/v1beta1/go_client, backend/api/v2beta1/go_client, backend/src/apiserver/client_manager, backend/src/apiserver/common, backend/src/apiserver/config, backend/src/apiserver/config/proxy, backend/src/apiserver/resource, backend/src/apiserver/server, backend/src/apiserver/template, backend/src/apiserver/webhook, backend/src/common/util, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, sdk/python/kfp/local
backend/src/apiserver/archive -> backend/src/common/util, sdk/python/kfp/local
backend/src/apiserver/auth -> backend/src/apiserver/client, backend/src/apiserver/common, backend/src/common/util
backend/src/apiserver/client -> backend/src/apiserver/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/client/clientset/versioned, backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1
backend/src/apiserver/client_manager -> backend/src/apiserver/archive, backend/src/apiserver/auth, backend/src/apiserver/client, backend/src/apiserver/common, backend/src/apiserver/model, backend/src/apiserver/storage, backend/src/apiserver/validation, backend/src/common/util, backend/src/crd/kubernetes/v2beta1, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller
backend/src/apiserver/common -> backend/src/common/util
backend/src/apiserver/config -> backend/src/apiserver/client, backend/src/apiserver/common, backend/src/apiserver/list, backend/src/apiserver/model, backend/src/apiserver/resource, backend/src/apiserver/server, backend/src/common/util
backend/src/apiserver/config/proxy -> backend/src/apiserver/common
backend/src/apiserver/filter -> backend/api/v1beta1/go_client, backend/api/v2beta1/go_client, backend/src/apiserver/model, backend/src/common/util, backend/src/crd/kubernetes/v2beta1
backend/src/apiserver/list -> backend/api/v1beta1/go_client, backend/src/apiserver/common, backend/src/apiserver/filter, backend/src/apiserver/model, backend/src/common/util
backend/src/apiserver/resource -> backend/api/v2beta1/go_client, backend/src/apiserver/archive, backend/src/apiserver/auth, backend/src/apiserver/client, backend/src/apiserver/common, backend/src/apiserver/config/proxy, backend/src/apiserver/list, backend/src/apiserver/model, backend/src/apiserver/storage, backend/src/apiserver/template, backend/src/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1, sdk/python/kfp/local
backend/src/apiserver/server -> api/v2alpha1/go/pipelinespec, backend/api/v1beta1/go_client, backend/api/v2beta1/go_client, backend/src/apiserver/client, backend/src/apiserver/common, backend/src/apiserver/config/proxy, backend/src/apiserver/filter, backend/src/apiserver/list, backend/src/apiserver/model, backend/src/apiserver/resource, backend/src/apiserver/template, backend/src/apiserver/validation, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, sdk/python/kfp/local
backend/src/apiserver/storage -> backend/api/v1beta1/go_client, backend/api/v2beta1/go_client, backend/src/apiserver/common, backend/src/apiserver/filter, backend/src/apiserver/list, backend/src/apiserver/model, backend/src/common/util, backend/src/crd/kubernetes/v2beta1, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, sdk/python/kfp/local
backend/src/apiserver/template -> api/v2alpha1/go/pipelinespec, backend/src/apiserver/common, backend/src/apiserver/config/proxy, backend/src/apiserver/model, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/v2/compiler/argocompiler, sdk/python/kfp/local
backend/src/apiserver/validation -> backend/src/apiserver/common, backend/src/apiserver/model, backend/src/common/util
backend/src/apiserver/visualization -> api/v2alpha1/google/protobuf, backend/src/common/util
backend/src/apiserver/visualization/types -> backend/src/common/util
backend/src/apiserver/webhook -> backend/src/apiserver/common, backend/src/apiserver/template, backend/src/crd/kubernetes/v2beta1
backend/src/cache -> backend/src/apiserver/archive, backend/src/cache/client, backend/src/cache/model, backend/src/cache/server, backend/src/cache/storage, backend/src/common/util
backend/src/cache/client -> backend/src/common/util
backend/src/cache/server -> backend/src/apiserver/archive, backend/src/cache/client, backend/src/cache/model, backend/src/cache/storage, backend/src/common/util, sdk/python/kfp/local
backend/src/cache/storage -> backend/src/apiserver/archive, backend/src/cache/model, backend/src/common/util
backend/src/common/client/api_server -> backend/src/common/util, backend/test/config
backend/src/common/client/api_server/v1 -> backend/api/v1beta1/go_http_client/experiment_client, backend/api/v1beta1/go_http_client/experiment_client/experiment_service, backend/api/v1beta1/go_http_client/experiment_model, backend/api/v1beta1/go_http_client/healthz_client, backend/api/v1beta1/go_http_client/healthz_client/healthz_service, backend/api/v1beta1/go_http_client/healthz_model, backend/api/v1beta1/go_http_client/job_client, backend/api/v1beta1/go_http_client/job_client/job_service, backend/api/v1beta1/go_http_client/job_model, backend/api/v1beta1/go_http_client/pipeline_client, backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service, backend/api/v1beta1/go_http_client/pipeline_model, backend/api/v1beta1/go_http_client/pipeline_upload_client, backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service, backend/api/v1beta1/go_http_client/pipeline_upload_model, backend/api/v1beta1/go_http_client/run_client, backend/api/v1beta1/go_http_client/run_client/run_service, backend/api/v1beta1/go_http_client/run_model, backend/api/v1beta1/go_http_client/visualization_client, backend/api/v1beta1/go_http_client/visualization_client/visualization_service, backend/api/v1beta1/go_http_client/visualization_model, backend/src/apiserver/template, backend/src/common/client/api_server, backend/src/common/util
backend/src/common/client/api_server/v2 -> backend/api/v2beta1/go_http_client/experiment_client, backend/api/v2beta1/go_http_client/experiment_client/experiment_service, backend/api/v2beta1/go_http_client/experiment_model, backend/api/v2beta1/go_http_client/healthz_client, backend/api/v2beta1/go_http_client/healthz_client/healthz_service, backend/api/v2beta1/go_http_client/healthz_model, backend/api/v2beta1/go_http_client/pipeline_client, backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service, backend/api/v2beta1/go_http_client/pipeline_model, backend/api/v2beta1/go_http_client/pipeline_upload_client, backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service, backend/api/v2beta1/go_http_client/pipeline_upload_model, backend/api/v2beta1/go_http_client/recurring_run_client, backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service, backend/api/v2beta1/go_http_client/recurring_run_model, backend/api/v2beta1/go_http_client/run_client, backend/api/v2beta1/go_http_client/run_client/run_service, backend/api/v2beta1/go_http_client/run_model, backend/src/apiserver/common, backend/src/apiserver/model, backend/src/apiserver/server, backend/src/apiserver/template, backend/src/common/client/api_server, backend/src/common/util, backend/src/crd/kubernetes/v2beta1, sdk/python/kfp/local
backend/src/common/util -> backend/api/v1beta1/go_client, backend/src/agent/persistence/client/artifactclient, backend/src/common, backend/src/crd/pkg/apis/scheduledworkflow, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, sdk/python/kfp/local
backend/src/crd/controller/scheduledworkflow -> backend/api/v2beta1/go_client, backend/src/common/util, backend/src/crd/controller/scheduledworkflow/client, backend/src/crd/controller/scheduledworkflow/util, backend/src/crd/pkg/apis/scheduledworkflow, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/client/clientset/versioned, backend/src/crd/pkg/client/clientset/versioned/scheme, backend/src/crd/pkg/client/informers/externalversions
backend/src/crd/controller/scheduledworkflow/client -> backend/src/common, backend/src/common/util, backend/src/crd/controller/scheduledworkflow/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/client/clientset/versioned, backend/src/crd/pkg/client/informers/externalversions/scheduledworkflow/v1beta1
backend/src/crd/controller/scheduledworkflow/util -> backend/src/apiserver/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1
backend/src/crd/controller/viewer -> backend/src/apiserver/archive, backend/src/crd/controller/viewer/reconciler, backend/src/crd/pkg/apis/viewer/v1beta1
backend/src/crd/controller/viewer/reconciler -> backend/src/common/util, backend/src/crd/pkg/apis/viewer/v1beta1
backend/src/crd/kubernetes/v2beta1 -> backend/src/apiserver/model, backend/src/apiserver/template
backend/src/crd/pkg/apis/scheduledworkflow/v1beta1 -> backend/src/common, backend/src/crd/pkg/apis/scheduledworkflow
backend/src/crd/pkg/apis/viewer/v1beta1 -> backend/src/crd/pkg/apis/viewer
backend/src/crd/pkg/client/clientset/versioned -> backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1
backend/src/crd/pkg/client/clientset/versioned/fake -> backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/client/clientset/versioned, backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1, backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1/fake
backend/src/crd/pkg/client/clientset/versioned/scheme -> backend/src/crd/pkg/apis/scheduledworkflow/v1beta1
backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1 -> backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/client/clientset/versioned/scheme
backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1/fake -> backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1
backend/src/crd/pkg/client/informers/externalversions -> backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/client/clientset/versioned, backend/src/crd/pkg/client/informers/externalversions/internalinterfaces, backend/src/crd/pkg/client/informers/externalversions/scheduledworkflow, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller
backend/src/crd/pkg/client/informers/externalversions/internalinterfaces -> backend/src/common/util, backend/src/crd/pkg/client/clientset/versioned
backend/src/crd/pkg/client/informers/externalversions/scheduledworkflow -> backend/src/crd/pkg/client/informers/externalversions/internalinterfaces, backend/src/crd/pkg/client/informers/externalversions/scheduledworkflow/v1beta1
backend/src/crd/pkg/client/informers/externalversions/scheduledworkflow/v1beta1 -> backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/client/clientset/versioned, backend/src/crd/pkg/client/informers/externalversions/internalinterfaces, backend/src/crd/pkg/client/listers/scheduledworkflow/v1beta1
backend/src/crd/pkg/client/listers/scheduledworkflow/v1beta1 -> backend/src/crd/pkg/apis/scheduledworkflow/v1beta1
backend/src/v2/cacheutils -> api/v2alpha1/go/cachekey, api/v2alpha1/go/pipelinespec, backend/api/v1beta1/go_client
backend/src/v2/client_manager -> backend/src/common/util, backend/src/v2/cacheutils, backend/src/v2/metadata
backend/src/v2/cmd/compiler -> api/v2alpha1/go/pipelinespec, backend/src/v2/compiler/argocompiler
backend/src/v2/cmd/driver -> api/v2alpha1/go/pipelinespec, backend/src/apiserver/config/proxy, backend/src/common/util, backend/src/v2/cacheutils, backend/src/v2/config, backend/src/v2/driver, backend/src/v2/metadata, kubernetes_platform/go/kubernetesplatform
backend/src/v2/cmd/launcher-v2 -> backend/src/v2/client_manager, backend/src/v2/component, backend/src/v2/config
backend/src/v2/compiler -> api/v2alpha1/go/pipelinespec
backend/src/v2/compiler/argocompiler -> api/v2alpha1/go/pipelinespec, backend/src/apiserver/common, backend/src/apiserver/config/proxy, backend/src/common/util, backend/src/v2/compiler, backend/src/v2/component, backend/src/v2/config, backend/src/v2/metadata, kubernetes_platform/go/kubernetesplatform
backend/src/v2/compiler/testdata -> tools/diagnose
backend/src/v2/component -> api/v2alpha1/go/pipelinespec, backend/api/v1beta1/go_client, backend/src/common/util, backend/src/v2/cacheutils, backend/src/v2/client_manager, backend/src/v2/config, backend/src/v2/metadata, backend/src/v2/objectstore, sdk/python/kfp/local, third_party/ml-metadata/go/ml_metadata
backend/src/v2/config -> backend/src/apiserver/common, backend/src/v2/objectstore
backend/src/v2/driver -> api/v2alpha1/go/cachekey, api/v2alpha1/go/pipelinespec, backend/api/v1beta1/go_client, backend/src/apiserver/config/proxy, backend/src/common/util, backend/src/v2/cacheutils, backend/src/v2/component, backend/src/v2/config, backend/src/v2/expression, backend/src/v2/metadata, backend/src/v2/objectstore, kubernetes_platform/go/kubernetesplatform, third_party/ml-metadata/go/ml_metadata
backend/src/v2/expression -> api/v2alpha1/go/pipelinespec, backend/src/v2/metadata
backend/src/v2/metadata -> api/v2alpha1/go/pipelinespec, backend/src/apiserver/common, backend/src/common/util, backend/src/v2/metadata/testutils, backend/src/v2/objectstore, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller, third_party/ml-metadata/go/ml_metadata
backend/src/v2/metadata/testutils -> backend/test/v2, third_party/ml-metadata/go/ml_metadata
backend/src/v2/objectstore -> sdk/python/kfp/local
backend/test -> backend/api/v1beta1/go_client, backend/api/v1beta1/go_http_client/experiment_client/experiment_service, backend/api/v1beta1/go_http_client/experiment_model, backend/api/v1beta1/go_http_client/job_client/job_service, backend/api/v1beta1/go_http_client/job_model, backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service, backend/api/v1beta1/go_http_client/pipeline_model, backend/api/v1beta1/go_http_client/run_client/run_service, backend/api/v1beta1/go_http_client/run_model, backend/src/common/client/api_server/v1, backend/src/common/util
backend/test/compiler -> api/v2alpha1/go/pipelinespec, backend/src/apiserver/archive, backend/src/apiserver/config/proxy, backend/src/v2/compiler, backend/src/v2/compiler/argocompiler, backend/test/compiler/matchers, backend/test/compiler/utils, backend/test/constants, backend/test/logger, backend/test/testutil
backend/test/compiler/matchers -> backend/test/logger, backend/test/v2/api/matcher
backend/test/compiler/utils -> api/v2alpha1/go/pipelinespec, backend/src/v2/compiler/argocompiler, backend/test/logger, backend/test/testutil
backend/test/end2end -> backend/api/v2beta1/go_http_client/experiment_model, backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service, backend/api/v2beta1/go_http_client/pipeline_upload_model, backend/api/v2beta1/go_http_client/run_model, backend/src/apiserver/archive, backend/src/common/client/api_server/v2, backend/src/common/util, backend/test/compiler/utils, backend/test/config, backend/test/constants, backend/test/end2end/utils, backend/test/logger, backend/test/testutil, backend/test/v2/api
backend/test/end2end/utils -> backend/api/v2beta1/go_http_client/run_client/run_service, backend/api/v2beta1/go_http_client/run_model, backend/src/common/client/api_server/v2, backend/src/common/util, backend/test/config, backend/test/logger, backend/test/testutil, backend/test/v2/api
backend/test/initialization -> backend/api/v1beta1/go_http_client/experiment_client/experiment_service, backend/src/common/client/api_server/v1, backend/src/common/util, backend/test, backend/test/config
backend/test/integration -> backend/api/v1beta1/go_client, backend/api/v1beta1/go_http_client/experiment_client/experiment_service, backend/api/v1beta1/go_http_client/experiment_model, backend/api/v1beta1/go_http_client/job_client/job_service, backend/api/v1beta1/go_http_client/job_model, backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service, backend/api/v1beta1/go_http_client/pipeline_model, backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service, backend/api/v1beta1/go_http_client/pipeline_upload_model, backend/api/v1beta1/go_http_client/run_client/run_service, backend/api/v1beta1/go_http_client/run_model, backend/api/v1beta1/go_http_client/visualization_client/visualization_service, backend/api/v1beta1/go_http_client/visualization_model, backend/src/apiserver/client, backend/src/apiserver/client_manager, backend/src/apiserver/template, backend/src/common/client/api_server/v1, backend/src/common/util, backend/src/crd/kubernetes/v2beta1, backend/test, backend/test/config, backend/test/testutil, sdk/python/kfp/local
backend/test/proto_tests -> api/v2alpha1/go/pipelinespec, backend/api/v2beta1/go_client, backend/src/apiserver/server, backend/src/common/util
backend/test/resources -> tools/diagnose
backend/test/testutil -> api/v2alpha1/go/pipelinespec, backend/api/v2beta1/go_http_client/experiment_client/experiment_service, backend/api/v2beta1/go_http_client/experiment_model, backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service, backend/api/v2beta1/go_http_client/pipeline_model, backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service, backend/api/v2beta1/go_http_client/pipeline_upload_model, backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service, backend/api/v2beta1/go_http_client/recurring_run_model, backend/api/v2beta1/go_http_client/run_client/run_service, backend/api/v2beta1/go_http_client/run_model, backend/src/apiserver/common, backend/src/apiserver/server, backend/src/apiserver/template, backend/src/common/client/api_server/v2, backend/src/common/util, backend/test/config, backend/test/constants, backend/test/logger, sdk/python/kfp/local
backend/test/v2 -> backend/api/v2beta1/go_http_client/experiment_client/experiment_service, backend/api/v2beta1/go_http_client/experiment_model, backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service, backend/api/v2beta1/go_http_client/pipeline_model, backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service, backend/api/v2beta1/go_http_client/recurring_run_model, backend/api/v2beta1/go_http_client/run_client/run_service, backend/api/v2beta1/go_http_client/run_model, backend/src/common/client/api_server/v2, backend/src/common/util
backend/test/v2/api -> backend/api/v2beta1/go_http_client/experiment_client/experiment_service, backend/api/v2beta1/go_http_client/experiment_model, backend/api/v2beta1/go_http_client/pipeline_model, backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service, backend/api/v2beta1/go_http_client/pipeline_upload_model, backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service, backend/api/v2beta1/go_http_client/recurring_run_model, backend/api/v2beta1/go_http_client/run_client/run_service, backend/api/v2beta1/go_http_client/run_model, backend/src/apiserver/archive, backend/src/common/client/api_server/v2, backend/src/common/util, backend/test/config, backend/test/constants, backend/test/logger, backend/test/testutil, backend/test/v2/api/matcher
backend/test/v2/api/matcher -> backend/api/v2beta1/go_http_client/pipeline_upload_model, backend/api/v2beta1/go_http_client/run_model, backend/src/apiserver/template, backend/src/common/util, backend/test/config, backend/test/logger, backend/test/testutil
backend/test/v2/integration -> backend/api/v2beta1/go_http_client/experiment_client/experiment_service, backend/api/v2beta1/go_http_client/experiment_model, backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service, backend/api/v2beta1/go_http_client/pipeline_model, backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service, backend/api/v2beta1/go_http_client/pipeline_upload_model, backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service, backend/api/v2beta1/go_http_client/recurring_run_model, backend/api/v2beta1/go_http_client/run_client/run_service, backend/api/v2beta1/go_http_client/run_model, backend/src/apiserver/client, backend/src/common/client/api_server/v2, backend/src/common/util, backend/src/v2/metadata, backend/src/v2/metadata/testutils, backend/test/config, backend/test/testutil, backend/test/v2, third_party/ml-metadata/go/ml_metadata
components/PyTorch/_samples -> tools/diagnose
components/PyTorch/pytorch-kfp-components -> backend/src/common, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, tools
components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/mar -> components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/base
components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/minio -> backend/src/apiserver/client, backend/src/v2/config, components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/base
components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/trainer -> components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/base
components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/visualization -> backend/src/common/util, components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/base, components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/minio
components/PyTorch/pytorch-kfp-components/tests -> api/v2alpha1/google/protobuf, backend/src/common/util, components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/mar, components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/minio, components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/trainer, components/PyTorch/pytorch-kfp-components/pytorch_kfp_components/components/visualization, tools/diagnose
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
components/aws/sagemaker/common -> backend/src/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1
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
components/aws/sagemaker/tests/integration_tests -> backend/src/apiserver/common, components/google-cloud/google_cloud_pipeline_components, components/google-cloud/google_cloud_pipeline_components/_implementation/llm, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation, components/google-cloud/google_cloud_pipeline_components/container/v1/aiplatform, components/google-cloud/google_cloud_pipeline_components/preview/automl/forecasting, components/google-cloud/google_cloud_pipeline_components/preview/automl/tabular, components/google-cloud/google_cloud_pipeline_components/preview/custom_job, components/google-cloud/google_cloud_pipeline_components/preview/model_evaluation, components/google-cloud/google_cloud_pipeline_components/v1/automl/forecasting, components/google-cloud/google_cloud_pipeline_components/v1/automl/tabular, components/google-cloud/google_cloud_pipeline_components/v1/custom_job, components/google-cloud/google_cloud_pipeline_components/v1/hyperparameter_tuning_job, frontend/server, sdk/python/kfp/dsl, sdk/python/kfp/local, tools/diagnose
components/aws/sagemaker/tests/integration_tests/component_tests -> backend/src/apiserver/common, backend/src/common/util, components/aws/sagemaker/tests/unit_tests/tests/workteam, components/google-cloud/google_cloud_pipeline_components, components/google-cloud/google_cloud_pipeline_components/_implementation/llm, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation, components/google-cloud/google_cloud_pipeline_components/container/v1/aiplatform, components/google-cloud/google_cloud_pipeline_components/preview/automl/forecasting, components/google-cloud/google_cloud_pipeline_components/preview/automl/tabular, components/google-cloud/google_cloud_pipeline_components/preview/custom_job, components/google-cloud/google_cloud_pipeline_components/preview/model_evaluation, components/google-cloud/google_cloud_pipeline_components/v1/automl/forecasting, components/google-cloud/google_cloud_pipeline_components/v1/automl/tabular, components/google-cloud/google_cloud_pipeline_components/v1/custom_job, components/google-cloud/google_cloud_pipeline_components/v1/hyperparameter_tuning_job, frontend/server, sdk/python/kfp/dsl, sdk/python/kfp/local
components/aws/sagemaker/tests/integration_tests/resources/definition -> tools/diagnose
components/aws/sagemaker/tests/integration_tests/scripts -> sdk/python/kfp/local
components/aws/sagemaker/tests/integration_tests/utils -> backend/src/apiserver/client, backend/src/apiserver/common, backend/src/common/util, backend/src/v2/config, components/google-cloud/google_cloud_pipeline_components, components/google-cloud/google_cloud_pipeline_components/_implementation/llm, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation, components/google-cloud/google_cloud_pipeline_components/container/v1/aiplatform, components/google-cloud/google_cloud_pipeline_components/preview/automl/forecasting, components/google-cloud/google_cloud_pipeline_components/preview/automl/tabular, components/google-cloud/google_cloud_pipeline_components/preview/custom_job, components/google-cloud/google_cloud_pipeline_components/preview/model_evaluation, components/google-cloud/google_cloud_pipeline_components/v1/automl/forecasting, components/google-cloud/google_cloud_pipeline_components/v1/automl/tabular, components/google-cloud/google_cloud_pipeline_components/v1/custom_job, components/google-cloud/google_cloud_pipeline_components/v1/hyperparameter_tuning_job, frontend/server, frontend/src/third_party/mlmd, sdk/python/kfp/dsl, sdk/python/kfp/local
components/aws/sagemaker/tests/unit_tests/tests/MonitoringSchedule -> api/v2alpha1/google/protobuf, components/aws/sagemaker/MonitoringSchedule/src, components/aws/sagemaker/commonv2
components/aws/sagemaker/tests/unit_tests/tests/batch_transform -> api/v2alpha1/google/protobuf, components/aws/sagemaker/batch_transform/src, components/aws/sagemaker/common
components/aws/sagemaker/tests/unit_tests/tests/common -> api/v2alpha1/google/protobuf, backend/src/common/util, components/aws/sagemaker/common
components/aws/sagemaker/tests/unit_tests/tests/deploy -> api/v2alpha1/google/protobuf, components/aws/sagemaker/common, components/aws/sagemaker/deploy/src
components/aws/sagemaker/tests/unit_tests/tests/ground_truth -> api/v2alpha1/google/protobuf, components/aws/sagemaker/common, components/aws/sagemaker/ground_truth/src
components/aws/sagemaker/tests/unit_tests/tests/hyperparameter_tuning -> api/v2alpha1/google/protobuf, components/aws/sagemaker/common, components/aws/sagemaker/hyperparameter_tuning/src
components/aws/sagemaker/tests/unit_tests/tests/model -> api/v2alpha1/google/protobuf, components/aws/sagemaker/common, components/aws/sagemaker/model/src
components/aws/sagemaker/tests/unit_tests/tests/process -> api/v2alpha1/google/protobuf, backend/src/common/util, components/aws/sagemaker/common, components/aws/sagemaker/process/src
components/aws/sagemaker/tests/unit_tests/tests/rlestimator -> api/v2alpha1/google/protobuf, components/aws/sagemaker/common, components/aws/sagemaker/rlestimator/src
components/aws/sagemaker/tests/unit_tests/tests/robomaker -> api/v2alpha1/google/protobuf, backend/src/common/util, components/aws/sagemaker/common, components/aws/sagemaker/create_simulation_app/src, components/aws/sagemaker/delete_simulation_app/src, components/aws/sagemaker/simulation_job/src, components/aws/sagemaker/simulation_job_batch/src
components/aws/sagemaker/tests/unit_tests/tests/train -> api/v2alpha1/google/protobuf, components/aws/sagemaker/common, components/aws/sagemaker/train/src
components/aws/sagemaker/tests/unit_tests/tests/workteam -> api/v2alpha1/google/protobuf, backend/src/common/util, components/aws/sagemaker/common, components/aws/sagemaker/workteam/src
components/aws/sagemaker/train/src -> components/aws/sagemaker/common
components/aws/sagemaker/workteam/src -> components/aws/sagemaker/common
components/google-cloud -> backend/src/common, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, tools
components/google-cloud/docs/source -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components -> backend/src/common/util, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/llm -> api/v2alpha1/google/protobuf, backend/src/common/util, tools, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model -> components/google-cloud/google_cloud_pipeline_components/_implementation/model/get_model
components/google-cloud/google_cloud_pipeline_components/_implementation/model/get_model -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation -> components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/chunking, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/data_sampler, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/dataset_preprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/endpoint_batch_predict, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/error_analysis_annotation, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/evaluated_annotation, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/feature_attribution, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/feature_extractor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/import_evaluated_annotation, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_classification_postprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_embedding_retrieval, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_evaluation, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_evaluation_preprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_information_retrieval_preprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_retrieval_metrics, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_safety_bias, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/model_name_preprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/target_field_data_remover
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/chunking -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/dataset_preprocessor -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/endpoint_batch_predict -> backend/src/common/util, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/evaluated_annotation -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/feature_attribution -> components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/data_sampler, components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/feature_extractor -> components/google-cloud/google_cloud_pipeline_components/types
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/import_evaluated_annotation -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_classification_postprocessor -> backend/src/common/util, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_embedding -> components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/endpoint_batch_predict, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_embedding_retrieval, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_information_retrieval_preprocessor, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_retrieval_metrics, components/google-cloud/google_cloud_pipeline_components/preview/model_evaluation, components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_evaluation -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_evaluation_preprocessor -> backend/src/common/util, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/llm_safety_bias -> components/google-cloud/google_cloud_pipeline_components/types, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/text2sql -> components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/endpoint_batch_predict, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/text2sql_evaluation, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/text2sql_preprocess, components/google-cloud/google_cloud_pipeline_components/_implementation/model_evaluation/text2sql_validate_and_process, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net -> components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/dataprep, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/evaluation, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/get_training_artifacts, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/maybe_set_tfrecord_args, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_dataprep_args, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_eval_args, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_test_set, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_tfrecord_args, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_train_args, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/train, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/upload_decomposition_plots, components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/upload_model
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/dataprep -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/evaluation -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/get_training_artifacts -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/maybe_set_tfrecord_args -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_dataprep_args -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_eval_args -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_test_set -> backend/src/common/util, tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_tfrecord_args -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/set_train_args -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/train -> tools/diagnose
components/google-cloud/google_cloud_pipeline_components/_implementation/starry_net/upload_decomposition_plots -> backend/src/common/util, tools/diagnose
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
frontend -> frontend/vite-plugins
frontend/mock-backend -> frontend/mock-backend/data/v1/runtime, frontend/src/apis/experiment, frontend/src/apis/filter, frontend/src/apis/job, frontend/src/apis/pipeline, frontend/src/apis/run, frontend/src/lib
frontend/scripts -> frontend/vite-plugins
frontend/server -> backend/src/apiserver/client, backend/src/v2/config
frontend/server/handlers -> backend/src/apiserver/client, backend/src/v2/config
frontend/server/integration-tests -> backend/src/apiserver/client, backend/src/v2/config
frontend/src -> frontend/src/__mocks__, frontend/src/components, frontend/src/lib, frontend/src/pages
frontend/src/apis/experiment/apis -> frontend/src/apis/experiment, frontend/src/apis/experiment/models
frontend/src/apis/experiment/models -> frontend/src/apis/experiment
frontend/src/apis/filter/models -> frontend/src/apis/filter
frontend/src/apis/job/apis -> frontend/src/apis/job, frontend/src/apis/job/models
frontend/src/apis/job/models -> frontend/src/apis/job
frontend/src/apis/pipeline/apis -> frontend/src/apis/pipeline
frontend/src/apis/pipeline/models -> frontend/src/apis/pipeline
frontend/src/apis/run/apis -> frontend/src/apis/run
frontend/src/apis/run/models -> frontend/src/apis/run
frontend/src/apis/visualization/apis -> frontend/src/apis/visualization, frontend/src/apis/visualization/models
frontend/src/apis/visualization/models -> frontend/src/apis/visualization
frontend/src/apisv2beta1/experiment/apis -> frontend/src/apisv2beta1/experiment
frontend/src/apisv2beta1/experiment/models -> frontend/src/apisv2beta1/experiment
frontend/src/apisv2beta1/filter/models -> frontend/src/apisv2beta1/filter
frontend/src/apisv2beta1/pipeline/apis -> frontend/src/apisv2beta1/pipeline
frontend/src/apisv2beta1/pipeline/models -> frontend/src/apisv2beta1/pipeline
frontend/src/apisv2beta1/recurringrun/apis -> frontend/src/apisv2beta1/recurringrun
frontend/src/apisv2beta1/recurringrun/models -> frontend/src/apisv2beta1/recurringrun
frontend/src/apisv2beta1/run/apis -> frontend/src/apisv2beta1/run, frontend/src/apisv2beta1/run/models
frontend/src/apisv2beta1/run/models -> frontend/src/apisv2beta1/run
frontend/src/apisv2beta1/visualization/apis -> frontend/src/apisv2beta1/visualization, frontend/src/apisv2beta1/visualization/models
frontend/src/apisv2beta1/visualization/models -> frontend/src/apisv2beta1/visualization
frontend/src/atoms -> frontend/src, frontend/src/__mocks__, frontend/src/lib
frontend/src/components -> frontend/src, frontend/src/__mocks__, frontend/src/apis/pipeline, frontend/src/apis/run, frontend/src/apisv2beta1/filter, frontend/src/atoms, frontend/src/components/tabs, frontend/src/components/viewers, frontend/src/icons, frontend/src/lib, frontend/src/pages, frontend/src/third_party/mlmd, tools
frontend/src/components/graph -> frontend/src/__mocks__
frontend/src/components/tabs -> frontend/src/components, frontend/src/components/viewers
frontend/src/components/viewers -> frontend/src, frontend/src/__mocks__, frontend/src/apis/visualization, frontend/src/atoms, frontend/src/components, frontend/src/lib, tools
frontend/src/lib -> frontend/src, frontend/src/__mocks__, frontend/src/apis/job, frontend/src/apis/pipeline, frontend/src/apis/run, frontend/src/apis/visualization, frontend/src/atoms, frontend/src/components, frontend/src/components/viewers, frontend/src/lib/v2, frontend/src/pages, frontend/src/third_party/mlmd
frontend/src/lib/v2 -> frontend/src/__mocks__
frontend/src/mlmd -> frontend/src/__mocks__
frontend/src/pages -> frontend/src, frontend/src/__mocks__, frontend/src/apis/experiment, frontend/src/apis/filter, frontend/src/apis/job, frontend/src/apis/pipeline, frontend/src/apis/run, frontend/src/atoms, frontend/src/components, frontend/src/components/viewers, frontend/src/icons, frontend/src/lib, frontend/src/pages/functional_components, frontend/src/pages/v2, frontend/src/third_party/mlmd, tools
frontend/src/pages/functional_components -> frontend/src/__mocks__
frontend/src/stories/v2 -> frontend/src/components/graph
kubernetes_platform/go/kubernetesplatform -> api/v2alpha1/go/pipelinespec, manifests/kustomize/base/installs/multi-user/pipelines-profile-controller
kubernetes_platform/python -> tools
kubernetes_platform/python/docs -> tools/diagnose
kubernetes_platform/python/kfp/kubernetes -> frontend/src/third_party/mlmd, sdk/python/kfp/compiler, tools/diagnose
kubernetes_platform/python/test/snapshot -> api/v2alpha1/google/protobuf, tools/diagnose
kubernetes_platform/python/test/snapshot/data -> kubernetes_platform/python/kfp/kubernetes, tools/diagnose
kubernetes_platform/python/test/unit -> tools/diagnose
manifests/kustomize/base/installs/multi-user/pipelines-profile-controller -> api/v2alpha1/google/protobuf, backend/src/common/util
proxy -> api/v2alpha1/google/protobuf, backend/src/common/util
samples/core/XGBoost -> tools/diagnose
samples/core/caching -> backend/src/common/util, tools/diagnose
samples/core/condition -> api/v2alpha1/google/protobuf, tools/diagnose
samples/core/execution_order -> tools/diagnose
samples/core/exit_handler -> api/v2alpha1/google/protobuf, tools/diagnose
samples/core/kubernetes_pvc -> tools/diagnose
samples/core/loop_output -> api/v2alpha1/google/protobuf, tools/diagnose
samples/core/loop_parallelism -> tools/diagnose
samples/core/loop_parameter -> api/v2alpha1/google/protobuf, tools/diagnose
samples/core/loop_static -> api/v2alpha1/google/protobuf, backend/src/common/util, tools/diagnose
samples/core/parallel_join -> tools/diagnose
samples/core/recursion -> tools/diagnose
samples/core/resource_spec -> tools/diagnose
samples/core/retry -> tools/diagnose
samples/core/secret -> tools/diagnose
samples/core/sequential -> tools/diagnose
samples/core/train_until_good -> tools/diagnose
samples/tutorials/DSL - Control structures -> tools/diagnose
samples/tutorials/Data passing in python components -> backend/src/common/util, tools/diagnose
sdk/python -> tools
sdk/python/kfp -> api/v2alpha1/google/protobuf, tools/diagnose
sdk/python/kfp/cli -> api/v2alpha1/google/protobuf, backend/src/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, sdk/python/kfp/compiler, sdk/python/kfp/dsl, tools, tools/diagnose
sdk/python/kfp/cli/diagnose_me -> api/v2alpha1/google/protobuf, backend/src/common/util
sdk/python/kfp/cli/utils -> api/v2alpha1/google/protobuf, backend/src/common/util, tools/diagnose
sdk/python/kfp/client -> api/v2alpha1/google/protobuf, backend/src/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, frontend/src/third_party/mlmd, tools/diagnose
sdk/python/kfp/compiler -> api/v2alpha1/google/protobuf, backend/src/common/util, sdk/python/kfp/dsl, sdk/python/kfp/v2, tools, tools/diagnose
sdk/python/kfp/components -> api/v2alpha1/google/protobuf, sdk/python/kfp/dsl, tools/diagnose
sdk/python/kfp/dsl -> api/v2alpha1/google/protobuf, backend/src/common, backend/src/common/util, backend/src/crd/pkg/apis/scheduledworkflow/v1beta1, backend/src/crd/pkg/apis/viewer/v1beta1, sdk/python/kfp/dsl/templates, sdk/python/kfp/dsl/types, sdk/python/kfp/local, tools, tools/diagnose
sdk/python/kfp/dsl/templates -> backend/src/common/util
sdk/python/kfp/dsl/types -> api/v2alpha1/google/protobuf, backend/src/common/util, sdk/python/kfp/dsl, tools/diagnose
sdk/python/kfp/local -> api/v2alpha1/google/protobuf, backend/src/common/util, tools, tools/diagnose
sdk/python/kfp/local/orchestrator -> backend/src/common/util
sdk/python/kfp/registry -> api/v2alpha1/google/protobuf, backend/src/common/util
sdk/python/kfp/v2 -> tools/diagnose
sdk/python/test -> sdk/python/test/test_utils
sdk/python/test/client -> backend/api/v2beta1/python_http_client/kfp_server_api/models, backend/src/common/util, sdk/python/test/test_utils, test_data/sdk_compiled_pipelines/valid, tools/diagnose
sdk/python/test/compilation -> sdk/python/test/test_utils, test_data/sdk_compiled_pipelines/valid, test_data/sdk_compiled_pipelines/valid/critical, test_data/sdk_compiled_pipelines/valid/critical/modelcar, test_data/sdk_compiled_pipelines/valid/essential, test_data/sdk_compiled_pipelines/valid/failing, tools, tools/diagnose
sdk/python/test/local_execution -> test_data/sdk_compiled_pipelines/valid, test_data/sdk_compiled_pipelines/valid/critical, test_data/sdk_compiled_pipelines/valid/essential, tools, tools/diagnose
sdk/python/test/runtime -> backend/src/common/util, sdk/python/test/test_utils, tools/diagnose
test/gcpc-tests -> api/v2alpha1/google/protobuf
test/kfp-kubernetes-native-migration-tests -> backend/src/common/util, tools/diagnose, tools/k8s-native
test/tools/clean-mysql-runs -> backend/src/common/util, tools/diagnose
test/tools/project-cleaner -> backend/src/apiserver/archive, backend/src/common/util
test_data/sdk_compiled_pipelines/valid -> backend/src/common/util, tools, tools/diagnose
test_data/sdk_compiled_pipelines/valid/critical -> backend/src/common/util, sdk/python/kfp/dsl/types, tools/diagnose
test_data/sdk_compiled_pipelines/valid/critical/modelcar -> tools/diagnose
test_data/sdk_compiled_pipelines/valid/essential -> backend/src/common/util, sdk/python/kfp/local, sdk/python/test/test_utils, tools/diagnose
test_data/sdk_compiled_pipelines/valid/failing -> tools/diagnose
test_data/sdk_compiled_pipelines/valid/integration -> backend/src/common/util, tools/diagnose
test_data/sdk_uncompiled_pipelines -> backend/src/common/util, tools/diagnose
test_data/sdk_uncompiled_pipelines/v1 -> tools/diagnose
third_party/metadata_envoy -> backend/src/common/util
third_party/ml-metadata/go/ml_metadata -> manifests/kustomize/base/installs/multi-user/pipelines-profile-controller
tools/code-tree -> api/v2alpha1/google/protobuf, backend/src/common/util
tools/metadatastore-upgrade -> backend/src/apiserver/archive, backend/src/common/util
```

## Test Coverage Map

| Source File | Tested By |
|------------|-----------|
| `api/v2alpha1/go/cachekey/cache_key.pb.go` | `backend/src/v2/cacheutils/cache_test.go`, `backend/src/v2/driver/cache_test.go` |
| `api/v2alpha1/go/pipelinespec/pipeline_spec.pb.go` | `backend/src/apiserver/template/template_test.go`, `backend/src/v2/cacheutils/cache_test.go`, `backend/src/v2/compiler/argocompiler/argo_workspace_test.go` +17 more |
| `api/v2alpha1/google/protobuf/any.proto` | `api/v2alpha1/google/protobuf/any_test.proto` |
| `api/v2alpha1/google/protobuf/type.proto` | `backend/test/constants/test_type.go` |
| `api/v2alpha1/google/protobuf/unittest.proto` | `backend/api/v1beta1/python_http_client/test/test_api_cron_schedule.py`, `backend/api/v1beta1/python_http_client/test/test_api_experiment.py`, `backend/api/v1beta1/python_http_client/test/test_api_experiment_storage_state.py` +189 more |
| `backend/api/v1beta1/go_client/auth.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/auth.pb.gw.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/auth_grpc.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/error.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/experiment.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/experiment.pb.gw.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/experiment_grpc.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/filter.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/filter_grpc.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/healthz.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/healthz.pb.gw.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/healthz_grpc.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/job.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/job.pb.gw.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/job_grpc.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/parameter.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/pipeline.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/pipeline.pb.gw.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/pipeline_grpc.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/pipeline_spec.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/report.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/report.pb.gw.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/report_grpc.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/resource_reference.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |
| `backend/api/v1beta1/go_client/run.pb.go` | `backend/src/agent/persistence/worker/metrics_reporter_test.go`, `backend/src/apiserver/filter/filter_test.go`, `backend/src/apiserver/list/list_test.go` +22 more |

*... and 905 more tested files*

