from common.sagemaker_component import SageMakerJobStatus
from ground_truth.src.sagemaker_ground_truth_spec import SageMakerGroundTruthSpec
from ground_truth.src.sagemaker_ground_truth_component import (
    SageMakerGroundTruthComponent,
)
from tests.unit_tests.tests.ground_truth.test_ground_truth_spec import (
    GroundTruthSpecTestCase,
)
import unittest

from unittest.mock import patch, MagicMock, ANY


class GroundTruthComponentTestCase(unittest.TestCase):
    REQUIRED_ARGS = GroundTruthSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = SageMakerGroundTruthComponent()
        # Instantiate without calling Do()
        cls.component._labeling_job_name = "my-labeling-job"

    @patch("ground_truth.src.sagemaker_ground_truth_component.super", MagicMock())
    def test_do_sets_name(self):
        named_spec = SageMakerGroundTruthSpec(
            self.REQUIRED_ARGS + ["--job_name", "test-job"]
        )
        unnamed_spec = SageMakerGroundTruthSpec(self.REQUIRED_ARGS)

        self.component.Do(named_spec)
        self.assertEqual("test-job", self.component._labeling_job_name)

        with patch(
            "ground_truth.src.sagemaker_ground_truth_component.SageMakerComponent._generate_unique_timestamped_id",
            MagicMock(return_value="unique"),
        ):
            self.component.Do(unnamed_spec)
            self.assertEqual("unique", self.component._labeling_job_name)

    def test_create_ground_truth_job(self):
        spec = SageMakerGroundTruthSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "LabelingJobName": "my-labeling-job",
                "LabelAttributeName": None,
                "InputConfig": {
                    "DataSource": {
                        "S3DataSource": {"ManifestS3Uri": "s3://fake-bucket/manifest"}
                    }
                },
                "OutputConfig": {
                    "S3OutputPath": "s3://fake-bucket/output",
                    "KmsKeyId": "",
                },
                "RoleArn": "arn:aws:iam::123456789012:user/Development/product_1234/*",
                "HumanTaskConfig": {
                    "WorkteamArn": None,
                    "UiConfig": {"UiTemplateS3Uri": "s3://fake-bucket/ui_template"},
                    "PreHumanTaskLambdaArn": "",
                    "TaskTitle": "fake-image-labelling-work",
                    "TaskDescription": "fake job",
                    "NumberOfHumanWorkersPerDataObject": 1,
                    "TaskTimeLimitInSeconds": 180,
                    "AnnotationConsolidationConfig": {
                        "AnnotationConsolidationLambdaArn": ""
                    },
                },
                "Tags": [],
            },
        )

    def test_create_ground_truth_job_all_args(self):
        spec = SageMakerGroundTruthSpec(
            self.REQUIRED_ARGS
            + [
                "--label_attribute_name",
                "fake-attribute",
                "--max_human_labeled_objects",
                "10",
                "--max_percent_objects",
                "50",
                "--enable_auto_labeling",
                "True",
                "--initial_model_arn",
                "fake-model-arn",
                "--task_availibility",
                "30",
                "--max_concurrent_tasks",
                "10",
                "--task_keywords",
                "fake-keyword",
                "--worker_type",
                "public",
                "--no_adult_content",
                "True",
                "--no_ppi",
                "True",
                "--tags",
                '{"fake_key": "fake_value"}',
            ]
        )

        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "LabelingJobName": "my-labeling-job",
                "LabelAttributeName": "fake-attribute",
                "InputConfig": {
                    "DataSource": {
                        "S3DataSource": {"ManifestS3Uri": "s3://fake-bucket/manifest"}
                    },
                    "DataAttributes": {
                        "ContentClassifiers": [
                            "FreeOfAdultContent",
                            "FreeOfPersonallyIdentifiableInformation",
                        ]
                    },
                },
                "OutputConfig": {
                    "S3OutputPath": "s3://fake-bucket/output",
                    "KmsKeyId": "",
                },
                "RoleArn": "arn:aws:iam::123456789012:user/Development/product_1234/*",
                "StoppingConditions": {
                    "MaxHumanLabeledObjectCount": 10,
                    "MaxPercentageOfInputDatasetLabeled": 50,
                },
                "LabelingJobAlgorithmsConfig": {
                    "LabelingJobAlgorithmSpecificationArn": "",
                    "InitialActiveLearningModelArn": "",
                    "LabelingJobResourceConfig": {"VolumeKmsKeyId": ""},
                },
                "HumanTaskConfig": {
                    "WorkteamArn": "arn:aws:sagemaker:us-west-2:394669845002:workteam/public-crowd/default",
                    "UiConfig": {"UiTemplateS3Uri": "s3://fake-bucket/ui_template"},
                    "PreHumanTaskLambdaArn": "",
                    "TaskKeywords": ["fake-keyword"],
                    "TaskTitle": "fake-image-labelling-work",
                    "TaskDescription": "fake job",
                    "NumberOfHumanWorkersPerDataObject": 1,
                    "TaskTimeLimitInSeconds": 180,
                    "TaskAvailabilityLifetimeInSeconds": 30,
                    "MaxConcurrentTaskCount": 10,
                    "AnnotationConsolidationConfig": {
                        "AnnotationConsolidationLambdaArn": ""
                    },
                    "PublicWorkforceTaskPrice": {
                        "AmountInUsd": {
                            "Dollars": 0,
                            "Cents": 0,
                            "TenthFractionsOfACent": 0,
                        }
                    },
                },
                "Tags": [{"Key": "fake_key", "Value": "fake_value"}],
            },
        )

    def test_get_job_status(self):
        self.component._sm_client = MagicMock()

        self.component._sm_client.describe_labeling_job.return_value = {
            "LabelingJobStatus": "Starting"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Starting"),
        )

        self.component._sm_client.describe_labeling_job.return_value = {
            "LabelingJobStatus": "Completed"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="Completed"),
        )

        self.component._sm_client.describe_labeling_job.return_value = {
            "LabelingJobStatus": "Failed",
            "FailureReason": "lolidk",
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(
                is_completed=True,
                raw_status="Failed",
                has_error=True,
                error_message="lolidk",
            ),
        )

    def test_after_job_completed(self):
        spec = SageMakerGroundTruthSpec(self.REQUIRED_ARGS)
        auto_labeling_spec = SageMakerGroundTruthSpec(
            self.REQUIRED_ARGS + ["--enable_auto_labeling", "True"]
        )

        self.component._sm_client = mock_client = MagicMock()
        mock_client.describe_labeling_job.return_value = {
            "LabelingJobOutput": {
                "OutputDatasetS3Uri": "s3://path/",
                "FinalActiveLearningModelArn": "model-arn",
            }
        }

        self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)
        self.assertEqual(spec.outputs.active_learning_model_arn, " ")
        self.assertEqual(spec.outputs.output_manifest_location, "s3://path/")

        self.component._after_job_complete(
            {}, {}, auto_labeling_spec.inputs, auto_labeling_spec.outputs
        )
        self.assertEqual(
            auto_labeling_spec.outputs.active_learning_model_arn, "model-arn"
        )
        self.assertEqual(
            auto_labeling_spec.outputs.output_manifest_location, "s3://path/"
        )
