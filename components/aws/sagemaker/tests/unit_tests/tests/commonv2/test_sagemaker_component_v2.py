import json
import os
from re import L
import unittest
from unittest.mock import mock_open, patch, call, MagicMock, ANY

from commonv2.common_inputs import (
    COMMON_INPUTS,
)
from commonv2.sagemaker_component import (
    ComponentMetadata,
    SageMakerComponent,
    SageMakerJobStatus,
)
from tests.unit_tests.tests.commonv2.dummy_spec import (
    DummyOutputs,
    DummySpec,
)
from tests.unit_tests.tests.commonv2.dummy_component import DummyComponent


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
        # cls.boto3_manager_patch = patch("commonv2.sagemaker_component.Boto3Manager")
        # cls.boto3_manager_patch.start()

        # set up mock job_request_outline_location
        test_files_dir = os.path.join(
            os.path.dirname(os.path.relpath(__file__)), "files"
        )
        test_job_request_outline_location = os.path.join(
            test_files_dir, "test_job_request_outline.yaml.tpl"
        )
        cls.component.job_request_outline_location = test_job_request_outline_location

    @classmethod
    def tearDown(cls):
        # cls.boto3_manager_patch.stop()
        pass

    def test_do_exits_with_error(self):
        self.component._init_configure_k8s = MagicMock()
        self.component._do = MagicMock(side_effect=Exception("Fire!"))

        # Expect the exception is raised up to root
        with self.assertRaises(Exception):
            self.component.Do(COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS)

    def test_do_initialization_error(self):
        self.component._init_configure_k8s = MagicMock(side_effect=Exception("Fire!"))

        with patch("commonv2.sagemaker_component.sys") as mock_sys:
            self.component.Do(COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS)

        mock_sys.exit.assert_called_with(1)

    def test_do_returns_false_exists(self):
        self.component._do = MagicMock(return_value=False)
        self.component._init_configure_k8s = MagicMock()

        with patch("commonv2.sagemaker_component.sys") as mock_sys:
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

        self.component._check_resource_conditions = MagicMock(return_value=None)
        self.component._get_resource = MagicMock(
            return_value={"status": {"ackResourceMetadata": {"arn": "test-arn"}}}
        )

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

    @patch("commonv2.sagemaker_component.logging")
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

        self.component._check_resource_conditions = MagicMock(return_value=None)
        self.component._get_resource = MagicMock(
            return_value={"status": {"ackResourceMetadata": {"arn": "test-arn"}}}
        )

        response = self.component._do(
            COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS
        )

        mock_logging.error.assert_any_call("abc123")
        self.component._after_job_complete.assert_not_called()
        self.assertFalse(response)

    @patch(
        "commonv2.sagemaker_component.strftime",
        MagicMock(return_value="20201231010203"),
    )
    @patch("commonv2.sagemaker_component.random.choice", MagicMock(return_value="A"))
    def test_generate_unique_timestamped_id(self):
        self.assertEqual(
            "20201231010203-aaaa", self.component._generate_unique_timestamped_id()
        )
        self.assertEqual(
            "20201231010203-aaaaaa",
            self.component._generate_unique_timestamped_id(size=6),
        )
        self.assertEqual(
            "PREFIX-20201231010203-aaaa",
            self.component._generate_unique_timestamped_id(prefix="PREFIX"),
        )
        self.assertEqual(
            "PREFIX-20201231010203-aaaaaa",
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

    @patch("commonv2.sagemaker_component.logging")
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

        with patch("commonv2.sagemaker_component.Path", MagicMock()) as mock_path:
            self.component._write_output(mock_output_path, "value1")

            mock_path.assert_has_calls(
                [
                    call(mock_output_path),
                    call().parent.mkdir(parents=True, exist_ok=True),
                    call(mock_output_path),
                    call().write_text("value1"),
                ]
            )

        with patch("commonv2.sagemaker_component.Path", MagicMock()) as mock_path:
            self.component._write_output(mock_output_path, "value2", json_encode=True)

            mock_path().write_text.assert_any_call('"value2"')

        with patch("commonv2.sagemaker_component.Path", MagicMock()) as mock_path:
            self.component._write_output(mock_output_path, ["value1"], json_encode=True)

            mock_path().write_text.assert_any_call(json.dumps(["value1"]))

    @patch.object(SageMakerComponent, "_get_k8s_api_client", MagicMock())
    @patch("kubernetes.client.CoreV1Api")
    def test_init_configure_k8s(self, mock_k8s_api):

        self.component.namespace = "test"

        self.component._init_configure_k8s()

        mock_k8s_api.assert_called_once()
        mock_k8s_api().list_namespaced_pod.assert_called_once_with(namespace="test")

    def test_get_current_namespace(self):

        with patch("os.path.exists", MagicMock(return_value=True)):
            with patch(
                "builtins.open",
                MagicMock(
                    return_value=open("tests/unit_tests/tests/commonv2/files/namespace")
                ),
            ) as mock_open:
                self.assertEqual(
                    self.component._get_current_namespace(), "namespace-in-file"
                )

        with patch("os.path.exists", MagicMock(return_value=False)):

            with patch(
                "kubernetes.config.list_kube_config_contexts"
            ) as mock_k8s_context:
                mock_k8s_context.return_value = (
                    " ",
                    {"context": {"namespace": "default-ns"}},
                )
                self.component._get_current_namespace()
                self.assertEqual("default-ns", self.component._get_current_namespace())

    @patch.object(SageMakerComponent, "_get_k8s_api_client", MagicMock())
    def test_wait_resource_consumed_by_controller(self):

        with patch.object(
            SageMakerComponent, "_get_resource_exists", return_value=True
        ) as mock_exists:
            self.component._get_resource = MagicMock(return_value={"status": "Running"})
            self.assertDictEqual(
                self.component._wait_resource_consumed_by_controller(5, 0.01),
                {"status": "Running"},
            )

        with patch.object(
            SageMakerComponent, "_get_resource_exists", return_value=True
        ) as mock_exists:
            self.component._get_resource_exists = MagicMock(return_value=True)
            self.component._get_resource = MagicMock(return_value={})
            self.component.job_name = "test"
            self.assertIsNone(
                self.component._wait_resource_consumed_by_controller(1, 0.01)
            )

        with patch.object(
            SageMakerComponent, "_get_resource_exists", return_value=False
        ) as mock_exists:
            self.component.namespace = "namespace-test"
            self.component.job_name = "ack-job-name-test"
            self.assertIsNone(
                self.component._wait_resource_consumed_by_controller(5, 0.01)
            )

    @patch.object(SageMakerComponent, "_get_k8s_api_client", MagicMock())
    @patch("kubernetes.client.CustomObjectsApi")
    def test_create_custom_resource(self, mock_custom_objects_api):

        cr_dict = {}
        self.component.group = "group-test"
        self.component.version = "version-test"
        self.component.plural = "plural-test"

        self.component._create_custom_resource(cr_dict)
        mock_custom_objects_api().create_cluster_custom_object.assert_called_once_with(
            "group-test", "version-test", "plural-test", {}
        )

        self.component.namespace = "namespace-test"
        self.component._create_custom_resource(cr_dict)
        mock_custom_objects_api().create_namespaced_custom_object.assert_called_once_with(
            "group-test", "version-test", "namespace-test", "plural-test", {}
        )

    @patch.object(SageMakerComponent, "_get_k8s_api_client", MagicMock())
    @patch("kubernetes.client.CustomObjectsApi")
    def test_get_resource(self, mock_custom_objects_api):
        self.component.group = "group-test"
        self.component.version = "version-test"
        self.component.plural = "plural-test"
        self.component.job_name = "ack-job-name-test"

        self.component._get_resource()
        mock_custom_objects_api().get_cluster_custom_object.assert_called_once_with(
            "group-test", "version-test", "plural-test", "ack-job-name-test"
        )

        self.component.namespace = "namespace-test"
        self.component._get_resource()
        mock_custom_objects_api().get_namespaced_custom_object.assert_called_once_with(
            "group-test",
            "version-test",
            "namespace-test",
            "plural-test",
            "ack-job-name-test",
        )

    @patch.object(SageMakerComponent, "_get_k8s_api_client", MagicMock())
    @patch("kubernetes.client.CustomObjectsApi")
    def test_delete_custom_resource(self, mock_custom_objects_api):
        self.component.group = "group-test"
        self.component.version = "version-test"
        self.component.plural = "plural-test"
        self.component.job_name = "ack-job-name-test"

        with patch.object(
            SageMakerComponent, "_get_resource_exists", return_value=False
        ) as mock_exists:
            _, ret_val = self.component._delete_custom_resource()
            mock_custom_objects_api().delete_cluster_custom_object.assert_called_once_with(
                "group-test", "version-test", "plural-test", "ack-job-name-test"
            )
            self.assertTrue(ret_val)

    def test_create_job_yaml(self):
        self.component.job_name = "ack-job-name-test"
        with patch(
            "builtins.open",
            MagicMock(return_value=open(self.component.job_request_outline_location)),
        ) as mock_open:
            Args = ["--input1", "abc123", "--input2", "123", "--region", "us-west-1"]
            nSpec = DummySpec(Args)
            
            result = self.component._create_job_yaml(nSpec._inputs, DummySpec.OUTPUTS)

            sample = {
                "apiVersion": "sagemaker.services.k8s.aws/v1alpha1",
                "kind": "TrainingJob",
                "metadata": {
                    "name": "ack-job-name-test",
                    "annotations": {"services.k8s.aws/region": "us-west-1"},
                },
                "spec": {
                    "spotInstance": False,
                    "maxWaitTime": 1,
                    "inputStr": "woof",
                    "inputInt": 1,
                    "inputBool": False,
                },
            }
            print("\n\n", result, "\n\n", sample, "\n\n")

            self.assertDictEqual(result, sample)

    def test_check_resource_conditions(self):
        ackdict_terminal = {
            "status": {"conditions": [{"type": "ACK.Terminal", "status": "True"}]}
        }
        ackdict_recoverable = {
            "status": {"conditions": [{"type": "ACK.Recoverable", "status": "True"}]}
        }
        ackdict_recoverable_validation = {
            "status": {
                "conditions": [
                    {
                        "type": "ACK.Recoverable",
                        "status": "True",
                        "message": "ValidationException",
                    }
                ]
            }
        }
        self.component._get_resource = MagicMock(return_value=ackdict_terminal)
        assert self.component._check_resource_conditions() == False
        self.component._get_resource = MagicMock(return_value=ackdict_recoverable)
        assert self.component._check_resource_conditions() == True
        self.component._get_resource = MagicMock(
            return_value=ackdict_recoverable_validation
        )
        assert self.component._check_resource_conditions() == False

    def test_resource_synced(self):
        ackdict = {"conditions": [{"type": "ACK.ResourceSynced", "status": "True"}]}
        res = self.component._get_resource_synced_status(ackdict)
        assert res == True
        ackdict["conditions"][0]["status"] = "False"
        res = self.component._get_resource_synced_status(ackdict)
        assert res == False


if __name__ == "__main__":
    unittest.main()
