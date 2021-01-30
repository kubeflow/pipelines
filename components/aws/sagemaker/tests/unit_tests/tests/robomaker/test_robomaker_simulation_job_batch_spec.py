from simulation_job_batch.src.robomaker_simulation_job_batch_spec import (
    RoboMakerSimulationJobBatchSpec,
)
import unittest
import json


class RoboMakerSimulationJobBatchSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--role",
        "role-arn",
        "--timeout_in_secs",
        "5800",
        "--max_concurrency",
        "3",
        "--simulation_job_requests",
        json.dumps(
            [
                {
                    "outputLocation": {
                        "s3Bucket": "fake-bucket",
                        "s3Prefix": "fake-key",
                    },
                    "loggingConfig": {"recordAllRosTopics": "True"},
                    "maxJobDurationInSeconds": "123",
                    "iamRole": "TestRole",
                    "failureBehavior": "Fail",
                    "simulationApplications": [
                        {
                            "application": "test-arn",
                            "applicationVersion": "1",
                            "launchConfig": {
                                "packageName": "package-name",
                                "launchFile": "launch-file.py",
                                "environmentVariables": {"Env": "var",},
                                "portForwardingConfig": {
                                    "portMappings": [
                                        {
                                            "jobPort": "123",
                                            "applicationPort": "123",
                                            "enableOnPublicIp": "True",
                                        }
                                    ]
                                },
                                "streamUI": "True",
                            },
                        }
                    ],
                    "dataSources": {
                        "name": "data-source-name",
                        "s3Bucket": "data-source-bucket",
                        "s3Keys": [{"s3Key": "data-source-key",}],
                    },
                }
            ]
        ),
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = RoboMakerSimulationJobBatchSpec(self.REQUIRED_ARGS)
