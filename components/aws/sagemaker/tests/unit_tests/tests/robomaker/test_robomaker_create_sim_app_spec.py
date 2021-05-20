from create_simulation_app.src.robomaker_create_simulation_app_spec import (
    RoboMakerCreateSimulationAppSpec,
)
import unittest


class RoboMakerCreateSimAppSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--app_name",
        "app-name",
        "--sources",
        '[{"s3Bucket": "sources_bucket", "s3Key": "sources_key", "architecture": "X86_64"}]',
        "--simulation_software_name",
        "simulation_software_name",
        "--simulation_software_version",
        "simulation_software_version",
        "--robot_software_name",
        "robot_software_name",
        "--robot_software_version",
        "robot_software_version",
        "--rendering_engine_name",
        "rendering_engine_name",
        "--rendering_engine_version",
        "rendering_engine_version",
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = RoboMakerCreateSimulationAppSpec(self.REQUIRED_ARGS)
