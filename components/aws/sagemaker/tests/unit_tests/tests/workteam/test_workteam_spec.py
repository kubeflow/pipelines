from workteam.src.sagemaker_workteam_spec import SageMakerWorkteamSpec
import unittest


class WorkteamSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--team_name",
        "test-team",
        "--description",
        "fake team",
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = SageMakerWorkteamSpec(self.REQUIRED_ARGS)
