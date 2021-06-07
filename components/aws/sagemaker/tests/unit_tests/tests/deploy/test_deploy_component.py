from common.sagemaker_component import SageMakerComponent, SageMakerJobStatus
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
        cls.component._should_update_existing = False

    @patch("deploy.src.sagemaker_deploy_component.super", MagicMock())
    def test_do_sets_name(self):
        given_endpoint_name = SageMakerDeploySpec(
            self.REQUIRED_ARGS + ["--endpoint_name", "my-endpoint"]
        )
        given_endpoint_config_name = SageMakerDeploySpec(
            self.REQUIRED_ARGS + ["--endpoint_config_name", "my-endpoint-config"]
        )
        unnamed_spec = SageMakerDeploySpec(self.REQUIRED_ARGS)

        with patch(
            "deploy.src.sagemaker_deploy_component.SageMakerComponent._generate_unique_timestamped_id",
            MagicMock(return_value="-generated"),
        ):
            self.component.Do(given_endpoint_name)
            self.assertEqual(
                "EndpointConfig-generated", self.component._endpoint_config_name
            )
            self.assertEqual("my-endpoint", self.component._endpoint_name)

            self.component.Do(given_endpoint_config_name)
            self.assertEqual("my-endpoint-config", self.component._endpoint_config_name)
            self.assertEqual("Endpoint-generated", self.component._endpoint_name)

            self.component.Do(unnamed_spec)
            self.assertEqual(
                "EndpointConfig-generated", self.component._endpoint_config_name
            )
            self.assertEqual("Endpoint-generated", self.component._endpoint_name)

    @patch("deploy.src.sagemaker_deploy_component.super", MagicMock())
    def test_update_endpoint_do_sets_name(self):
        given_endpoint_name = SageMakerDeploySpec(
            self.REQUIRED_ARGS
            + ["--endpoint_name", "my-endpoint", "--update_endpoint", "True"]
        )
        given_endpoint_config_name = SageMakerDeploySpec(
            self.REQUIRED_ARGS
            + [
                "--endpoint_config_name",
                "my-endpoint-config",
                "--update_endpoint",
                "True",
            ]
        )
        unnamed_spec = SageMakerDeploySpec(self.REQUIRED_ARGS)
        SageMakerDeployComponent._generate_unique_timestamped_id = MagicMock(
            return_value="-generated-update"
        )
        self.component._endpoint_name_exists = MagicMock(return_value=True)
        self.component._get_endpoint_config = MagicMock(return_value="existing-config")

        with patch(
            "deploy.src.sagemaker_deploy_component.SageMakerComponent._generate_unique_timestamped_id",
            MagicMock(return_value="-generated-update"),
        ):
            self.component.Do(given_endpoint_name)
            self.assertEqual(
                "EndpointConfig-generated-update", self.component._endpoint_config_name
            )
            self.assertEqual("my-endpoint", self.component._endpoint_name)
            self.assertTrue(self.component._should_update_existing)

            # Ignore given endpoint config name for update
            self.component.Do(given_endpoint_config_name)
            self.assertEqual(
                "EndpointConfig-generated-update", self.component._endpoint_config_name
            )
            self.assertEqual("Endpoint-generated-update", self.component._endpoint_name)
            self.assertTrue(self.component._should_update_existing)

            self.component.Do(unnamed_spec)
            self.assertEqual(
                "EndpointConfig-generated-update", self.component._endpoint_config_name
            )
            self.assertEqual("Endpoint-generated-update", self.component._endpoint_name)
            self.assertFalse(self.component._should_update_existing)

    def test_create_deploy_job_requests(self):
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

    def test_create_update_deploy_job_requests(self):
        spec = SageMakerDeploySpec(self.REQUIRED_ARGS)
        self.component._should_update_existing = True
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
        self.component._sm_client = mock_client = MagicMock()

        self.component._sm_client.describe_endpoint.return_value = {
            "EndpointStatus": "Creating"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Creating"),
        )

        self.component._sm_client.describe_endpoint.return_value = {
            "EndpointStatus": "Updating"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Updating"),
        )

        self.component._sm_client.describe_endpoint.return_value = {
            "EndpointStatus": "InService"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="InService"),
        )

        self.component._sm_client.describe_endpoint.return_value = {
            "EndpointStatus": "Failed",
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
        spec = SageMakerDeploySpec(self.REQUIRED_ARGS)

        self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

        self.assertEqual(spec.outputs.endpoint_name, "endpoint")

    def test_submit_update_job_request(self):
        self.component._should_update_existing = True
        self.component._existing_endpoint_config_name = "old-config"
        self.component._delete_endpoint_config = MagicMock(return_value=True)
        self.component._sm_client = MagicMock()

        requests = EndpointRequests(
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
        )

        self.component._submit_job_request(requests)

        self.component._sm_client.update_endpoint.assert_called_once_with(
            **{"EndpointName": "endpoint", "EndpointConfigName": "endpoint-config",}
        )
        self.component._delete_endpoint_config.assert_called_once_with("old-config")
