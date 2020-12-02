from common.sagemaker_component import SageMakerJobStatus
from simulation_job.src.robomaker_simulation_job_spec import RoboMakerSimulationJobSpec
from simulation_job.src.robomaker_simulation_job_component import (
    RoboMakerSimulationJobComponent,
)
from tests.unit_tests.tests.robomaker.test_robomaker_simulation_job_spec import (
    RoboMakerSimulationJobSpecTestCase,
)
import unittest

from unittest.mock import MagicMock


class RoboMakerSimulationJobTestCase(unittest.TestCase):
    REQUIRED_ARGS = RoboMakerSimulationJobSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = RoboMakerSimulationJobComponent()
        cls.component._arn = "fake-arn"
        cls.component._job_id = "fake-id"

    def test_create_simulation_job(self):
        spec = RoboMakerSimulationJobSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "outputLocation": {
                    "s3Bucket": "output-bucket-name",
                    "s3Prefix": "output-bucket-key",
                },
                "maxJobDurationInSeconds": 900,
                "iamRole": "role-arn",
                "failureBehavior": "Fail",
                "simulationApplications": [
                    {
                        "application": "simulation_app_arn",
                        "launchConfig": {
                            "environmentVariables": {"Env": "var"},
                            "launchFile": "launch-file.py",
                            "packageName": "package-name",
                            "portForwardingConfig": {
                                "portMappings": [
                                    {
                                        "applicationPort": "123",
                                        "enableOnPublicIp": "True",
                                        "jobPort": "123",
                                    }
                                ]
                            },
                            "streamUI": "True",
                        },
                    }
                ],
                "dataSources": [
                    {
                        "name": "data-source-name",
                        "s3Bucket": "data-source-bucket",
                        "s3Keys": [{"s3Key": "data-source-key"}],
                    }
                ],
                "compute": {"simulationUnitLimit": 15},
                "tags": {},
            },
        )

    def test_get_job_status(self):
        self.component._rm_client = MagicMock()

        self.component._rm_client.describe_simulation_job.return_value = {
            "status": "Starting"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Starting"),
        )

        self.component._rm_client.describe_simulation_job.return_value = {
            "status": "Downloading"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Downloading"),
        )

        self.component._rm_client.describe_simulation_job.return_value = {
            "status": "Completed"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="Completed"),
        )

        self.component._rm_client.describe_simulation_job.return_value = {
            "status": "Failed",
            "failureCode": "InternalServiceError",
            "failureReason": "Big Reason",
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(
                is_completed=True,
                raw_status="Failed",
                has_error=True,
                error_message="Simulation job is in status:Failed\nSimulation failed with reason:Big ReasonSimulation failed with errorCode:InternalServiceError",
            ),
        )

    def test_after_job_completed(self):
        spec = RoboMakerSimulationJobSpec(self.REQUIRED_ARGS)

        mock_out = "s3://cool-bucket/fake-key"
        self.component._get_job_outputs = MagicMock(return_value=mock_out)

        self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

        self.assertEqual(spec.outputs.output_artifacts, mock_out)

    def test_get_job_outputs(self):
        self.component._rm_client = mock_client = MagicMock()
        mock_client.describe_simulation_job.return_value = {
            "outputLocation": {"s3Bucket": "cool-bucket", "s3Prefix": "fake-key",}
        }

        self.assertEqual(
            self.component._get_job_outputs(), "s3://cool-bucket/fake-key",
        )

    def test_no_simulation_app_defined(self):
        no_sim_app = self.REQUIRED_ARGS.copy()
        no_sim_app.remove("--sim_app_arn")
        no_sim_app.remove("simulation_app_arn")

        with self.assertRaises(Exception):
            spec = RoboMakerSimulationJobSpec(no_sim_app)
            self.component._create_job_request(spec.inputs, spec.outputs)

    def test_no_launch_config_defined(self):
        no_launch_config = self.REQUIRED_ARGS.copy()
        no_launch_config = no_launch_config[
            : no_launch_config.index("--sim_app_launch_config")
        ]

        with self.assertRaises(Exception):
            spec = RoboMakerSimulationJobSpec(no_launch_config)
            self.component._create_job_request(spec.inputs, spec.outputs)
