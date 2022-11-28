from commonv2.sagemaker_component import SageMakerJobStatus

from Endpoint.src.Endpoint_spec import SageMakerEndpointSpec
from Endpoint.src.Endpoint_component import SageMakerEndpointComponent
import unittest

from unittest.mock import patch, MagicMock


class EndpointComponentTestCase(unittest.TestCase):
    REQUIRED_ARGS = [
        "--region",
        "us-west-1",
        "--endpoint_config_name",
        "test",
        "--endpoint_name",
        "test1",
    ]

    @classmethod
    def setUp(cls):
        cls.component = SageMakerEndpointComponent()
        cls.component.job_name = "test"

    def test_get_job_status(self):
        with patch(
            "Endpoint.src.Endpoint_component.SageMakerComponent._get_resource"
        ) as mock_get_resource:
            # InService Endpoint
            mock_get_resource.return_value = {"status": {"endpointStatus": "InService", "conditions": [ {"type": "ACK.ResourceSynced",  "status": "True"} ]}}

            self.assertEqual(
                self.component._get_job_status(),
                SageMakerJobStatus(
                    is_completed=True, raw_status="InService", has_error=False
                ),
            )

            mock_get_resource.return_value = {
                "status": {
                    "endpointStatus": "Failed",
                    "failureReason": "invalid model",
                    "conditions": [ {"type": "ACK.ResourceSynced",  "status": "True"} ]

                }
            }

            # Failed Endpoint
            self.assertEqual(
                self.component._get_job_status(),
                SageMakerJobStatus(
                    is_completed=True,
                    raw_status="Failed",
                    has_error=True,
                    error_message="invalid model",
                ),
            )

            mock_get_resource.return_value = {
                "status": {
                    "endpointStatus": "OutOfService",
                    "conditions": [ {"type": "ACK.ResourceSynced",  "status": "True"} ]
                }
            }

            # Out of service Endpoint
            self.assertEqual(
                self.component._get_job_status(),
                SageMakerJobStatus(
                    is_completed=True,
                    raw_status="OutOfService",
                    has_error=True,
                    error_message="Sagemaker endpoint is Out of Service",
                ),
            )

            mock_get_resource.return_value = {
                "status": {
                    "endpointStatus": "Creating",
                }, "conditions": [ {"type": "ACK.ResourceSynced",  "status": "False"}]
            }

            self.assertEqual(
                self.component._get_job_status(),
                SageMakerJobStatus(
                    is_completed=False, raw_status="Creating", has_error=False
                ),
            )

    def test_get_upgrade_status(self):
        with patch(
            "Endpoint.src.Endpoint_component.SageMakerComponent._get_resource"
        ) as mock_get_resource:
            with patch(
                "Endpoint.src.Endpoint_component.SageMakerComponent._get_conditions_of_type"
            ) as mock_condition_list:

                mock_get_resource.return_value = {"status": {"endpointStatus": "InService", "conditions": [ {"type": "ACK.ResourceSynced",  "status": "True"} ]}}
                mock_condition_list.return_value = []

                self.assertEqual(
                    self.component._get_upgrade_status(),
                    SageMakerJobStatus(
                        is_completed=True, raw_status="InService", has_error=False
                    ),
                )

                mock_get_resource.return_value = {
                    "status": {"endpointStatus": "Updating"}
                }

                self.assertEqual(
                    self.component._get_upgrade_status(),
                    SageMakerJobStatus(
                        is_completed=False, raw_status="Updating", has_error=False
                    ),
                )

                mock_get_resource.return_value = {
                    "status": {
                        "endpointStatus": "Failed",
                        "failureReason": "invalid model",
                        "conditions": [ {"type": "ACK.ResourceSynced",  "status": "True"} ],
                    }
                }

                # Failed Endpoint
                self.assertEqual(
                    self.component._get_upgrade_status(),
                    SageMakerJobStatus(
                        is_completed=True,
                        raw_status="Failed",
                        has_error=True,
                        error_message="invalid model",
                    ),
                )

    def test_after_job_submit(self):
        spec = SageMakerEndpointSpec(self.REQUIRED_ARGS)
        request = {"spec": {"endpointName": ""}}
        request["spec"]["endpointName"] = spec.inputs.endpoint_name
        self.component.job_name = request["spec"]["endpointName"]
        with self.assertLogs() as captured:
            self.component._after_submit_job_request(
                {}, request, spec.inputs, spec.outputs
            )
            log_line1 = "Endpoint in Sagemaker: https://us-west-1.console.aws.amazon.com/sagemaker/home?region=us-west-1#/endpoints/test1"
            log_line2 = "CloudWatch logs: https://us-west-1.console.aws.amazon.com/cloudwatch/home?region=us-west-1#logStream:group=/aws/sagemaker/Endpoints/test1"
            assert log_line1 in captured.records[0].getMessage()
            assert log_line2 in captured.records[1].getMessage()

    @patch("Endpoint.src.Endpoint_component.super", MagicMock())
    def test_do_sets_name(self):

        named_spec = SageMakerEndpointSpec(self.REQUIRED_ARGS)
        with patch(
            "Endpoint.src.Endpoint_component.SageMakerComponent._get_current_namespace"
        ) as mock_namespace:
            mock_namespace.return_value = "test-namespace"
            self.component.Do(named_spec)
            self.assertEqual("test1", self.component.job_name)

    def test_after_job_completed(self):

        spec = SageMakerEndpointSpec(self.REQUIRED_ARGS)

        statuses = {
            "status": {
                "ackResourceMetadata": 1,
                "conditions": 2,
                "creationTime": 3,
                "endpointStatus": 4,
                "failureReason": 5,
                "lastModifiedTime": 6,
                "pendingDeploymentSummary": 7,
                "productionVariants": 8,
            }
        }

        with patch(
            "Endpoint.src.Endpoint_component.SageMakerComponent._get_resource",
            MagicMock(return_value=statuses),
        ):

            self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

            self.assertEqual(spec.outputs.ack_resource_metadata, "1")
            self.assertEqual(spec.outputs.conditions, "2")
            self.assertEqual(spec.outputs.creation_time, "3")
            self.assertEqual(spec.outputs.endpoint_status, "4")
            self.assertEqual(spec.outputs.failure_reason, "5")
            self.assertEqual(spec.outputs.last_modified_time, "6")
            self.assertEqual(spec.outputs.pending_deployment_summary, "7")
            self.assertEqual(spec.outputs.production_variants, "8")
            self.assertEqual(spec.outputs.sagemaker_resource_name, "test")


if __name__ == "__main__":
    unittest.main()
