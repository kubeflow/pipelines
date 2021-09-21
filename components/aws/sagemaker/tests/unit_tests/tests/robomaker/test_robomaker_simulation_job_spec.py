from simulation_job.src.robomaker_simulation_job_spec import RoboMakerSimulationJobSpec
import unittest
import json


class RoboMakerSimulationJobSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--role",
        "role-arn",
        "--output_bucket",
        "output-bucket-name",
        "--output_path",
        "output-bucket-key",
        "--max_run",
        "900",
        "--data_sources",
        json.dumps(
            [
                {
                    "name": "data-source-name",
                    "s3Bucket": "data-source-bucket",
                    "s3Keys": [{"s3Key": "data-source-key",}],
                }
            ]
        ),
        "--sim_app_arn",
        "simulation_app_arn",
        "--sim_app_version",
        "1",
        "--sim_app_launch_config",
        json.dumps(
            {
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
            }
        ),
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = RoboMakerSimulationJobSpec(self.REQUIRED_ARGS)
