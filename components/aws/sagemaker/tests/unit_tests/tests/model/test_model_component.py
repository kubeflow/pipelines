from common.sagemaker_component import SageMakerJobStatus
from model.src.sagemaker_model_spec import SageMakerCreateModelSpec
from model.src.sagemaker_model_component import SageMakerCreateModelComponent
from tests.unit_tests.tests.model.test_model_spec import CreateModelSpecTestCase
import unittest

from unittest.mock import patch, MagicMock, ANY


class CreateModelComponentTestCase(unittest.TestCase):
    REQUIRED_ARGS = CreateModelSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = SageMakerCreateModelComponent()
        # Instantiate without calling Do()
        cls.component._model_name = "test-model"

    @patch("model.src.sagemaker_model_component.super", MagicMock())
    def test_do_sets_name(self):
        named_spec = SageMakerCreateModelSpec(
            self.REQUIRED_ARGS + ["--model_name", "job-name"]
        )

        self.component.Do(named_spec)
        self.assertEqual("job-name", self.component._model_name)

    def test_create_model_job(self):
        spec = SageMakerCreateModelSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "ModelName": "test-model",
                "PrimaryContainer": {
                    "Image": "test-image",
                    "ModelDataUrl": "s3://fake-bucket/model_artifact",
                    "Environment": {},
                },
                "ExecutionRoleArn": "arn:aws:iam::123456789012:user/Development/product_1234/*",
                "Tags": [],
                "EnableNetworkIsolation": True,
            },
        )

    def test_create_model_job_arguments(self):
        spec = SageMakerCreateModelSpec(
            self.REQUIRED_ARGS
            + [
                "--container_host_name",
                "fake-host",
                "--tags",
                '{"fake_key": "fake_value"}',
                "--vpc_security_group_ids",
                "fake-ids",
                "--vpc_subnets",
                "fake-subnets",
            ]
        )

        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "ModelName": "test-model",
                "PrimaryContainer": {
                    "ContainerHostname": "fake-host",
                    "Image": "test-image",
                    "ModelDataUrl": "s3://fake-bucket/model_artifact",
                    "Environment": {},
                },
                "ExecutionRoleArn": "arn:aws:iam::123456789012:user/Development/product_1234/*",
                "Tags": [{"Key": "fake_key", "Value": "fake_value"}],
                "VpcConfig": {
                    "SecurityGroupIds": ["fake-ids"],
                    "Subnets": ["fake-subnets"],
                },
                "EnableNetworkIsolation": True,
            },
        )

    def test_create_model_job_model_package(self):
        arguments = [
            "--region",
            "us-west-2",
            "--model_name",
            "model_test",
            "--role",
            "arn:aws:iam::123456789012:user/Development/product_1234/*",
        ]

        # Should not raise any exceptions
        image_spec = SageMakerCreateModelSpec(
            arguments
            + [
                "--image",
                "test-image",
                "--model_artifact_url",
                "s3://fake-bucket/model_artifact",
            ]
        )
        self.component._create_job_request(image_spec.inputs, image_spec.outputs)

        # Should not raise any exceptions
        model_package_spec = SageMakerCreateModelSpec(
            arguments + ["--model_package", "fake-package"]
        )
        self.component._create_job_request(
            model_package_spec.inputs, model_package_spec.outputs
        )

        with self.assertRaises(Exception):
            bad_spec = SageMakerCreateModelSpec(arguments)
            self.component._create_job_request(bad_spec.inputs, bad_spec.outputs)
        with self.assertRaises(Exception):
            bad_spec = SageMakerCreateModelSpec(arguments + ["--image", "test-image"])
            self.component._create_job_request(bad_spec.inputs, bad_spec.outputs)
        with self.assertRaises(Exception):
            bad_spec = SageMakerCreateModelSpec(
                arguments + ["--model_artifact_url", "s3://fake-bucket/model_artifact"]
            )
            self.component._create_job_request(bad_spec.inputs, bad_spec.outputs)

    def test_get_job_status(self):
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status=""),
        )

    def test_after_job_completed(self):
        spec = SageMakerCreateModelSpec(self.REQUIRED_ARGS)

        self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

        self.assertEqual(spec.outputs.model_name, "test-model")
