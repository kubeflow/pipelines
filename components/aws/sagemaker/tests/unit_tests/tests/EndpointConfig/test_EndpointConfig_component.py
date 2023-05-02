from commonv2.sagemaker_component import SageMakerJobStatus

from unittest.mock import patch, MagicMock
import unittest
from EndpointConfig.src.EndpointConfig_spec import SageMakerEndpointConfigSpec
from EndpointConfig.src.EndpointConfig_component import SageMakerEndpointConfigComponent


class EndpointConfigComponentTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-1",
        "--endpoint_config_name",
        "sample-endpoint-config",
        "--production_variants",
        "[]",
    ]

    @classmethod
    def setUp(cls):
        cls.component = SageMakerEndpointConfigComponent()
        cls.component.job_name = "sample-endpoint-config"

    @patch("EndpointConfig.src.EndpointConfig_component.super", MagicMock())
    def test_do_sets_name(self):

        named_spec = SageMakerEndpointConfigSpec(self.REQUIRED_ARGS)
        with patch(
            "EndpointConfig.src.EndpointConfig_component.SageMakerComponent._get_current_namespace"
        ) as mock_namespace:
            mock_namespace.return_value = "test-namespace"
            self.component.Do(named_spec)
            self.assertEqual("sample-endpoint-config", self.component.job_name)

    def test_get_job_status(self):
        with patch(
            "EndpointConfig.src.EndpointConfig_component.SageMakerComponent._get_resource"
        ) as mock_get_resource:
            mock_get_resource.return_value = {"arn": "arn:aws:sagemaker:us-east-1:000000000000:endpoint-config/sample-endpoint-config"}

            self.assertEqual(
                self.component._get_job_status(),
                SageMakerJobStatus(
                    is_completed=True, raw_status="Completed", has_error=False
                ),
            )

    def test_after_job_submit(self):
        spec = SageMakerEndpointConfigSpec(self.REQUIRED_ARGS)
        request = {"spec": {"endpointConfigName": ""}}
        request["spec"]["endpointConfigName"] = spec.inputs.endpoint_config_name
        self.component.job_name = request["spec"]["endpointConfigName"]
        with self.assertLogs() as captured:
            self.component._after_submit_job_request(
                {}, request, spec.inputs, spec.outputs
            )
            log_line1 = "Endpoint Config in Sagemaker: https://us-west-1.console.aws.amazon.com/sagemaker/home?region=us-west-1#/endpointConfig/sample-endpoint-config"
            assert log_line1 in captured.records[0].getMessage()

    def test_after_job_completed(self):

        spec = SageMakerEndpointConfigSpec(self.REQUIRED_ARGS)

        statuses = {
            "status": {
                "ackResourceMetadata": 1,
                "conditions": 2,
            }
        }

        with patch(
            "EndpointConfig.src.EndpointConfig_component.SageMakerComponent._get_resource",
            MagicMock(return_value=statuses),
        ):

            self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

            self.assertEqual(spec.outputs.ack_resource_metadata, "1")
            self.assertEqual(spec.outputs.conditions, "2")
            self.assertEqual(spec.outputs.sagemaker_resource_name, "sample-endpoint-config")

    def test_get_upgrade_status(self):
        with patch(
            "EndpointConfig.src.EndpointConfig_component.SageMakerComponent._get_resource"
        ) as mock_get_resource:
            with patch(
                "EndpointConfig.src.EndpointConfig_component.SageMakerComponent._get_conditions_of_type"
            ) as mock_condition_list:

                # Accidental update, no change.
                # Updates with change are caught by the common unit tests.
                mock_get_resource.return_value = {"arn": "arn:aws:sagemaker:us-east-1:000000000000:endpoint-config/sample-endpoint-config"}
                mock_condition_list.return_value = []
                self.assertEqual(
                    self.component._get_upgrade_status(),
                    SageMakerJobStatus(
                        is_completed=True, raw_status="Completed", has_error=False
                    ),
                )


if __name__ == "__main__":
    unittest.main()
