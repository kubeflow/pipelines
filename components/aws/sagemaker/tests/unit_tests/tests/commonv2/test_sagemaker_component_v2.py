import json
import os
import unittest
from unittest.mock import patch, call, MagicMock

from commonv2.sagemaker_component import (
    ComponentMetadata,
    SageMakerComponent,
)
from tests.unit_tests.tests.commonv2.dummy_spec import (
    DummyOutputs,
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
    @classmethod
    def setUp(cls):
        """Bootstrap Unit test resources for testing runtime.
        """
        cls.component = SageMakerComponent()
        # Turn off polling interval for instant tests
        cls.component.STATUS_POLL_INTERVAL = 0

        # set up mock job_request_outline_location
        test_files_dir = os.path.join(
            os.path.dirname(os.path.relpath(__file__)), "files"
        )
        test_job_request_outline_location = os.path.join(
            test_files_dir, "test_job_request_outline.yaml.tpl"
        )
        cls.component.job_request_outline_location = test_job_request_outline_location

    @patch(
        "commonv2.sagemaker_component.strftime",
        MagicMock(return_value="20201231010203"),
    )
    @patch("commonv2.sagemaker_component.random.choice", MagicMock(return_value="A"))
    def test_generate_unique_timestamped_id(self):
        """ Test timestamp generation
        """
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
        """ Test if component writes the output at the end of a pipeline run.
        """
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
        """ Test invalid write all output
        """
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
        """ Test write output function.
        """
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
        """ Test init k8 function
        """

        self.component.namespace = "test"

        self.component._init_configure_k8s()

        mock_k8s_api.assert_called_once()
        mock_k8s_api().list_namespaced_pod.assert_called_once_with(namespace="test")

    def test_get_current_namespace(self):
        """ Test namespace retrieval
        """

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
        """ Test initital resource poll.
        """

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

    def test_check_resource_conditions(self):
        """ Test check_resource_conditions works properly.
        """
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
        """" Tests _get_resource_synced_status method.
        """
        ackdict = {"conditions": [{"type": "ACK.ResourceSynced", "status": "True"}]}
        res = self.component._get_resource_synced_status(ackdict)
        assert res == True
        ackdict["conditions"][0]["status"] = "False"
        res = self.component._get_resource_synced_status(ackdict)
        assert res == False


if __name__ == "__main__":
    unittest.main()
