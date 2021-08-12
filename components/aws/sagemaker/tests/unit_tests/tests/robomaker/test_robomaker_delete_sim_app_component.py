from common.sagemaker_component import SageMakerJobStatus
from delete_simulation_app.src.robomaker_delete_simulation_app_spec import (
    RoboMakerDeleteSimulationAppSpec,
)
from delete_simulation_app.src.robomaker_delete_simulation_app_component import (
    RoboMakerDeleteSimulationAppComponent,
)
from tests.unit_tests.tests.robomaker.test_robomaker_delete_sim_app_spec import (
    RoboMakerDeleteSimAppSpecTestCase,
)
import unittest
import json

from unittest.mock import patch, MagicMock


class RoboMakerDeleteSimAppTestCase(unittest.TestCase):
    REQUIRED_ARGS = RoboMakerDeleteSimAppSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = RoboMakerDeleteSimulationAppComponent()
        # Instantiate without calling Do()
        cls.component._arn = "cool-arn"

    @patch(
        "delete_simulation_app.src.robomaker_delete_simulation_app_component.super",
        MagicMock(),
    )
    def test_do_sets_version(self):
        named_spec = RoboMakerDeleteSimulationAppSpec(
            self.REQUIRED_ARGS + ["--version", "cool-version"]
        )

        self.component.Do(named_spec)
        self.assertEqual("cool-version", self.component._version)

    def test_delete_simulation_application_request(self):
        spec = RoboMakerDeleteSimulationAppSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(request, {"application": "cool-arn",})

    def test_missing_required_input(self):
        missing_input = self.REQUIRED_ARGS.copy()
        missing_input.remove("--arn")
        missing_input.remove("cool-arn")

        with self.assertRaises(SystemExit):
            spec = RoboMakerDeleteSimulationAppSpec(missing_input)
            self.component._create_job_request(spec.inputs, spec.outputs)

    def test_get_job_status(self):
        self.component._rm_client = MagicMock()
        self.component._arn = "cool-arn"

        self.component._rm_client.describe_simulation_application.return_value = {
            "arn": None
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="Item deleted"),
        )

        self.component._rm_client.describe_simulation_application.return_value = {
            "arn": "arn:aws:robomaker:us-west-2:111111111111:simulation-application/MyRobotApplication/1551203301792"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(
                is_completed=False,
                raw_status="arn:aws:robomaker:us-west-2:111111111111:simulation-application/MyRobotApplication/1551203301792",
            ),
        )

    @patch(
        "delete_simulation_app.src.robomaker_delete_simulation_app_component.logging"
    )
    def test_after_submit_job_request(self, mock_logging):
        spec = RoboMakerDeleteSimulationAppSpec(self.REQUIRED_ARGS)
        self.component._after_submit_job_request(
            {"arn": "cool-arn"}, {}, spec.inputs, spec.outputs
        )
        mock_logging.info.assert_called_once()

    def test_after_job_completed(self):
        spec = RoboMakerDeleteSimulationAppSpec(self.REQUIRED_ARGS)

        mock_job_response = {}
        self.component._version = "cool-version"
        self.component._after_job_complete(
            mock_job_response, {}, spec.inputs, spec.outputs
        )

        # We expect to get returned the initial value we set for arn from REQUIRED_ARGS
        # The response from the api for the delete call will always be empty or None
        self.assertEqual(spec.outputs.arn, "cool-arn")
