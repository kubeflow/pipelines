from deploy.src.sagemaker_deploy_spec import SageMakerDeploySpec
import unittest


class DeploySpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--model_name_1",
        "model-test",
        "--endpoint_name_output_path",
        "/tmp/output",
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = SageMakerDeploySpec(self.REQUIRED_ARGS)
