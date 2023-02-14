import os
import unittest
from unittest.mock import patch, MagicMock

from commonv2.common_inputs import (
    COMMON_INPUTS,
)
from commonv2.sagemaker_component import (
    SageMakerComponent,
    SageMakerJobStatus,
)
from tests.unit_tests.tests.commonv2.dummy_spec import (
    DummySpec,
)

class SageMakerComponentReconcilationTestCase(unittest.TestCase):
    @classmethod
    def setUp(cls):
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
            SageMakerJobStatus(is_completed=True, raw_status="unreachable"),
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
            SageMakerJobStatus(is_completed=False, raw_status="unreachable"),
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
            SageMakerJobStatus(is_completed=True, raw_status="unreachable"),
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
            SageMakerJobStatus(is_completed=False, raw_status="unreachable"),
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

if __name__ == "__main__":
    unittest.main()