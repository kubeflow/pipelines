from Endpoint.src.Endpoint_spec import SageMakerEndpointSpec

import unittest


class EndpointSpecTestCase(unittest.TestCase):
    REQUIRED_ARGS = [
        "--region",
        "us-west-1",
        "--endpoint_config_name",
        "sample_endpoint_config",
        "--endpoint_name",
        "xgboost_endpoint",
    ]
    INCORRECT_ARGS = ["--empty"]

    def test_minimum_required_args(self):
        # Will raise an exception if the inputs are incorrect
        spec = SageMakerEndpointSpec(self.REQUIRED_ARGS)

    def test_incorrect_args(self):
        # Will raise an exception if the inputs are incorrect
        with self.assertRaises(SystemExit):
            spec = SageMakerEndpointSpec(self.INCORRECT_ARGS)


if __name__ == "__main__":
    unittest.main()
