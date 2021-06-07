from common.sagemaker_component import SageMakerJobStatus
from create_simulation_app.src.robomaker_create_simulation_app_spec import (
    RoboMakerCreateSimulationAppSpec,
)
from create_simulation_app.src.robomaker_create_simulation_app_component import (
    RoboMakerCreateSimulationAppComponent,
)
from tests.unit_tests.tests.robomaker.test_robomaker_create_sim_app_spec import (
    RoboMakerCreateSimAppSpecTestCase,
)
import unittest

from unittest.mock import patch, MagicMock


class RoboMakerCreateSimAppTestCase(unittest.TestCase):
    REQUIRED_ARGS = RoboMakerCreateSimAppSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = RoboMakerCreateSimulationAppComponent()
        # Instantiate without calling Do()
        cls.component._app_name = "test-app"

    @patch(
        "create_simulation_app.src.robomaker_create_simulation_app_component.super",
        MagicMock(),
    )
    def test_do_sets_name(self):
        named_spec = RoboMakerCreateSimulationAppSpec(
            self.REQUIRED_ARGS + ["--app_name", "my-app-name"]
        )

        self.component.Do(named_spec)
        self.assertEqual("my-app-name", self.component._app_name)

    def test_create_simulation_application_request(self):
        spec = RoboMakerCreateSimulationAppSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "name": "test-app",
                "renderingEngine": {
                    "name": "rendering_engine_name",
                    "version": "rendering_engine_version",
                },
                "robotSoftwareSuite": {
                    "name": "robot_software_name",
                    "version": "robot_software_version",
                },
                "simulationSoftwareSuite": {
                    "name": "simulation_software_name",
                    "version": "simulation_software_version",
                },
                "sources": [
                    {
                        "architecture": "X86_64",
                        "s3Bucket": "sources_bucket",
                        "s3Key": "sources_key",
                    }
                ],
                "tags": {},
            },
        )

    def test_missing_required_input(self):
        missing_input = self.REQUIRED_ARGS.copy()
        missing_input.remove("--app_name")
        missing_input.remove("app-name")

        with self.assertRaises(SystemExit):
            spec = RoboMakerCreateSimulationAppSpec(missing_input)
            self.component._create_job_request(spec.inputs, spec.outputs)

    def test_get_job_status(self):
        self.component._rm_client = MagicMock()
        self.component._arn = "cool-arn"

        self.component._rm_client.describe_simulation_application.return_value = {
            "arn": None
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message="No ARN present",
                raw_status=None,
            ),
        )

        self.component._rm_client.describe_simulation_application.return_value = {
            "arn": "arn:aws:robomaker:us-west-2:111111111111:simulation-application/MyRobotApplication/1551203301792"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(
                is_completed=True,
                raw_status="arn:aws:robomaker:us-west-2:111111111111:simulation-application/MyRobotApplication/1551203301792",
            ),
        )

    @patch(
        "create_simulation_app.src.robomaker_create_simulation_app_component.logging"
    )
    def test_after_submit_job_request(self, mock_logging):
        spec = RoboMakerCreateSimulationAppSpec(self.REQUIRED_ARGS)
        self.component._after_submit_job_request(
            {"arn": "cool-arn"}, {}, spec.inputs, spec.outputs
        )
        mock_logging.info.assert_called_once()

    def test_after_job_completed(self):
        spec = RoboMakerCreateSimulationAppSpec(self.REQUIRED_ARGS)

        mock_job_response = {
            "arn": "arn:aws:robomaker:us-west-2:111111111111:simulation-application/MyRobotApplication/1551203301792",
            "version": "latest",
            "revisionId": "ee753e53-519c-4d37-895d-65e79bcd1914",
            "tags": {},
        }
        self.component._after_job_complete(
            mock_job_response, {}, spec.inputs, spec.outputs
        )

        self.assertEqual(
            spec.outputs.arn,
            "arn:aws:robomaker:us-west-2:111111111111:simulation-application/MyRobotApplication/1551203301792",
        )
