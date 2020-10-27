from common.sagemaker_component import SageMakerComponent, SageMakerJobStatus
from batch_transform.src.sagemaker_transform_spec import SageMakerTransformSpec
from batch_transform.src.sagemaker_transform_component import (
    SageMakerTransformComponent,
)
from tests.unit_tests.tests.batch_transform.test_transform_spec import (
    TransformSpecTestCase,
)
import unittest

from unittest.mock import patch, MagicMock, ANY


class TransformComponentTestCase(unittest.TestCase):
    REQUIRED_ARGS = TransformSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = SageMakerTransformComponent()
        # Instantiate without calling Do()
        cls.component._transform_job_name = "test-job"

    @patch("batch_transform.src.sagemaker_transform_component.super", MagicMock())
    def test_do_sets_name(self):
        named_spec = SageMakerTransformSpec(
            self.REQUIRED_ARGS + ["--job_name", "job-name"]
        )
        unnamed_spec = SageMakerTransformSpec(self.REQUIRED_ARGS)

        with patch(
            "batch_transform.src.sagemaker_transform_component.SageMakerComponent._generate_unique_timestamped_id",
            MagicMock(return_value="BatchTransform-generated"),
        ):
            self.component.Do(named_spec)
            self.assertEqual("job-name", self.component._transform_job_name)

            self.component.Do(unnamed_spec)
            self.assertEqual(
                "BatchTransform-generated", self.component._transform_job_name
            )

    def test_create_transform_job(self):
        spec = SageMakerTransformSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "TransformJobName": "test-job",
                "ModelName": "model-test",
                "MaxConcurrentTransforms": 0,
                "MaxPayloadInMB": 6,
                "Environment": {},
                "TransformInput": {
                    "DataSource": {
                        "S3DataSource": {
                            "S3DataType": "S3Prefix",
                            "S3Uri": "s3://fake-bucket/data",
                        }
                    },
                    "ContentType": "",
                    "CompressionType": "None",
                    "SplitType": "None",
                },
                "TransformOutput": {
                    "S3OutputPath": "s3://fake-bucket/output",
                    "Accept": None,
                    "KmsKeyId": "",
                },
                "TransformResources": {
                    "InstanceType": "ml.c5.18xlarge",
                    "InstanceCount": 1,
                    "VolumeKmsKeyId": "",
                },
                "DataProcessing": {
                    "InputFilter": "",
                    "OutputFilter": "",
                    "JoinSource": "None",
                },
                "Tags": [],
            },
        )

    def test_create_transform_job_arguments(self):
        spec = SageMakerTransformSpec(
            self.REQUIRED_ARGS
            + [
                "--job_name",
                "test-batch-job",
                "--max_concurrent",
                "5",
                "--max_payload",
                "100",
                "--batch_strategy",
                "MultiRecord",
                "--data_type",
                "S3Prefix",
                "--compression_type",
                "Gzip",
                "--split_type",
                "RecordIO",
                "--assemble_with",
                "Line",
                "--join_source",
                "Input",
                "--tags",
                '{"fake_key": "fake_value"}',
            ]
        )

        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "TransformJobName": "test-job",
                "ModelName": "model-test",
                "MaxConcurrentTransforms": 5,
                "MaxPayloadInMB": 100,
                "BatchStrategy": "MultiRecord",
                "Environment": {},
                "TransformInput": {
                    "DataSource": {
                        "S3DataSource": {
                            "S3DataType": "S3Prefix",
                            "S3Uri": "s3://fake-bucket/data",
                        }
                    },
                    "ContentType": "",
                    "CompressionType": "Gzip",
                    "SplitType": "RecordIO",
                },
                "TransformOutput": {
                    "S3OutputPath": "s3://fake-bucket/output",
                    "Accept": None,
                    "AssembleWith": "Line",
                    "KmsKeyId": "",
                },
                "TransformResources": {
                    "InstanceType": "ml.c5.18xlarge",
                    "InstanceCount": 1,
                    "VolumeKmsKeyId": "",
                },
                "DataProcessing": {
                    "InputFilter": "",
                    "OutputFilter": "",
                    "JoinSource": "Input",
                },
                "Tags": [{"Key": "fake_key", "Value": "fake_value"}],
            },
        )

    def test_get_job_status(self):
        self.component._sm_client = MagicMock()

        self.component._sm_client.describe_transform_job.return_value = {
            "TransformJobStatus": "Starting"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Starting"),
        )

        self.component._sm_client.describe_transform_job.return_value = {
            "TransformJobStatus": "Downloading"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Downloading"),
        )

        self.component._sm_client.describe_transform_job.return_value = {
            "TransformJobStatus": "Completed"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="Completed"),
        )

        self.component._sm_client.describe_transform_job.return_value = {
            "TransformJobStatus": "Failed",
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
        spec = SageMakerTransformSpec(self.REQUIRED_ARGS)

        self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

        self.assertEqual(spec.outputs.output_location, "s3://fake-bucket/output")
