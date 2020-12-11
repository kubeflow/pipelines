from delete_simulation_app.src.robomaker_delete_simulation_app_spec import (
    RoboMakerDeleteSimulationAppSpec,
)
import unittest


class RoboMakerDeleteSimAppSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-east-1",
        "--arn",
        "cool-arn",
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = RoboMakerDeleteSimulationAppSpec(self.REQUIRED_ARGS)
