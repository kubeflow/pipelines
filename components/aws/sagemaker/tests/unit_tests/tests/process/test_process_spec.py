from process.src.sagemaker_process_spec import SageMakerProcessSpec
import unittest
import json


class ProcessSpecTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-2",
        "--role",
        "arn:aws:iam::123456789012:user/Development/product_1234/*",
        "--image",
        "test-image",
        "--instance_type",
        "ml.m4.xlarge",
        "--instance_count",
        "1",
        "--input_config",
        json.dumps(
            [
                {
                    "InputName": "dataset-input",
                    "S3Input": {
                        "S3Uri": "s3://my-bucket/dataset.csv",
                        "LocalPath": "/opt/ml/processing/input",
                        "S3DataType": "S3Prefix",
                        "S3InputMode": "File",
                    },
                }
            ]
        ),
        "--output_config",
        json.dumps(
            [
                {
                    "OutputName": "training-outputs",
                    "S3Output": {
                        "S3Uri": "s3://my-bucket/outputs/train.csv",
                        "LocalPath": "/opt/ml/processing/output/train",
                        "S3UploadMode": "Continuous",
                    },
                }
            ]
        ),
    ]

    def test_minimum_required_args(self):
        # Will raise if the inputs are incorrect
        spec = SageMakerProcessSpec(self.REQUIRED_ARGS)
