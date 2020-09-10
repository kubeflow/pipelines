from common.sagemaker_component import SageMakerJobStatus
from hyperparameter_tuning.src.sagemaker_tuning_spec import SageMakerTuningSpec
from hyperparameter_tuning.src.sagemaker_tuning_component import (
    SageMakerTuningComponent,
)
from tests.unit_tests.tests.hyperparameter_tuning.test_tuning_spec import (
    TuningSpecTestCase,
)
import unittest

from unittest.mock import patch, MagicMock, ANY


class TuningComponentTestCase(unittest.TestCase):
    REQUIRED_ARGS = TuningSpecTestCase.REQUIRED_ARGS

    @classmethod
    def setUp(cls):
        cls.component = SageMakerTuningComponent()
        # Instantiate without calling Do()
        cls.component._tuning_job_name = "test-job"

    @patch("hyperparameter_tuning.src.sagemaker_tuning_component.super", MagicMock())
    def test_do_sets_name(self):
        named_spec = SageMakerTuningSpec(
            self.REQUIRED_ARGS + ["--job_name", "job-name"]
        )
        unnamed_spec = SageMakerTuningSpec(self.REQUIRED_ARGS)

        self.component.Do(named_spec)
        self.assertEqual("job-name", self.component._tuning_job_name)

        with patch(
            "hyperparameter_tuning.src.sagemaker_tuning_component.SageMakerComponent._generate_unique_timestamped_id",
            MagicMock(return_value="unique"),
        ):
            self.component.Do(unnamed_spec)
            self.assertEqual("unique", self.component._tuning_job_name)

    def test_create_tuning_job(self):
        spec = SageMakerTuningSpec(self.REQUIRED_ARGS)
        request = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(
            request,
            {
                "HyperParameterTuningJobName": "test-job",
                "HyperParameterTuningJobConfig": {
                    "Strategy": "Bayesian",
                    "HyperParameterTuningJobObjective": {
                        "Type": "Maximize",
                        "MetricName": "test-metric",
                    },
                    "ResourceLimits": {
                        "MaxNumberOfTrainingJobs": 5,
                        "MaxParallelTrainingJobs": 2,
                    },
                    "ParameterRanges": {
                        "IntegerParameterRanges": [],
                        "ContinuousParameterRanges": [],
                        "CategoricalParameterRanges": [],
                    },
                    "TrainingJobEarlyStoppingType": "Off",
                },
                "TrainingJobDefinition": {
                    "StaticHyperParameters": {},
                    "AlgorithmSpecification": {
                        "TrainingImage": "test-image",
                        "TrainingInputMode": "File",
                    },
                    "RoleArn": "arn:aws:iam::123456789012:user/Development/product_1234/*",
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
                    "OutputDataConfig": {
                        "KmsKeyId": "",
                        "S3OutputPath": "test-output-location",
                    },
                    "ResourceConfig": {
                        "InstanceType": "ml.m4.xlarge",
                        "InstanceCount": 1,
                        "VolumeSizeInGB": 30,
                        "VolumeKmsKeyId": "",
                    },
                    "StoppingCondition": {"MaxRuntimeInSeconds": 86400},
                    "EnableNetworkIsolation": True,
                    "EnableInterContainerTrafficEncryption": False,
                    "EnableManagedSpotTraining": False,
                },
                "Tags": [],
            },
        )

    def test_get_job_status(self):
        self.component._sm_client = MagicMock()

        self.component._sm_client.describe_hyper_parameter_tuning_job.return_value = {
            "HyperParameterTuningJobStatus": "Starting"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Starting"),
        )

        self.component._sm_client.describe_hyper_parameter_tuning_job.return_value = {
            "HyperParameterTuningJobStatus": "Downloading"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=False, raw_status="Downloading"),
        )

        self.component._sm_client.describe_hyper_parameter_tuning_job.return_value = {
            "HyperParameterTuningJobStatus": "Completed"
        }
        self.assertEqual(
            self.component._get_job_status(),
            SageMakerJobStatus(is_completed=True, raw_status="Completed"),
        )

        self.component._sm_client.describe_hyper_parameter_tuning_job.return_value = {
            "HyperParameterTuningJobStatus": "Failed",
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
        spec = SageMakerTuningSpec(self.REQUIRED_ARGS)

        self.component._get_best_training_job_and_hyperparameters = MagicMock(
            return_value=("best-job", {"hp1": "val1"})
        )
        self.component._get_model_artifacts_from_job = MagicMock(
            return_value="model-url"
        )
        self.component._get_image_from_job = MagicMock(return_value="image-url")

        self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

        self.assertEqual(spec.outputs.best_hyperparameters, {"hp1": "val1"})
        self.assertEqual(spec.outputs.best_job_name, "best-job")
        self.assertEqual(spec.outputs.model_artifact_url, "model-url")
        self.assertEqual(spec.outputs.training_image, "image-url")
        self.assertEqual(spec.outputs.hpo_job_name, "test-job")

    def test_best_training_job(self):
        self.component._sm_client = mock_client = MagicMock()
        mock_client.describe_hyper_parameter_tuning_job.return_value = {
            "BestTrainingJob": {"TrainingJobName": "best_training_job"}
        }
        mock_client.describe_training_job.return_value = {
            "HyperParameters": {"hp": "val", "_tuning_objective_metric": "remove_me"}
        }

        name, params = self.component._get_best_training_job_and_hyperparameters()

        self.assertEqual("best_training_job", name)
        self.assertEqual("val", params["hp"])

    def test_warm_start_and_parents_args(self):
        # specifying both params
        spec = SageMakerTuningSpec(
            self.REQUIRED_ARGS
            + ["--warm_start_type", "TransferLearning"]
            + ["--parent_hpo_jobs", "A,B,C"]
        )
        response = self.component._create_job_request(spec.inputs, spec.outputs)
        self.assertIn("WarmStartConfig", response)
        self.assertIn("ParentHyperParameterTuningJobs", response["WarmStartConfig"])
        self.assertIn("WarmStartType", response["WarmStartConfig"])
        self.assertEqual(
            response["WarmStartConfig"]["ParentHyperParameterTuningJobs"][0][
                "HyperParameterTuningJobName"
            ],
            "A",
        )
        self.assertEqual(
            response["WarmStartConfig"]["ParentHyperParameterTuningJobs"][1][
                "HyperParameterTuningJobName"
            ],
            "B",
        )
        self.assertEqual(
            response["WarmStartConfig"]["ParentHyperParameterTuningJobs"][2][
                "HyperParameterTuningJobName"
            ],
            "C",
        )
        self.assertEqual(
            response["WarmStartConfig"]["WarmStartType"], "TransferLearning"
        )

    def test_either_warm_start_or_parents_args(self):
        # It will generate an exception if either warm_start_type or parent hpo jobs is being passed
        missing_parent_hpo_jobs_args = SageMakerTuningSpec(
            self.REQUIRED_ARGS + ["--warm_start_type", "TransferLearning"]
        )
        with self.assertRaises(Exception):
            self.component._create_job_request(
                missing_parent_hpo_jobs_args.inputs,
                missing_parent_hpo_jobs_args.outputs,
            )

        missing_warm_start_type_args = SageMakerTuningSpec(
            self.REQUIRED_ARGS + ["--parent_hpo_jobs", "A,B,C"]
        )
        with self.assertRaises(Exception):
            self.component._create_job_request(
                missing_warm_start_type_args.inputs,
                missing_warm_start_type_args.outputs,
            )

    def test_reasonable_required_args(self):
        spec = SageMakerTuningSpec(self.REQUIRED_ARGS)
        response = self.component._create_job_request(spec.inputs, spec.outputs)

        # Ensure all of the optional arguments have reasonable default values
        self.assertFalse(response["TrainingJobDefinition"]["EnableManagedSpotTraining"])
        self.assertDictEqual(
            response["TrainingJobDefinition"]["StaticHyperParameters"], {}
        )
        self.assertNotIn("VpcConfig", response["TrainingJobDefinition"])
        self.assertNotIn("MetricDefinitions", response["TrainingJobDefinition"])
        self.assertEqual(response["Tags"], [])
        self.assertEqual(
            response["TrainingJobDefinition"]["AlgorithmSpecification"][
                "TrainingInputMode"
            ],
            "File",
        )
        self.assertEqual(
            response["TrainingJobDefinition"]["OutputDataConfig"]["S3OutputPath"],
            "test-output-location",
        )

    def test_metric_definitions(self):
        metric_definition_args = SageMakerTuningSpec(
            self.REQUIRED_ARGS
            + [
                "--metric_definitions",
                '{"metric1": "regexval1", "metric2": "regexval2"}',
            ]
        )
        response = self.component._create_job_request(
            metric_definition_args.inputs, metric_definition_args.outputs
        )

        self.assertIn(
            "MetricDefinitions",
            response["TrainingJobDefinition"]["AlgorithmSpecification"],
        )
        response_metric_definitions = response["TrainingJobDefinition"][
            "AlgorithmSpecification"
        ]["MetricDefinitions"]

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

        parsed_args = SageMakerTuningSpec(no_image_args)

        with self.assertRaises(Exception):
            self.component._create_job_request(parsed_args.inputs, parsed_args.outputs)

    def test_first_party_algorithm(self):
        algorithm_name_args = SageMakerTuningSpec(
            self.REQUIRED_ARGS + ["--algorithm_name", "first-algorithm"]
        )

        # Should not throw an exception
        response = self.component._create_job_request(
            algorithm_name_args.inputs, algorithm_name_args.outputs
        )
        self.assertIn("TrainingJobDefinition", response)
        self.assertIn(
            "TrainingImage", response["TrainingJobDefinition"]["AlgorithmSpecification"]
        )
        self.assertNotIn(
            "AlgorithmName", response["TrainingJobDefinition"]["AlgorithmSpecification"]
        )

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

        parsed_args = SageMakerTuningSpec(known_algorithm_args)

        with patch(
            "hyperparameter_tuning.src.sagemaker_tuning_component.retrieve",
            MagicMock(return_value="seq2seq-url"),
        ) as mock_retrieve:
            response = self.component._create_job_request(
                parsed_args.inputs, parsed_args.outputs
            )

        mock_retrieve.assert_called_with("seq2seq", "us-west-2")
        self.assertEqual(
            response["TrainingJobDefinition"]["AlgorithmSpecification"][
                "TrainingImage"
            ],
            "seq2seq-url",
        )

    def test_known_algorithm_value(self):
        # This passes an algorithm that is a known SageMaker algorithm name
        known_algorithm_args = self.REQUIRED_ARGS + ["--algorithm_name", "seq2seq"]
        image_index = self.REQUIRED_ARGS.index("--image")
        # Cut out --image and it's associated value
        known_algorithm_args = (
            known_algorithm_args[:image_index] + known_algorithm_args[image_index + 2 :]
        )

        parsed_args = SageMakerTuningSpec(known_algorithm_args)

        with patch(
            "hyperparameter_tuning.src.sagemaker_tuning_component.retrieve",
            MagicMock(return_value="seq2seq-url"),
        ) as mock_retrieve:
            response = self.component._create_job_request(
                parsed_args.inputs, parsed_args.outputs
            )

        mock_retrieve.assert_called_with("seq2seq", "us-west-2")
        self.assertEqual(
            response["TrainingJobDefinition"]["AlgorithmSpecification"][
                "TrainingImage"
            ],
            "seq2seq-url",
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

        parsed_args = SageMakerTuningSpec(known_algorithm_args)

        with patch(
            "hyperparameter_tuning.src.sagemaker_tuning_component.retrieve",
            MagicMock(return_value="seq2seq-url"),
        ) as mock_retrieve:
            response = self.component._create_job_request(
                parsed_args.inputs, parsed_args.outputs
            )

        # Should just place the algorithm name in regardless
        mock_retrieve.assert_not_called()
        self.assertEqual(
            response["TrainingJobDefinition"]["AlgorithmSpecification"][
                "AlgorithmName"
            ],
            "unknown algorithm",
        )

    def test_no_channels(self):
        no_channels_args = self.REQUIRED_ARGS.copy()
        channels_index = self.REQUIRED_ARGS.index("--channels")
        # Replace the value after the flag with an empty list
        no_channels_args[channels_index + 1] = "[]"
        parsed_args = SageMakerTuningSpec(no_channels_args)

        with self.assertRaises(Exception):
            self.component._create_job_request(parsed_args.inputs, parsed_args.outputs)

    def test_tags(self):
        spec = SageMakerTuningSpec(
            self.REQUIRED_ARGS + ["--tags", '{"key1": "val1", "key2": "val2"}']
        )
        response = self.component._create_job_request(spec.inputs, spec.outputs)
        self.assertIn({"Key": "key1", "Value": "val1"}, response["Tags"])
        self.assertIn({"Key": "key2", "Value": "val2"}, response["Tags"])

    def test_valid_hyperparameters(self):
        hyperparameters_str = '{"hp1": "val1", "hp2": "val2", "hp3": "val3"}'
        categorical_params = '[{"Name" : "categorical", "Values": ["A", "B"]}]'
        integer_params = '[{"MaxValue": "integer_val1", "MinValue": "integer_val2", "Name": "integer", "ScalingType": "test_integer"}]'
        continuous_params = '[{"MaxValue": "continuous_val1", "MinValue": "continuous_val2", "Name": "continuous", "ScalingType": "test_continuous"}]'
        good_args = SageMakerTuningSpec(
            self.REQUIRED_ARGS
            + ["--static_parameters", hyperparameters_str]
            + ["--integer_parameters", integer_params]
            + ["--continuous_parameters", continuous_params]
            + ["--categorical_parameters", categorical_params]
        )
        response = self.component._create_job_request(
            good_args.inputs, good_args.outputs
        )

        self.assertIn("hp1", response["TrainingJobDefinition"]["StaticHyperParameters"])
        self.assertIn("hp2", response["TrainingJobDefinition"]["StaticHyperParameters"])
        self.assertIn("hp3", response["TrainingJobDefinition"]["StaticHyperParameters"])
        self.assertEqual(
            response["TrainingJobDefinition"]["StaticHyperParameters"]["hp1"], "val1"
        )
        self.assertEqual(
            response["TrainingJobDefinition"]["StaticHyperParameters"]["hp2"], "val2"
        )
        self.assertEqual(
            response["TrainingJobDefinition"]["StaticHyperParameters"]["hp3"], "val3"
        )

        self.assertIn("ParameterRanges", response["HyperParameterTuningJobConfig"])
        self.assertIn(
            "IntegerParameterRanges",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"],
        )
        self.assertIn(
            "ContinuousParameterRanges",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"],
        )
        self.assertIn(
            "CategoricalParameterRanges",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"],
        )
        self.assertIn(
            "Name",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "CategoricalParameterRanges"
            ][0],
        )
        self.assertIn(
            "Values",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "CategoricalParameterRanges"
            ][0],
        )
        self.assertIn(
            "MaxValue",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "IntegerParameterRanges"
            ][0],
        )
        self.assertIn(
            "MinValue",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "IntegerParameterRanges"
            ][0],
        )
        self.assertIn(
            "Name",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "IntegerParameterRanges"
            ][0],
        )
        self.assertIn(
            "ScalingType",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "IntegerParameterRanges"
            ][0],
        )
        self.assertIn(
            "MaxValue",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "ContinuousParameterRanges"
            ][0],
        )
        self.assertIn(
            "MinValue",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "ContinuousParameterRanges"
            ][0],
        )
        self.assertIn(
            "Name",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "ContinuousParameterRanges"
            ][0],
        )
        self.assertIn(
            "ScalingType",
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "ContinuousParameterRanges"
            ][0],
        )

        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "CategoricalParameterRanges"
            ][0]["Name"],
            "categorical",
        )
        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "CategoricalParameterRanges"
            ][0]["Values"][0],
            "A",
        )
        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "CategoricalParameterRanges"
            ][0]["Values"][1],
            "B",
        )
        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "IntegerParameterRanges"
            ][0]["MaxValue"],
            "integer_val1",
        )
        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "IntegerParameterRanges"
            ][0]["MinValue"],
            "integer_val2",
        )
        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "IntegerParameterRanges"
            ][0]["Name"],
            "integer",
        )
        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "IntegerParameterRanges"
            ][0]["ScalingType"],
            "test_integer",
        )
        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "ContinuousParameterRanges"
            ][0]["MaxValue"],
            "continuous_val1",
        )
        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "ContinuousParameterRanges"
            ][0]["MinValue"],
            "continuous_val2",
        )
        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "ContinuousParameterRanges"
            ][0]["Name"],
            "continuous",
        )
        self.assertEqual(
            response["HyperParameterTuningJobConfig"]["ParameterRanges"][
                "ContinuousParameterRanges"
            ][0]["ScalingType"],
            "test_continuous",
        )

    def test_empty_hyperparameters(self):
        hyperparameters_str = "{}"

        spec = SageMakerTuningSpec(
            self.REQUIRED_ARGS + ["--static_parameters", hyperparameters_str]
        )
        response = self.component._create_job_request(spec.inputs, spec.outputs)

        self.assertEqual(response["TrainingJobDefinition"]["StaticHyperParameters"], {})

    def test_object_hyperparameters(self):
        hyperparameters_str = '{"hp1": {"innerkey": "innerval"}}'

        invalid_args = SageMakerTuningSpec(
            self.REQUIRED_ARGS + ["--static_parameters", hyperparameters_str]
        )
        with self.assertRaises(Exception):
            self.component._create_job_request(
                invalid_args.inputs, invalid_args.outputs
            )

    def test_vpc_configuration(self):
        required_vpc_args = SageMakerTuningSpec(
            self.REQUIRED_ARGS
            + [
                "--vpc_security_group_ids",
                "sg1,sg2",
                "--vpc_subnets",
                "subnet1,subnet2",
            ]
        )
        response = self.component._create_job_request(
            required_vpc_args.inputs, required_vpc_args.outputs
        )

        self.assertIn("TrainingJobDefinition", response)
        self.assertIn("VpcConfig", response["TrainingJobDefinition"])
        self.assertIn(
            "sg1", response["TrainingJobDefinition"]["VpcConfig"]["SecurityGroupIds"]
        )
        self.assertIn(
            "sg2", response["TrainingJobDefinition"]["VpcConfig"]["SecurityGroupIds"]
        )
        self.assertIn(
            "subnet1", response["TrainingJobDefinition"]["VpcConfig"]["Subnets"]
        )
        self.assertIn(
            "subnet2", response["TrainingJobDefinition"]["VpcConfig"]["Subnets"]
        )

    def test_training_mode(self):
        required_vpc_args = SageMakerTuningSpec(
            self.REQUIRED_ARGS + ["--training_input_mode", "Pipe"]
        )
        response = self.component._create_job_request(
            required_vpc_args.inputs, required_vpc_args.outputs
        )

        self.assertEqual(
            response["TrainingJobDefinition"]["AlgorithmSpecification"][
                "TrainingInputMode"
            ],
            "Pipe",
        )
