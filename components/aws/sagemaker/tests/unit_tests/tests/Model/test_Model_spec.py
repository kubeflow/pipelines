from Model.src.Model_spec import SageMakerModelSpec

import unittest


class ModelSpecTestCase(unittest.TestCase):
    REQUIRED_ARGS = [
        "--region",
        "us-west-1",
        "--model_name",
        "test1",
        "--execution_role_arn",
        "arn",
    ]

    INCORRECT_ARGS = ["--empty"]

    def test_minimum_required_args(self):
        # Will raise an exception if the inputs are incorrect
        spec = SageMakerModelSpec(self.REQUIRED_ARGS)

    def test_incorrect_args(self):
        # Will raise an exception if the inputs are incorrect
        with self.assertRaises(SystemExit):
            spec = SageMakerModelSpec(self.INCORRECT_ARGS)


if __name__ == "__main__":
    unittest.main()
