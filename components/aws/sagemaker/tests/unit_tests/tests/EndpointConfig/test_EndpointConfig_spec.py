from EndpointConfig.src.EndpointConfig_spec import SageMakerEndpointConfigSpec

import unittest


class EndpointSpecTestCase(unittest.TestCase):
    REQUIRED_ARGS = [
        "--region",
        "us-west-1",
        "--endpoint_config_name",
        "test",
        "--production_variants",
        "[]",
    ]
    INCORRECT_ARGS = ["--empty"]

    def test_minimum_required_args(self):
        # Will raise an exception if the inputs are incorrect
        spec = SageMakerEndpointConfigSpec(self.REQUIRED_ARGS)

    def test_incorrect_args(self):
        # Will raise an exception if the inputs are incorrect
        with self.assertRaises(SystemExit):
            spec = SageMakerEndpointConfigSpec(self.INCORRECT_ARGS)


if __name__ == "__main__":
    unittest.main()
