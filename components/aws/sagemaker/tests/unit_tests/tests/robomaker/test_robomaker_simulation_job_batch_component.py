from yaml.parser import ParserError

from common.sagemaker_component import SageMakerJobStatus
from simulation_job_batch.src.robomaker_simulation_job_batch_spec import (
    RoboMakerSimulationJobBatchSpec,
)
from simulation_job_batch.src.robomaker_simulation_job_batch_component import (
    RoboMakerSimulationJobBatchComponent,
)
from tests.unit_tests.tests.robomaker.test_robomaker_simulation_job_batch_spec import (
    RoboMakerSimulationJobBatchSpecTestCase,
)
import unittest

from unittest.mock import MagicMock


class RoboMakerSimulationJobTestCase(unittest.TestCase):
    REQUIRED_ARGS = RoboMakerSimulationJobBatchSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = RoboMakerSimulationJobBatchComponent()
        cls.component._arn = "fake-arn"
        cls.component._batch_job_id = "fake-id"
        cls.component._sim_request_ids = set()

    def test_create_simulation_batch_job(self):
        spec = RoboMakerSimulationJobBatchSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "batchPolicy": {"maxConcurrency": 3, "timeoutInSeconds": 5800},
                "createSimulationJobRequests": [
                    {
                        "dataSources": {
                            "name": "data-source-name",
                            "s3Bucket": "data-source-bucket",
                            "s3Keys": [{"s3Key": "data-source-key"}],
                        },
                        "failureBehavior": "Fail",
                        "iamRole": "TestRole",
                        "loggingConfig": {"recordAllRosTopics": "True"},
                        "maxJobDurationInSeconds": "123",
                        "outputLocation": {
                            "s3Bucket": "fake-bucket",
                            "s3Prefix": "fake-key",
                        },
                        "simulationApplications": [
                            {
                                "application": "test-arn",
                                "applicationVersion": "1",
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
                    }
                ],
                "tags": {},
            },
        )

    def test_get_job_status(self):
        self.component._rm_client = MagicMock()

        self.component._rm_client.describe_simulation_job_batch.return_value = {
            "status": "Starting"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Starting"),
        )

        self.component._rm_client.describe_simulation_job_batch.return_value = {
            "status": "Downloading"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Downloading"),
        )

        self.component._rm_client.describe_simulation_job_batch.return_value = {
            "status": "Completed",
            "createdRequests": [{"status": "Completed",}],
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="Completed"),
        )

        self.component._rm_client.describe_simulation_job_batch.return_value = {
            "status": "Canceled",
            "createdRequests": [{"status": "Failed", "arn": "fake-arn"}],
        }
        self.component._rm_client.describe_simulation_job.return_value = {
            "status": "Failed",
            "arn": "fake-arn",
            "failureCode": "InternalServiceError",
            "failureReason": "Big Reason",
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(
                is_completed=True,
                raw_status="Canceled",
                has_error=True,
                error_message="Simulation jobs are completed\nSimulation job: fake-arn failed with errorCode:InternalServiceError\n",
            ),
        )

        self.component._rm_client.describe_simulation_job_batch.return_value = {
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
                error_message="Simulation batch job is in status:Failed\nSimulation failed with reason:Big ReasonSimulation failed with errorCode:InternalServiceError",
            ),
        )

    def test_no_simulation_job_requests(self):
        no_job_requests = self.REQUIRED_ARGS.copy()
        no_job_requests = no_job_requests[
            : no_job_requests.index("--simulation_job_requests")
        ]

        with self.assertRaises(SystemExit):
            spec = RoboMakerSimulationJobBatchSpec(no_job_requests)
            self.component._create_job_request(spec.inputs, spec.outputs)

    def test_empty_simulation_job_requests(self):
        empty_job_requests = self.REQUIRED_ARGS.copy()
        empty_job_requests[-1:] = "[]"

        print(empty_job_requests)

        with self.assertRaises(ParserError):
            spec = RoboMakerSimulationJobBatchSpec(empty_job_requests)
            self.component._create_job_request(spec.inputs, spec.outputs)
