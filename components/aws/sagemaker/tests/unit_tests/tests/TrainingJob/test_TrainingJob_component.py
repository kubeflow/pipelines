from commonv2.sagemaker_component import SageMakerJobStatus
from TrainingJob.src.TrainingJob_spec import SageMakerTrainingJobSpec
from TrainingJob.src.TrainingJob_component import SageMakerTrainingJobComponent
from tests.unit_tests.tests.TrainingJob.test_TrainingJob_spec import (
    TrainingSpecTestCase,
)
import unittest

from unittest.mock import patch, MagicMock


class TrainingComponentTestCase(unittest.TestCase):
    REQUIRED_ARGS = TrainingSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = SageMakerTrainingJobComponent()
        # Instantiate without calling Do()

    @patch("TrainingJob.src.TrainingJob_component.super", MagicMock())
    def test_do_sets_name(self):
        named_spec = SageMakerTrainingJobSpec(self.REQUIRED_ARGS)
        with patch("TrainingJob.src.TrainingJob_component.SageMakerComponent._get_current_namespace") as mock_namespace:
            mock_namespace.return_value = "test-namespace"
            self.component.Do(named_spec)
            self.assertEqual("kfp-ack-training-job-99999", self.component.job_name)

    def test_get_job_status(self):

        with patch(
            "TrainingJob.src.TrainingJob_component.SageMakerComponent._get_resource"
        ) as mock_get_resource:
            with patch(
                "TrainingJob.src.TrainingJob_component.SageMakerComponent._get_resource_synced_status"
            ) as mock_resource_sync:

                mock_resource_sync.return_value = False

                mock_get_resource.return_value = {
                    "status": {"trainingJobStatus": "Starting"}
                }
                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(is_completed=False, raw_status="Starting"),
                )

                mock_resource_sync.return_value = True

                mock_get_resource.return_value = {
                    "status": {"trainingJobStatus": "Completed"}
                }
                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(
                        is_completed=True, raw_status="Completed", has_error=False
                    ),
                )

                mock_get_resource.return_value = {
                    "status": {"trainingJobStatus": "Stopped"}
                }
                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(
                        is_completed=True,
                        raw_status="Stopped",
                        has_error=True,
                        error_message="Sagemaker job was stopped",
                    ),
                )

                mock_get_resource.return_value = {
                    "status": {"failureReason": "crash", "trainingJobStatus": "Failed"}
                }

                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(
                        is_completed=True,
                        raw_status="Failed",
                        has_error=True,
                        error_message="crash",
                    ),
                )

    def test_job_status_debugger(self):
        with patch(
            "TrainingJob.src.TrainingJob_component.SageMakerComponent._get_resource"
        ) as mock_get_resource:
            with patch(
                "TrainingJob.src.TrainingJob_component.SageMakerComponent._get_resource_synced_status"
            ) as mock_resource_sync:

                mock_get_resource.return_value = {
                    "status": {"trainingJobStatus": "Completed"}
                }

                mock_resource_sync.return_value = False

                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(
                        is_completed=False, raw_status="Completed", has_error=False
                    ),
                )

                debugRulesStatus = [
                    {
                        "lastModifiedTime": "2022-08-30T22:53:17Z",
                        "ruleConfigurationName": "LossNotDecreasing",
                        "ruleEvaluationStatus": "IssuesFound",
                    }
                ]
                profilerRulesStatus = [
                    {
                        "lastModifiedTime": "2022-08-30T22:53:17Z",
                        "ruleConfigurationName": "ProfilerReport",
                        "ruleEvaluationStatus": "NoIssuesFound",
                    }
                ]
                mock_get_resource.return_value = {
                    "status": {
                        "trainingJobStatus": "Completed",
                        "debugRuleEvaluationStatuses": debugRulesStatus,
                        "profilerRuleEvaluationStatuses": profilerRulesStatus,
                    }
                }
                mock_resource_sync.return_value = True
                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(
                        is_completed=True, raw_status="Completed", has_error=True
                    ),
                )
                debugRulesStatus = [
                    {
                        "lastModifiedTime": "2022-08-30T22:53:17Z",
                        "ruleConfigurationName": "LossNotDecreasing",
                        "ruleEvaluationStatus": "NoIssuesFound",
                    }
                ]
                mock_get_resource.return_value = {
                    "status": {
                        "trainingJobStatus": "Completed",
                        "debugRuleEvaluationStatuses": debugRulesStatus,
                        "profilerRuleEvaluationStatuses": profilerRulesStatus,
                    }
                }
                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(
                        is_completed=True, raw_status="Completed", has_error=False
                    ),
                )

    def test_after_job_completed(self):

        spec = SageMakerTrainingJobSpec(self.REQUIRED_ARGS)

        statuses = {
            "status": {
                "ackResourceMetadata": 1,
                "conditions": 2,
                "debugRuleEvaluationStatuses": 3,
                "failureReason": 4,
                "modelArtifacts": 5,
                "profilerRuleEvaluationStatuses": 6,
                "secondaryStatus": 7,
                "trainingJobStatus": 8,
            }
        }

        with patch(
            "TrainingJob.src.TrainingJob_component.SageMakerComponent._get_resource",
            MagicMock(return_value=statuses),
        ):

            self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

            self.assertEqual(spec.outputs.ack_resource_metadata, "1")
            self.assertEqual(spec.outputs.conditions, "2")
            self.assertEqual(spec.outputs.debug_rule_evaluation_statuses, "3")
            self.assertEqual(spec.outputs.failure_reason, "4")
            self.assertEqual(spec.outputs.model_artifacts, "5")
            self.assertEqual(spec.outputs.profiler_rule_evaluation_statuses, "6")
            self.assertEqual(spec.outputs.secondary_status, "7")
            self.assertEqual(spec.outputs.training_job_status, "8")

    def test_after_job_submit(self):
        spec = SageMakerTrainingJobSpec(self.REQUIRED_ARGS)
        request = {"spec": {"trainingJobName": ""}}
        request["spec"]["trainingJobName"] = spec.inputs.training_job_name
        self.component.job_name = request["spec"]["trainingJobName"]
        with self.assertLogs() as captured:
            self.component._after_submit_job_request(
                {}, request, spec.inputs, spec.outputs
            )
            log_line1 = "Training job in SageMaker: https://us-west-1.console.aws.amazon.com/sagemaker/home?region=us-west-1#/jobs/kfp-ack-training-job-99999"
            log_line2 = "CloudWatch logs: https://us-west-1.console.aws.amazon.com/cloudwatch/home?region=us-west-1#logStream:group=/aws/sagemaker/TrainingJobs;prefix=kfp-ack-training-job-99999;streamFilter=typeLogStreamPrefix"
            assert log_line1 in captured.records[1].getMessage()
            assert log_line2 in captured.records[2].getMessage()


if __name__ == "__main__":
    unittest.main()
