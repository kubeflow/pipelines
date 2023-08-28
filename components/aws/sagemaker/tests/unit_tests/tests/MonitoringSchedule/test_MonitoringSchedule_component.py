from commonv2.sagemaker_component import SageMakerJobStatus

from unittest.mock import patch, MagicMock
import unittest
from MonitoringSchedule.src.MonitoringSchedule_spec import (
    SageMakerMonitoringScheduleSpec,
)
from MonitoringSchedule.src.MonitoringSchedule_component import (
    SageMakerMonitoringScheduleComponent,
)


class MonitoringScheduleComponentTestCase(unittest.TestCase):

    REQUIRED_ARGS = [
        "--region",
        "us-west-1",
        "--monitoring_schedule_name",
        "test",
        "--monitoring_schedule_config",
        "{'test': 'test'}",
    ]

    @classmethod
    def setUp(cls):
        cls.component = SageMakerMonitoringScheduleComponent()
        cls.component.job_name = "test"

    @patch("MonitoringSchedule.src.MonitoringSchedule_component.super", MagicMock())
    def test_do_sets_name(self):

        named_spec = SageMakerMonitoringScheduleSpec(self.REQUIRED_ARGS)
        with patch(
            "MonitoringSchedule.src.MonitoringSchedule_component.SageMakerComponent._get_current_namespace"
        ) as mock_namespace:
            mock_namespace.return_value = "test-namespace"
            self.component.Do(named_spec)
            self.assertEqual("test", self.component.job_name)

    def test_get_job_status(self):
        with patch(
            "MonitoringSchedule.src.MonitoringSchedule_component.SageMakerComponent._get_resource"
        ) as mock_get_resource:
            with patch(
                "MonitoringSchedule.src.MonitoringSchedule_component.SageMakerComponent._get_resource_synced_status"
            ) as mock_resource_sync:

                mock_resource_sync.return_value = False

                mock_get_resource.return_value = {
                    "status": {"monitoringScheduleStatus": "Pending"}
                }
                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(is_completed=False, raw_status="Pending"),
                )

                mock_resource_sync.return_value = True

                mock_get_resource.return_value = {
                    "status": {"monitoringScheduleStatus": "Scheduled"}
                }
                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(is_completed=True, raw_status="Scheduled"),
                )

                mock_get_resource.return_value = {
                    "status": {"monitoringScheduleStatus": "Stopped"}
                }
                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(
                        is_completed=True,
                        raw_status="Stopped",
                        has_error=True,
                        error_message="The schedule was stopped.",
                    ),
                )

                mock_get_resource.return_value = {
                    "status": {
                        "failureReason": "crash",
                        "monitoringScheduleStatus": "Failed",
                    }
                }

                self.assertEqual(
                    self.component._get_job_status(),
                    SageMakerJobStatus(
                        is_completed=True,
                        raw_status="Failed",
                        has_error=True,
                        error_message="crash",
                    ),
                )

    def test_after_job_completed(self):

        spec = SageMakerMonitoringScheduleSpec(self.REQUIRED_ARGS)

        statuses = {
            "status": {
                "ackResourceMetadata": 1,
                "conditions": 2,
            }
        }

        with patch(
            "MonitoringSchedule.src.MonitoringSchedule_component.SageMakerComponent._get_resource",
            MagicMock(return_value=statuses),
        ):

            self.component._after_job_complete({}, {}, spec.inputs, spec.outputs)

            self.assertEqual(spec.outputs.ack_resource_metadata, "1")
            self.assertEqual(spec.outputs.conditions, "2")
            self.assertEqual(spec.outputs.sagemaker_resource_name, "test")

    def test_get_upgrade_status(self):
        with patch(
            "MonitoringSchedule.src.MonitoringSchedule_component.SageMakerComponent._get_resource"
        ) as mock_get_resource:
            with patch(
                "MonitoringSchedule.src.MonitoringSchedule_component.SageMakerComponent._get_conditions_of_type"
            ) as mock_condition_list:

                mock_get_resource.return_value = {
                    "status": {
                        "monitoringScheduleStatus": "Scheduled",
                        "conditions": [
                            {"type": "ACK.ResourceSynced", "status": "True"}
                        ],
                    }
                }
                mock_condition_list.return_value = []
                self.assertEqual(
                    self.component._get_upgrade_status(),
                    SageMakerJobStatus(
                        is_completed=True, raw_status="Scheduled", has_error=False
                    ),
                )

                mock_get_resource.return_value = {
                    "status": {"monitoringScheduleStatus": "Updating"}
                }

                self.assertEqual(
                    self.component._get_upgrade_status(),
                    SageMakerJobStatus(
                        is_completed=False, raw_status="Updating", has_error=False
                    ),
                )

                mock_get_resource.return_value = {
                    "status": {
                        "monitoringScheduleStatus": "Failed",
                        "failureReason": "invalid config",
                        "conditions": [
                            {"type": "ACK.ResourceSynced", "status": "True"}
                        ],
                    }
                }

                # Failed Endpoint
                self.assertEqual(
                    self.component._get_upgrade_status(),
                    SageMakerJobStatus(
                        is_completed=True,
                        raw_status="Failed",
                        has_error=True,
                        error_message="invalid config",
                    ),
                )


if __name__ == "__main__":
    unittest.main()
