from MonitoringSchedule.src.MonitoringSchedule_spec import (
    SageMakerMonitoringScheduleSpec,
)

import unittest


class MonitoringScheduleSpecTestCase(unittest.TestCase):
    REQUIRED_ARGS = [
        "--region",
        "us-west-1",
        "--monitoring_schedule_name",
        "test_monitoring_schedule_name",
        "--monitoring_schedule_config",
        "{'monitoringType': 'DataQuality', 'monitoringJobDefinitionName': 'test_monitoring_name', 'scheduleConfig': '' }",
    ]
    INCORRECT_ARGS = [
        "--region",
        "us-west-1",
        "--monitoring_schedule_name",
        "test_monitoring_schedule_name",
    ]

    def test_minimum_required_args(self):
        # Will raise an exception if the inputs are incorrect
        spec = SageMakerMonitoringScheduleSpec(self.REQUIRED_ARGS)

    def test_incorrect_args(self):
        # Will raise an exception if the inputs are incorrect
        with self.assertRaises(SystemExit):
            spec = SageMakerMonitoringScheduleSpec(self.INCORRECT_ARGS)


if __name__ == "__main__":
    unittest.main()
