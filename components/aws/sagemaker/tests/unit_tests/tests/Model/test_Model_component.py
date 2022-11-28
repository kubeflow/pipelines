from commonv2.sagemaker_component import SageMakerJobStatus

from unittest.mock import patch, MagicMock
import unittest
from Model.src.Model_spec import SageMakerModelSpec
from Model.src.Model_component import SageMakerModelComponent


class ModelComponentTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-1",
        "--model_name",
        "test1",
        "--execution_role_arn",
        "arn",
    ]

    @classmethod
    def setUp(cls):
        cls.component = SageMakerModelComponent()
        cls.component.job_name = "test"

    @patch("Model.src.Model_component.super", MagicMock())
    def test_do_sets_name(self):

        named_spec = SageMakerModelSpec(self.REQUIRED_ARGS)
        with patch(
            "Model.src.Model_component.SageMakerComponent._get_current_namespace"
        ) as mock_namespace:
            mock_namespace.return_value = "test-namespace"
            self.component.Do(named_spec)
            self.assertEqual("test1", self.component.job_name)

    def test_get_job_status(self):
        with patch(
            "Model.src.Model_component.SageMakerComponent._get_resource"
        ) as mock_get_resource:
            mock_get_resource.return_value = {"arn": "arn"}

            self.assertEqual(
                self.component._get_job_status(),
                SageMakerJobStatus(
                    is_completed=True, raw_status="Completed", has_error=False
                ),
            )

    def test_after_job_submit(self):
        spec = SageMakerModelSpec(self.REQUIRED_ARGS)
        request = {"spec": {"modelName": ""}}
        request["spec"]["modelName"] = spec.inputs.model_name
        self.component.job_name = request["spec"]["modelName"]
        with self.assertLogs() as captured:
            self.component._after_submit_job_request(
                {}, request, spec.inputs, spec.outputs
            )
            log_line1 = "Model in Sagemaker: https://us-west-1.console.aws.amazon.com/sagemaker/home?region=us-west-1#/models/test"
            assert log_line1 in captured.records[0].getMessage()

    def test_after_job_completed(self):

        spec = SageMakerModelSpec(self.REQUIRED_ARGS)

        statuses = {
            "status": {
                "ackResourceMetadata": 1,
                "conditions": 2,
            }
        }

        with patch(
            "Model.src.Model_component.SageMakerComponent._get_resource",
            MagicMock(return_value=statuses),
        ):

            self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

            self.assertEqual(spec.outputs.ack_resource_metadata, "1")
            self.assertEqual(spec.outputs.conditions, "2")

    def test_get_upgrade_status(self):
        with patch(
            "Model.src.Model_component.SageMakerComponent._get_resource"
        ) as mock_get_resource:
            with patch(
                "Model.src.Model_component.SageMakerComponent._get_conditions_of_type"
            ) as mock_condition_list:

                # Accidental update, no change.
                # Updates with change are caught by the common unit tests.
                mock_get_resource.return_value = {"arn": "arn"}
                mock_condition_list.return_value = []
                self.assertEqual(
                    self.component._get_upgrade_status(),
                    SageMakerJobStatus(
                        is_completed=True, raw_status="Completed", has_error=False
                    ),
                )


if __name__ == "__main__":
    unittest.main()
