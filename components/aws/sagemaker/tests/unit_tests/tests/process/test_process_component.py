from common.sagemaker_component import SageMakerJobStatus
from process.src.sagemaker_process_spec import SageMakerProcessSpec
from process.src.sagemaker_process_component import SageMakerProcessComponent
from tests.unit_tests.tests.process.test_process_spec import ProcessSpecTestCase
import unittest
import json

from unittest.mock import patch, MagicMock, ANY


class ProcessComponentTestCase(unittest.TestCase):
    REQUIRED_ARGS = ProcessSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = SageMakerProcessComponent()
        # Instantiate without calling Do()
        cls.component._processing_job_name = "test-job"

    @patch("process.src.sagemaker_process_component.super", MagicMock())
    def test_do_sets_name(self):
        named_spec = SageMakerProcessSpec(
            self.REQUIRED_ARGS + ["--job_name", "job-name"]
        )
        unnamed_spec = SageMakerProcessSpec(self.REQUIRED_ARGS)

        self.component.Do(named_spec)
        self.assertEqual("job-name", self.component._processing_job_name)

        with patch(
            "process.src.sagemaker_process_component.SageMakerComponent._generate_unique_timestamped_id",
            MagicMock(return_value="unique"),
        ):
            self.component.Do(unnamed_spec)
            self.assertEqual("unique", self.component._processing_job_name)

    def test_create_process_job(self):
        spec = SageMakerProcessSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "ProcessingJobName": "test-job",
                "ProcessingInputs": [
                    {
                        "InputName": "dataset-input",
                        "S3Input": {
                            "S3Uri": "s3://my-bucket/dataset.csv",
                            "LocalPath": "/opt/ml/processing/input",
                            "S3DataType": "S3Prefix",
                            "S3InputMode": "File",
                        },
                    }
                ],
                "ProcessingOutputConfig": {
                    "Outputs": [
                        {
                            "OutputName": "training-outputs",
                            "S3Output": {
                                "S3Uri": "s3://my-bucket/outputs/train.csv",
                                "LocalPath": "/opt/ml/processing/output/train",
                                "S3UploadMode": "Continuous",
                            },
                        }
                    ]
                },
                "RoleArn": "arn:aws:iam::123456789012:user/Development/product_1234/*",
                "ProcessingResources": {
                    "ClusterConfig": {
                        "InstanceType": "ml.m4.xlarge",
                        "InstanceCount": 1,
                        "VolumeSizeInGB": 30,
                    }
                },
                "NetworkConfig": {
                    "EnableInterContainerTrafficEncryption": False,
                    "EnableNetworkIsolation": True,
                },
                "StoppingCondition": {"MaxRuntimeInSeconds": 86400},
                "AppSpecification": {"ImageUri": "test-image"},
                "Environment": {},
                "Tags": [],
            },
        )

    def test_get_job_status(self):
        self.component._sm_client = MagicMock()

        self.component._sm_client.describe_processing_job.return_value = {
            "ProcessingJobStatus": "Starting"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Starting"),
        )

        self.component._sm_client.describe_processing_job.return_value = {
            "ProcessingJobStatus": "Downloading"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Downloading"),
        )

        self.component._sm_client.describe_processing_job.return_value = {
            "ProcessingJobStatus": "Completed"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="Completed"),
        )

        self.component._sm_client.describe_processing_job.return_value = {
            "ProcessingJobStatus": "Failed",
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
        spec = SageMakerProcessSpec(self.REQUIRED_ARGS)

        mock_out = {"out1": "val1", "out2": "val2"}
        self.component._get_job_outputs = MagicMock(return_value=mock_out)

        self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

        self.assertEqual(spec.outputs.job_name, "test-job")
        self.assertEqual(
            spec.outputs.output_artifacts, {"out1": "val1", "out2": "val2"}
        )

    def test_get_job_outputs(self):
        self.component._sm_client = mock_client = MagicMock()
        mock_client.describe_processing_job.return_value = {
            "ProcessingOutputConfig": {
                "Outputs": [
                    {"OutputName": "train", "S3Output": {"S3Uri": "s3://train"}},
                    {"OutputName": "valid", "S3Output": {"S3Uri": "s3://valid"}},
                ]
            }
        }

        self.assertEqual(
            self.component._get_job_outputs(),
            {"train": "s3://train", "valid": "s3://valid"},
        )

    def test_no_defined_image(self):
        # Pass the image to pass the parser
        no_image_args = self.REQUIRED_ARGS.copy()
        image_index = no_image_args.index("--image")
        # Cut out --image and it's associated value
        no_image_args = no_image_args[:image_index] + no_image_args[image_index + 2 :]

        with self.assertRaises(SystemExit):
            SageMakerProcessSpec(no_image_args)

    def test_container_entrypoint(self):
        entrypoint, arguments = ["/bin/bash"], ["arg1", "arg2"]

        container_args = SageMakerProcessSpec(
            self.REQUIRED_ARGS
            + [
                "--container_entrypoint",
                json.dumps(entrypoint),
                "--container_arguments",
                json.dumps(arguments),
            ]
        )
        response = self.component._create_job_request(
            container_args.inputs, container_args.outputs
        )

        self.assertEqual(
            response["AppSpecification"]["ContainerEntrypoint"], entrypoint
        )
        self.assertEqual(response["AppSpecification"]["ContainerArguments"], arguments)

    def test_environment_variables(self):
        env_vars = {"key1": "val1", "key2": "val2"}

        environment_args = SageMakerProcessSpec(
            self.REQUIRED_ARGS + ["--environment", json.dumps(env_vars)]
        )
        response = self.component._create_job_request(
            environment_args.inputs, environment_args.outputs
        )

        self.assertEqual(response["Environment"], env_vars)

    def test_vpc_configuration(self):
        required_vpc_args = SageMakerProcessSpec(
            self.REQUIRED_ARGS
            + [
                "--vpc_security_group_ids",
                "sg1,sg2",
                "--vpc_subnets",
                "subnet1,subnet2",
            ]
        )
        response = self.component._create_job_request(
            required_vpc_args.inputs, required_vpc_args.outputs
        )

        self.assertIn("VpcConfig", response["NetworkConfig"])
        self.assertIn("sg1", response["NetworkConfig"]["VpcConfig"]["SecurityGroupIds"])
        self.assertIn("sg2", response["NetworkConfig"]["VpcConfig"]["SecurityGroupIds"])
        self.assertIn("subnet1", response["NetworkConfig"]["VpcConfig"]["Subnets"])
        self.assertIn("subnet2", response["NetworkConfig"]["VpcConfig"]["Subnets"])
