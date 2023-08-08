include "third_party/py/google_cloud_pipeline_components/ncl/blueprint_defs.ncl";

def test_PipelinesBuildableUnitName() = let
    assert PipelinesBuildableUnitName(service = "bigquery", version = "v1", test_type = "public") == "bigquery-v1-public-workflows";
  in ();

def test_PipelinesTargetName() = let
    assert PipelinesTargetName(service = "bigquery", version = "v1", test_type = "public") == "//third_party/py/google_cloud_pipeline_components/google/centralized_testing:gcpc_bigquery_v1_public_workflow";
  in ();

def test_PipelinesBuildableUnit() = let
    buildable_unit = ::blueprint::BuildableUnit(
      name = "buildable-unit-bigquery-v1-public-workflows",
      test_patterns = ["//third_party/py/google_cloud_pipeline_components/google/centralized_testing:gcpc_bigquery_v1_public_workflow"],
      enable_release = false,
    );
    assert PipelinesBuildableUnit(
      service = "bigquery",
      version = "v1",
      test_type = "public",
    ) == buildable_unit;
  in ();

def test_PipelinesContinuousIntegrationTest() = let
    continuous_test = ::blueprint::ContinuousIntegrationTest(
      name = "continuous-integration-bigquery-v1-public-workflows",
      buildable_unit_name = "buildable-unit-bigquery-v1-public-workflows",
      base_version_spec = "cl:HEAD",
      throttle = "every 24 hours",
      cluster_name = "vertex-pipelines-platform-e2e-test-cluster",
      requester = "cloud-aiplatform-guitar",
      env_params = [],
      notification_info = guitar::update_buganizer_config(
        ::blueprint::ContinuousIntegrationNotificationInfo(
          email_notification_criteria = ::devtools_blueprint::ContinuousIntegrationNotificationInfo::NotificationCriteria::NO_NOTIFICATION,
        ),
        ::guitar::BuganizerConfig(
          component_id = 1088378,
          reporter = "cloud-ml-pipelines+e2e-tests@google.com",
          cc = ["cjmccarthy@google.com"],
        ),
      ),
    );
    assert PipelinesContinuousIntegrationTest(service = "bigquery", version = "v1", test_type = "public") == continuous_test;
  in ();
