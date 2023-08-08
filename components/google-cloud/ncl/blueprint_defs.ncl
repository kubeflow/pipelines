include "devtools/blueprint/ncl/blueprint_file.ncl";
include "testing/integration/config/blueprint_extension.proto";
include "testing/integration/config/buganizer.proto";

def PipelinesBuildableUnitName(service, version, test_type) =
  service + "-" + version + "-" + test_type + "-" + "workflows";

def PipelinesTargetName(service, version, test_type) =
  "//third_party/py/google_cloud_pipeline_components/google/centralized_testing:gcpc_" + service + "_" + version + "_" + test_type + "_" + "workflow";

def PipelinesBuildableUnit(service, version, test_type) =
  ::blueprint::BuildableUnit(
    name = "buildable-unit-" + PipelinesBuildableUnitName(service, version, test_type),
    test_patterns = [PipelinesTargetName(service, version, test_type)],
    enable_release = false,
  );

def PipelinesContinuousIntegrationTest(
    service,
    version,
    test_type,
    throttle_rule = "every 24 hours",
    cluster_name = "vertex-pipelines-platform-e2e-test-cluster",
    requester = "cloud-aiplatform-guitar",
    env_params = [],
    bug_component = 1088378) = let
    unit_name = PipelinesBuildableUnitName(service, version, test_type);
  in
    ::blueprint::ContinuousIntegrationTest(
      name = "continuous-integration-" + unit_name,
      buildable_unit_name = "buildable-unit-" + unit_name,
      base_version_spec = "cl:HEAD",
      throttle = throttle_rule,
      cluster_name = cluster_name,
      requester = requester,
      env_params = env_params,
      notification_info = guitar::update_buganizer_config(
        ::blueprint::ContinuousIntegrationNotificationInfo(
          email_notification_criteria = ::devtools_blueprint::ContinuousIntegrationNotificationInfo::NotificationCriteria::NO_NOTIFICATION,
        ),
        ::guitar::BuganizerConfig(
          component_id = bug_component,
          reporter = "cloud-ml-pipelines+e2e-tests@google.com",
          cc = ["cjmccarthy@google.com"],
        ),
      ),
    );
