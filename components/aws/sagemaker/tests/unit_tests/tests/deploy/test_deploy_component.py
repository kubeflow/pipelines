from common.sagemaker_component import SageMakerJobStatus
from deploy.src.sagemaker_deploy_spec import SageMakerDeploySpec
from deploy.src.sagemaker_deploy_component import (
    EndpointRequests,
    SageMakerDeployComponent,
)
from tests.unit_tests.tests.deploy.test_deploy_spec import DeploySpecTestCase
import unittest

from unittest.mock import patch, MagicMock, ANY


class DeployComponentTestCase(unittest.TestCase):
    REQUIRED_ARGS = DeploySpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = SageMakerDeployComponent()
        # Instantiate without calling Do()
        cls.component._endpoint_config_name = "endpoint-config"
        cls.component._endpoint_name = "endpoint"

    @patch("deploy.src.sagemaker_deploy_component.super", MagicMock())
    def test_do_sets_name(self):
        given_endpoint_name = SageMakerDeploySpec(
            self.REQUIRED_ARGS + ["--endpoint_name", "my-endpoint"]
        )
        given_endpoint_config_name = SageMakerDeploySpec(
            self.REQUIRED_ARGS + ["--endpoint_config_name", "my-endpoint-config"]
        )
        unnamed_spec = SageMakerDeploySpec(self.REQUIRED_ARGS)

        self.component.Do(given_endpoint_name)
        self.assertEqual("EndpointConfig-test", self.component._endpoint_config_name)
        self.assertEqual("my-endpoint", self.component._endpoint_name)

        self.component.Do(given_endpoint_config_name)
        self.assertEqual("my-endpoint-config", self.component._endpoint_config_name)
        self.assertEqual("Endpoint-endpoint-config", self.component._endpoint_name)

        self.component.Do(unnamed_spec)
        self.assertEqual("EndpointConfig-test", self.component._endpoint_config_name)
        self.assertEqual("Endpoint-test", self.component._endpoint_name)

    def test_create_deploy_job(self):
        spec = SageMakerDeploySpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            EndpointRequests(
                config_request={
                    "EndpointConfigName": "endpoint-config",
                    "ProductionVariants": [
                        {
                            "VariantName": "variant-name-1",
                            "ModelName": "model-test",
                            "InitialInstanceCount": 1,
                            "InstanceType": "ml.m4.xlarge",
                            "InitialVariantWeight": 1.0,
                        }
                    ],
                    "Tags": [],
                },
                endpoint_request={
                    "EndpointName": "endpoint",
                    "EndpointConfigName": "endpoint-config",
                },
            ),
        )

    def test_create_deploy_job_multiple_variants(self):
        spec = SageMakerDeploySpec(
            self.REQUIRED_ARGS
            + [
                "--variant_name_1",
                "variant-test-1",
                "--initial_instance_count_1",
                "1",
                "--instance_type_1",
                "t1",
                "--initial_variant_weight_1",
                "0.1",
                "--accelerator_type_1",
                "ml.eia1.medium",
                "--model_name_2",
                "model-test-2",
                "--variant_name_2",
                "variant-test-2",
                "--initial_instance_count_2",
                "2",
                "--instance_type_2",
                "t2",
                "--initial_variant_weight_2",
                "0.2",
                "--accelerator_type_2",
                "ml.eia1.large",
            ]
        )

        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            EndpointRequests(
                config_request={
                    "EndpointConfigName": "endpoint-config",
                    "ProductionVariants": [
                        {
                            "VariantName": "variant-test-1",
                            "ModelName": "model-test",
                            "InitialInstanceCount": 1,
                            "InstanceType": "t1",
                            "InitialVariantWeight": 0.1,
                            "AcceleratorType": "ml.eia1.medium",
                        },
                        {
                            "VariantName": "variant-test-2",
                            "ModelName": "model-test-2",
                            "InitialInstanceCount": 2,
                            "InstanceType": "t2",
                            "InitialVariantWeight": 0.2,
                            "AcceleratorType": "ml.eia1.large",
                        },
                    ],
                    "Tags": [],
                },
                endpoint_request={
                    "EndpointName": "endpoint",
                    "EndpointConfigName": "endpoint-config",
                },
            ),
        )

    def test_get_job_status(self):
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status=""),
        )

    def test_after_job_completed(self):
        spec = SageMakerDeploySpec(self.REQUIRED_ARGS)

        self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

        self.assertEqual(spec.outputs.endpoint_name, "endpoint")
