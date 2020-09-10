from common.sagemaker_component import SageMakerJobStatus
from train.src.sagemaker_training_spec import SageMakerTrainingSpec
from train.src.sagemaker_training_component import SageMakerTrainingComponent
from tests.unit_tests.tests.train.test_train_spec import TrainingSpecTestCase
import unittest

from unittest.mock import patch, MagicMock, ANY


class TrainingComponentTestCase(unittest.TestCase):
    REQUIRED_ARGS = TrainingSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = SageMakerTrainingComponent()
        # Instantiate without calling Do()
        cls.component._training_job_name = "test-job"

    @patch("train.src.sagemaker_training_component.super", MagicMock())
    def test_do_sets_name(self):
        named_spec = SageMakerTrainingSpec(
            self.REQUIRED_ARGS + ["--job_name", "job-name"]
        )
        unnamed_spec = SageMakerTrainingSpec(self.REQUIRED_ARGS)

        self.component.Do(named_spec)
        self.assertEqual("job-name", self.component._training_job_name)

        with patch(
            "train.src.sagemaker_training_component.SageMakerComponent._generate_unique_timestamped_id",
            MagicMock(return_value="unique"),
        ):
            self.component.Do(unnamed_spec)
            self.assertEqual("unique", self.component._training_job_name)

    def test_create_training_job(self):
        spec = SageMakerTrainingSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "AlgorithmSpecification": {
                    "TrainingImage": "test-image",
                    "TrainingInputMode": "File",
                },
                "EnableInterContainerTrafficEncryption": False,
                "EnableManagedSpotTraining": False,
                "EnableNetworkIsolation": True,
                "HyperParameters": {},
                "InputDataConfig": [
                    {
                        "ChannelName": "train",
                        "DataSource": {
                            "S3DataSource": {
                                "S3Uri": "s3://fake-bucket/data",
                                "S3DataType": "S3Prefix",
                                "S3DataDistributionType": "FullyReplicated",
                            }
                        },
                        "ContentType": "",
                        "CompressionType": "None",
                        "RecordWrapperType": "None",
                        "InputMode": "File",
                    }
                ],
                "OutputDataConfig": {"KmsKeyId": "", "S3OutputPath": "test-path"},
                "ResourceConfig": {
                    "InstanceType": "ml.m4.xlarge",
                    "InstanceCount": 1,
                    "VolumeSizeInGB": 50,
                    "VolumeKmsKeyId": "",
                },
                "RoleArn": "arn:aws:iam::123456789012:user/Development/product_1234/*",
                "StoppingCondition": {"MaxRuntimeInSeconds": 3600},
                "Tags": [],
                "TrainingJobName": "test-job",
            },
        )

    def test_get_job_status(self):
        self.component._sm_client = mock_client = MagicMock()

        self.component._sm_client.describe_training_job.return_value = {
            "TrainingJobStatus": "Starting"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Starting"),
        )

        self.component._sm_client.describe_training_job.return_value = {
            "TrainingJobStatus": "Downloading"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Downloading"),
        )

        self.component._sm_client.describe_training_job.return_value = {
            "TrainingJobStatus": "Completed"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="Completed"),
        )

        self.component._sm_client.describe_training_job.return_value = {
            "TrainingJobStatus": "Failed",
            "FailureReason": "lolidk",
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(
                is_completed=True,
                raw_status="Failed",
                has_error=True,
                error_message="lolidk",
            ),
        )

    def test_after_job_completed(self):
        self.component._get_model_artifacts_from_job = MagicMock(return_value="model")
        self.component._get_image_from_job = MagicMock(return_value="image")

        spec = SageMakerTrainingSpec(self.REQUIRED_ARGS)

        self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

        self.assertEqual(spec.outputs.job_name, "test-job")
        self.assertEqual(spec.outputs.model_artifact_url, "model")
        self.assertEqual(spec.outputs.training_image, "image")

    def test_get_model_artifacts_from_job(self):
        self.component._sm_client = mock_client = MagicMock()
        mock_client.describe_training_job.return_value = {
            "ModelArtifacts": {"S3ModelArtifacts": "s3://path/"}
        }

        self.assertEqual(self.component._get_model_artifacts_from_job(), "s3://path/")

    def test_get_image_from_defined_job(self):
        self.component._sm_client = mock_client = MagicMock()
        mock_client.describe_training_job.return_value = {
            "AlgorithmSpecification": {"TrainingImage": "training-image-url"}
        }

        self.assertEqual(self.component._get_image_from_job(), "training-image-url")

    def test_get_image_from_algorithm_job(self):
        self.component._sm_client = mock_client = MagicMock()
        mock_client.describe_training_job.return_value = {
            "AlgorithmSpecification": {"AlgorithmName": "my-algorithm"}
        }
        mock_client.describe_algorithm.return_value = {
            "TrainingSpecification": {"TrainingImage": "training-image-url"}
        }

        self.assertEqual(self.component._get_image_from_job(), "training-image-url")

    def test_metric_definitions(self):
        spec = SageMakerTrainingSpec(
            self.REQUIRED_ARGS
            + [
                "--metric_definitions",
                '{"metric1": "regexval1", "metric2": "regexval2"}',
            ]
        )
        response = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertIn("MetricDefinitions", response["AlgorithmSpecification"])
        response_metric_definitions = response["AlgorithmSpecification"][
            "MetricDefinitions"
        ]

        self.assertEqual(
            response_metric_definitions,
            [
                {"Name": "metric1", "Regex": "regexval1"},
                {"Name": "metric2", "Regex": "regexval2"},
            ],
        )

    def test_no_defined_image(self):
        # Pass the image to pass the parser
        no_image_args = self.REQUIRED_ARGS.copy()
        image_index = no_image_args.index("--image")
        # Cut out --image and it's associated value
        no_image_args = no_image_args[:image_index] + no_image_args[image_index + 2 :]

        spec = SageMakerTrainingSpec(no_image_args)

        with self.assertRaises(Exception):
            self.component._create_job_request(spec.inputs, spec.outputs)

    def test_first_party_algorithm(self):
        spec = SageMakerTrainingSpec(
            self.REQUIRED_ARGS + ["--algorithm_name", "first-algorithm"]
        )

        # Should not throw an exception
        response = self.component._create_job_request(spec.inputs, spec.outputs)
        self.assertIn("TrainingImage", response["AlgorithmSpecification"])
        self.assertNotIn("AlgorithmName", response["AlgorithmSpecification"])

    def test_known_algorithm_key(self):
        # This passes an algorithm that is a known NAME of an algorithm
        known_algorithm_args = self.REQUIRED_ARGS + [
            "--algorithm_name",
            "seq2seq modeling",
        ]
        image_index = self.REQUIRED_ARGS.index("--image")
        # Cut out --image and it's associated value
        known_algorithm_args = (
            known_algorithm_args[:image_index] + known_algorithm_args[image_index + 2 :]
        )

        spec = SageMakerTrainingSpec(known_algorithm_args)

        with patch(
            "train.src.sagemaker_training_component.get_image_uri",
            MagicMock(return_value="seq2seq-url"),
        ) as mock_get_image_uri:
            response = self.component._create_job_request(spec.inputs, spec.outputs)

        mock_get_image_uri.assert_called_with("us-west-2", "seq2seq")
        self.assertEqual(
            response["AlgorithmSpecification"]["TrainingImage"], "seq2seq-url"
        )

    def test_known_algorithm_value(self):
        # This passes an algorithm that is a known SageMaker algorithm name
        known_algorithm_args = self.REQUIRED_ARGS + ["--algorithm_name", "seq2seq"]
        image_index = self.REQUIRED_ARGS.index("--image")
        # Cut out --image and it's associated value
        known_algorithm_args = (
            known_algorithm_args[:image_index] + known_algorithm_args[image_index + 2 :]
        )

        spec = SageMakerTrainingSpec(known_algorithm_args)

        # Patch get_image_uri
        with patch(
            "train.src.sagemaker_training_component.get_image_uri",
            MagicMock(return_value="seq2seq-url"),
        ) as mock_get_image_uri:
            response = self.component._create_job_request(spec.inputs, spec.outputs)

        mock_get_image_uri.assert_called_with("us-west-2", "seq2seq")
        self.assertEqual(
            response["AlgorithmSpecification"]["TrainingImage"], "seq2seq-url"
        )

    def test_unknown_algorithm(self):
        known_algorithm_args = self.REQUIRED_ARGS + [
            "--algorithm_name",
            "unknown algorithm",
        ]
        image_index = self.REQUIRED_ARGS.index("--image")
        # Cut out --image and it's associated value
        known_algorithm_args = (
            known_algorithm_args[:image_index] + known_algorithm_args[image_index + 2 :]
        )

        spec = SageMakerTrainingSpec(known_algorithm_args)

        # Patch get_image_uri
        with patch(
            "train.src.sagemaker_training_component.get_image_uri",
            MagicMock(return_value="unknown-url"),
        ) as mock_get_image_uri:
            response = self.component._create_job_request(spec.inputs, spec.outputs)

        # Should just place the algorithm name in regardless
        mock_get_image_uri.assert_not_called()
        self.assertEqual(
            response["AlgorithmSpecification"]["AlgorithmName"], "unknown algorithm"
        )

    def test_no_channels(self):
        no_channels_args = self.REQUIRED_ARGS.copy()
        channels_index = self.REQUIRED_ARGS.index("--channels")
        # Replace the value after the flag with an empty list
        no_channels_args[channels_index + 1] = "[]"
        spec = SageMakerTrainingSpec(no_channels_args)

        with self.assertRaises(Exception):
            self.component._create_job_request(spec.inputs, spec.outputs)

    def test_valid_hyperparameters(self):
        hyperparameters_str = '{"hp1": "val1", "hp2": "val2", "hp3": "val3"}'

        spec = SageMakerTrainingSpec(
            self.REQUIRED_ARGS + ["--hyperparameters", hyperparameters_str]
        )
        response = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertIn("hp1", response["HyperParameters"])
        self.assertIn("hp2", response["HyperParameters"])
        self.assertIn("hp3", response["HyperParameters"])
        self.assertEqual(response["HyperParameters"]["hp1"], "val1")
        self.assertEqual(response["HyperParameters"]["hp2"], "val2")
        self.assertEqual(response["HyperParameters"]["hp3"], "val3")

    def test_empty_hyperparameters(self):
        hyperparameters_str = "{}"

        spec = SageMakerTrainingSpec(
            self.REQUIRED_ARGS + ["--hyperparameters", hyperparameters_str]
        )
        response = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(response["HyperParameters"], {})

    def test_object_hyperparameters(self):
        hyperparameters_str = '{"hp1": {"innerkey": "innerval"}}'

        spec = SageMakerTrainingSpec(
            self.REQUIRED_ARGS + ["--hyperparameters", hyperparameters_str]
        )
        with self.assertRaises(Exception):
            self.component._create_job_request(spec.inputs, spec.outputs)

    def test_vpc_configuration(self):
        spec = SageMakerTrainingSpec(
            self.REQUIRED_ARGS
            + [
                "--vpc_security_group_ids",
                "sg1,sg2",
                "--vpc_subnets",
                "subnet1,subnet2",
            ]
        )
        response = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertIn("VpcConfig", response)
        self.assertIn("sg1", response["VpcConfig"]["SecurityGroupIds"])
        self.assertIn("sg2", response["VpcConfig"]["SecurityGroupIds"])
        self.assertIn("subnet1", response["VpcConfig"]["Subnets"])
        self.assertIn("subnet2", response["VpcConfig"]["Subnets"])

    def test_training_mode(self):
        spec = SageMakerTrainingSpec(
            self.REQUIRED_ARGS + ["--training_input_mode", "Pipe"]
        )
        response = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            response["AlgorithmSpecification"]["TrainingInputMode"], "Pipe"
        )