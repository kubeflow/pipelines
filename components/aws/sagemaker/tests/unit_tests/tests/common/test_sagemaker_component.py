import json
import unittest
from unittest.mock import patch, call, MagicMock, ANY
from botocore.exceptions import ClientError

from common.common_inputs import (
    COMMON_INPUTS,
    SPOT_INSTANCE_INPUTS,
    SageMakerComponentBaseOutputs,
    SageMakerComponentCommonInputs,
    SpotInstanceInputs,
)
from common.sagemaker_component import (
    ComponentMetadata,
    SageMakerComponent,
    SageMakerJobStatus,
)
from tests.unit_tests.tests.common.dummy_spec import (
    DummyOutputs,
    DummySpec,
)
from tests.unit_tests.tests.common.dummy_component import DummyComponent

from common.sagemaker_component_spec import SageMakerComponentSpec


class SageMakerComponentMetadataTestCase(unittest.TestCase):
    def test_applies_constants(self):
        self.assertNotEqual(DummyComponent.COMPONENT_NAME, "test1")
        self.assertNotEqual(DummyComponent.COMPONENT_DESCRIPTION, "test2")
        self.assertNotEqual(DummyComponent.COMPONENT_SPEC, str)

        # Run decorator in function form
        ComponentMetadata("test1", "test2", str)(DummyComponent)

        self.assertEqual(DummyComponent.COMPONENT_NAME, "test1")
        self.assertEqual(DummyComponent.COMPONENT_DESCRIPTION, "test2")
        self.assertEqual(DummyComponent.COMPONENT_SPEC, str)


class SageMakerComponentTestCase(unittest.TestCase):
    MOCK_LICENSE_FILE = (
        """Amazon SageMaker Components for Kubeflow Pipelines; version 1.2.3"""
    )
    MOCK_BAD_VERSION_LICENSE_FILE = (
        """Amazon SageMaker Components for Kubeflow Pipelines; version WHATTHE"""
    )
    MOCK_BAD_FORMAT_LICENSE_FILE = """This isn't the first line"""

    @classmethod
    def setUp(cls):
        cls.component = SageMakerComponent()
        # Turn off polling interval for instant tests
        cls.component.STATUS_POLL_INTERVAL = 0
        cls.boto3_manager_patch = patch("common.sagemaker_component.Boto3Manager")
        cls.boto3_manager_patch.start()

    @classmethod
    def tearDown(cls):
        cls.boto3_manager_patch.stop()

    def test_do_exits_with_error(self):
        self.component._do = MagicMock(side_effect=Exception("Fire!"))

        # Expect the exception is raised up to root
        with self.assertRaises(Exception):
            self.component.Do(COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS)

    def test_do_returns_false_exists(self):
        self.component._do = MagicMock(return_value=False)

        with patch("common.sagemaker_component.sys") as mock_sys:
            self.component.Do(COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS)

        mock_sys.exit.assert_called_once_with(1)

    def test_do_returns_false_for_faulty_submit(self):
        self.component._submit_job_request = MagicMock(
            side_effect=Exception("Failed to create")
        )

        self.component._after_submit_job_request = MagicMock()

        response = self.component._do(
            COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS
        )

        self.assertFalse(response)
        self.component._after_submit_job_request.assert_not_called()

    def test_do_polls_for_status(self):
        self.component._get_job_status = MagicMock()
        self.component._get_job_status.side_effect = [
            SageMakerJobStatus(is_completed=False, raw_status="status1"),
            SageMakerJobStatus(is_completed=False, raw_status="status2"),
            SageMakerJobStatus(is_completed=True, raw_status="status3"),
            SageMakerJobStatus(is_completed=True, raw_status="don't reach"),
        ]

        self.component._after_job_complete = MagicMock()
        self.component._write_all_outputs = MagicMock()

        response = self.component._do(
            COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS
        )

        self.component._after_job_complete.assert_called()

        self.assertTrue(response)

    def test_do_poll_handles_exceptions(self):
        self.component._get_job_status = MagicMock()
        self.component._get_job_status.side_effect = [
            SageMakerJobStatus(is_completed=False, raw_status="status1"),
            SageMakerJobStatus(is_completed=False, raw_status="status2"),
            Exception("A random error occurred"),
            SageMakerJobStatus(is_completed=False, raw_status="don't reach"),
        ]

        self.component._after_job_complete = MagicMock()

        response = self.component._do(
            COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS
        )

        self.component._after_job_complete.assert_not_called()
        self.assertFalse(response)

    @patch("common.sagemaker_component.logging")
    def test_do_polls_for_status_catches_errors(self, mock_logging):
        self.component._get_job_status = MagicMock()
        self.component._get_job_status.side_effect = [
            SageMakerJobStatus(is_completed=False, raw_status="status1"),
            SageMakerJobStatus(is_completed=False, raw_status="status2"),
            SageMakerJobStatus(
                is_completed=True,
                raw_status="status3",
                has_error=True,
                error_message="abc123",
            ),
        ]

        self.component._after_job_complete = MagicMock()
        self.component._write_all_outputs = MagicMock()

        response = self.component._do(
            COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS
        )

        mock_logging.error.assert_any_call("abc123")
        self.component._after_job_complete.assert_not_called()
        self.assertFalse(response)

    @patch(
        "common.sagemaker_component.strftime", MagicMock(return_value="20201231010203")
    )
    @patch("common.sagemaker_component.random.choice", MagicMock(return_value="A"))
    def test_generate_unique_timestamped_id(self):
        self.assertEqual(
            "20201231010203-AAAA", self.component._generate_unique_timestamped_id()
        )
        self.assertEqual(
            "20201231010203-AAAAAA",
            self.component._generate_unique_timestamped_id(size=6),
        )
        self.assertEqual(
            "PREFIX-20201231010203-AAAA",
            self.component._generate_unique_timestamped_id(prefix="PREFIX"),
        )
        self.assertEqual(
            "PREFIX-20201231010203-AAAAAA",
            self.component._generate_unique_timestamped_id(prefix="PREFIX", size=6),
        )

    def test_write_all_outputs(self):
        self.component._write_output = MagicMock()

        mock_paths = DummyOutputs(output1="/tmp/output1", output2="/tmp/output2")
        mock_values = DummyOutputs(output1="value1", output2=["value2"])

        self.component._write_all_outputs(mock_paths, mock_values)

        self.component._write_output.assert_has_calls(
            [
                call("/tmp/output1", "value1", json_encode=False),
                call("/tmp/output2", ["value2"], json_encode=True),
            ]
        )

    @patch("common.sagemaker_component.logging")
    def test_write_all_outputs_invalid_output(self, mock_logging):
        self.component._write_output = MagicMock()

        mock_paths = DummyOutputs(output1="/tmp/output1", output2="/tmp/output2")
        mock_values = DummyOutputs(output1="value1", output2=["value2"])
        mock_values.extra_param = "How did this even get here?"  # type: ignore

        self.component._write_all_outputs(mock_paths, mock_values)

        mock_logging.error.assert_called_once()
        self.component._write_output.assert_has_calls(
            [
                call("/tmp/output1", "value1", json_encode=False),
                call("/tmp/output2", ["value2"], json_encode=True),
            ]
        )

    def test_write_output(self):
        mock_output_path = "/tmp/output1"

        with patch("common.sagemaker_component.Path", MagicMock()) as mock_path:
            self.component._write_output(mock_output_path, "value1")

            mock_path.assert_has_calls(
                [
                    call(mock_output_path),
                    call().parent.mkdir(parents=True, exist_ok=True),
                    call(mock_output_path),
                    call().write_text("value1"),
                ]
            )

        with patch("common.sagemaker_component.Path", MagicMock()) as mock_path:
            self.component._write_output(mock_output_path, "value2", json_encode=True)

            mock_path().write_text.assert_any_call('"value2"')

        with patch("common.sagemaker_component.Path", MagicMock()) as mock_path:
            self.component._write_output(mock_output_path, ["value1"], json_encode=True)

            mock_path().write_text.assert_any_call(json.dumps(["value1"]))

    @patch("builtins.open")
    def test_get_component_version(self, mock_open):
        mock_open().__enter__().readline.return_value = (
            SageMakerComponentTestCase.MOCK_LICENSE_FILE
        )
        self.assertEqual("1.2.3", self.component._get_component_version())

        mock_open().__enter__().readline.return_value = (
            SageMakerComponentTestCase.MOCK_BAD_FORMAT_LICENSE_FILE
        )
        self.assertEqual("NULL", self.component._get_component_version())

        mock_open().__enter__().readline.return_value = (
            SageMakerComponentTestCase.MOCK_BAD_VERSION_LICENSE_FILE
        )
        self.assertEqual("NULL", self.component._get_component_version())

    @patch("common.sagemaker_component.yaml")
    @patch("builtins.open")
    def test_get_request_template(self, mock_open, mock_yaml):
        mock_yaml_blob = "keyname: value"

        mock_open().__enter__.return_value = mock_yaml_blob
        mock_yaml.safe_load.return_value = {"keyname": "value"}

        with patch(
            "common.sagemaker_component.SageMakerComponent._get_common_path",
            MagicMock(return_value="/my/path"),
        ):
            response = self.component._get_request_template("my-template")

        mock_open.assert_any_call("/my/path/templates/my-template.template.yaml", "r")
        mock_yaml.safe_load.assert_called_once_with(mock_yaml_blob)
        self.assertEqual(response, {"keyname": "value"})

    def test_cw_logging_successfully(self):
        self.component._cw_client = mock_cw_client = MagicMock()
        mock_cw_client.describe_log_streams.return_value = {
            "logStreams": [
                {"logStreamName": "logStream1"},
                {"logStreamName": "logStream2"},
            ]
        }

        def my_get_log_events(logStreamName, **kwargs):
            if logStreamName == "logStream1":
                return {
                    "events": [
                        {"message": "fake log logStream1 line1"},
                        {"message": "fake log logStream1 line2"},
                    ]
                }
            elif logStreamName == "logStream2":
                return {
                    "events": [
                        {"message": "fake log logStream2 line1"},
                        {"message": "fake log logStream2 line2"},
                    ]
                }

        mock_cw_client.get_log_events.side_effect = my_get_log_events

        with patch("logging.Logger.info") as infoLog:
            self.component._print_cloudwatch_logs(
                "/aws/sagemaker/FakeJobs", "fake_job_name"
            )
            print(infoLog.call_args_list)
            calls = [
                call("fake log logStream1 line1"),
                call("fake log logStream1 line2"),
                call("fake log logStream2 line1"),
                call("fake log logStream2 line2"),
            ]
            infoLog.assert_has_calls(calls, any_order=True)

    def test_cw_logging_error(self):
        self.component._cw_client = mock_cw_client = MagicMock()
        mock_exception = ClientError(
            {"Error": {"Message": "CloudWatch broke"}}, "describe_log_streams"
        )
        mock_cw_client.describe_log_streams.side_effect = mock_exception

        with patch("logging.Logger.error") as errorLog:
            self.component._print_cloudwatch_logs(
                "/aws/sagemaker/FakeJobs", "fake_job_name"
            )
            errorLog.assert_called()


class ComponentFeatureTestCase(unittest.TestCase):
    class CommonInputsSpec(
        SageMakerComponentSpec[
            SageMakerComponentCommonInputs, SageMakerComponentBaseOutputs
        ]
    ):
        INPUTS = COMMON_INPUTS
        OUTPUTS = {}

        def __init__(self, arguments):
            super().__init__(
                arguments, SageMakerComponentCommonInputs, SageMakerComponentBaseOutputs
            )

    class SpotInstanceSpec(
        SageMakerComponentSpec[SpotInstanceInputs, SageMakerComponentBaseOutputs]
    ):
        INPUTS = SPOT_INSTANCE_INPUTS
        OUTPUTS = {}

        def __init__(self, arguments):
            super().__init__(
                arguments, SpotInstanceInputs, SageMakerComponentBaseOutputs
            )

    @ComponentMetadata(name="spot", description="spot", spec=SpotInstanceSpec)
    class SpotInstanceComponent(SageMakerComponent):
        pass

    @ComponentMetadata(name="common", description="common", spec=CommonInputsSpec)
    class CommonInputsComponent(SageMakerComponent):
        pass

    @classmethod
    def setUp(cls):
        # Load the train template as an example
        cls.template = SageMakerComponent._get_request_template("train")

    def test_spot_bad_args(self):
        no_max_wait_args = self.SpotInstanceSpec(["--spot_instance", "True"])
        no_checkpoint_args = self.SpotInstanceSpec(
            ["--spot_instance", "True", "--max_wait_time", "3601"]
        )
        no_s3_uri_args = self.SpotInstanceSpec(
            [
                "--spot_instance",
                "True",
                "--max_wait_time",
                "3601",
                "--max_run_time",
                "3600",
                "--checkpoint_config",
                "{}",
            ]
        )
        max_wait_too_short_args = self.SpotInstanceSpec(
            [
                "--spot_instance",
                "True",
                "--max_wait_time",
                "3600",
                "--max_run_time",
                "3601",
                "--checkpoint_config",
                "{}",
            ]
        )

        for arg in [
            no_max_wait_args,
            no_checkpoint_args,
            no_s3_uri_args,
            max_wait_too_short_args,
        ]:
            with self.assertRaises(Exception):
                SageMakerComponent._enable_spot_instance_support(
                    self.template, arg.inputs
                )

    def test_spot_good_args(self):
        good_args = self.SpotInstanceSpec(
            [
                "--spot_instance",
                "True",
                "--max_wait_time",
                "3601",
                "--max_run_time",
                "3600",
                "--checkpoint_config",
                '{"S3Uri": "s3://fake-uri/"}',
            ]
        )
        response = SageMakerComponent._enable_spot_instance_support(
            self.template, good_args.inputs
        )
        self.assertTrue(response["EnableManagedSpotTraining"])
        self.assertEqual(response["StoppingCondition"]["MaxWaitTimeInSeconds"], 3601)
        self.assertEqual(response["CheckpointConfig"]["S3Uri"], "s3://fake-uri/")

    def test_spot_local_path(self):
        args = self.SpotInstanceSpec(
            [
                "--spot_instance",
                "True",
                "--max_wait_time",
                "3601",
                "--max_run_time",
                "3600",
                "--checkpoint_config",
                '{"S3Uri": "s3://fake-uri/", "LocalPath": "local-path"}',
            ]
        )
        response = SageMakerComponent._enable_spot_instance_support(
            self.template, args.inputs
        )
        self.assertEqual(response["CheckpointConfig"]["S3Uri"], "s3://fake-uri/")
        self.assertEqual(response["CheckpointConfig"]["LocalPath"], "local-path")

    def test_spot_disabled_deletes_args(self):
        args = self.SpotInstanceSpec(["--spot_instance", "False"])
        response = SageMakerComponent._enable_spot_instance_support(
            self.template, args.inputs
        )
        self.assertNotIn("MaxWaitTimeInSeconds", response["StoppingCondition"])
        self.assertNotIn("CheckpointConfig", response)

    def test_create_hyperparameters(self):
        valid_params = {"tag1": "val1", "tag2": "val2"}
        invalid_params = {"tag1": 1500}

        self.assertEqual(
            valid_params, SageMakerComponent._validate_hyperparameters(valid_params)
        )
        with self.assertRaises(Exception):
            SageMakerComponent._validate_hyperparameters(invalid_params)

    def test_tags(self):
        spec = self.CommonInputsSpec(
            ["--region", "us-east-1", "--tags", '{"key1": "val1", "key2": "val2"}']
        )
        response = SageMakerComponent._enable_tag_support(self.template, spec.inputs)
        self.assertIn({"Key": "key1", "Value": "val1"}, self.template["Tags"])
        self.assertIn({"Key": "key2", "Value": "val2"}, self.template["Tags"])

    def test_get_model_artifacts_from_job(self):
        component = self.CommonInputsComponent()
        component._sm_client = mock_client = MagicMock()
        mock_client.describe_training_job.return_value = {
            "ModelArtifacts": {"S3ModelArtifacts": "s3://path/"}
        }

        self.assertEqual(
            component._get_model_artifacts_from_job("job-name"), "s3://path/"
        )

    def test_get_image_from_defined_job(self):
        component = self.CommonInputsComponent()
        component._sm_client = mock_client = MagicMock()
        mock_client.describe_training_job.return_value = {
            "AlgorithmSpecification": {"TrainingImage": "training-image-url"}
        }

        self.assertEqual(
            component._get_image_from_job("job-name"), "training-image-url"
        )

    def test_get_image_from_algorithm_job(self):
        component = self.CommonInputsComponent()
        component._sm_client = mock_client = MagicMock()
        mock_client.describe_training_job.return_value = {
            "AlgorithmSpecification": {"AlgorithmName": "my-algorithm"}
        }
        mock_client.describe_algorithm.return_value = {
            "TrainingSpecification": {"TrainingImage": "training-image-url"}
        }

        self.assertEqual(
            component._get_image_from_job("job-name"), "training-image-url"
        )
