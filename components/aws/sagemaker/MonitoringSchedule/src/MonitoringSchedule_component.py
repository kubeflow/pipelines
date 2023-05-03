# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import logging
from typing import Dict
import json

from MonitoringSchedule.src.MonitoringSchedule_spec import (
    SageMakerMonitoringScheduleInputs,
    SageMakerMonitoringScheduleOutputs,
    SageMakerMonitoringScheduleSpec,
)
from commonv2.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)
from commonv2 import snake_to_camel


@ComponentMetadata(
    name="SageMaker - MonitoringSchedule",
    description="",
    spec=SageMakerMonitoringScheduleSpec,
)
class SageMakerMonitoringScheduleComponent(SageMakerComponent):

    """SageMaker component for MonitoringSchedule."""

    def Do(self, spec: SageMakerMonitoringScheduleSpec):

        self.namespace = self._get_current_namespace()
        logging.info("Current namespace: " + self.namespace)

        ############GENERATED SECTION BELOW############

        self.job_name = spec.inputs.monitoring_schedule_name = (
            spec.inputs.monitoring_schedule_name
            if spec.inputs.monitoring_schedule_name
            else SageMakerComponent._generate_unique_timestamped_id(
                prefix="monitoring-schedule"
            )
        )

        self.group = "sagemaker.services.k8s.aws"
        self.version = "v1alpha1"
        self.plural = "monitoringschedules"
        self.spaced_out_resource_name = "Monitoring Schedule"

        self.job_request_outline_location = (
            "MonitoringSchedule/src/MonitoringSchedule_request.yaml.tpl"
        )
        self.job_request_location = (
            "MonitoringSchedule/src/MonitoringSchedule_request.yaml"
        )
        self.update_supported = True
        ############GENERATED SECTION ABOVE############

        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _create_job_request(
        self,
        inputs: SageMakerMonitoringScheduleInputs,
        outputs: SageMakerMonitoringScheduleOutputs,
    ) -> Dict:

        return super()._create_job_yaml(inputs, outputs)

    def _submit_job_request(self, request: Dict) -> object:

        if self.resource_upgrade:
            ack_resource = self._get_resource()
            self.initial_status = ack_resource.get("status", None)
            return super()._patch_custom_resource(request)
        else:
            return super()._create_resource(request, 12, 15)

    def _on_job_terminated(self):
        super()._delete_custom_resource()

    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerMonitoringScheduleInputs,
        outputs: SageMakerMonitoringScheduleOutputs,
    ):
        pass

    def _get_job_status(self):

        ack_resource = super()._get_resource()
        resourceSynced = self._get_resource_synced_status(ack_resource["status"])
        sm_job_status = ack_resource["status"]["monitoringScheduleStatus"]
        if not resourceSynced:
            return SageMakerJobStatus(
                is_completed=False,
                raw_status=sm_job_status,
            )

        if sm_job_status == "Scheduled":
            return SageMakerJobStatus(
                is_completed=True, has_error=False, raw_status="Scheduled"
            )

        if sm_job_status == "Failed":
            message = ack_resource["status"]["failureReason"]
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=message,
                raw_status=sm_job_status,
            )

        if sm_job_status == "Stopped":
            message = "The schedule was stopped."
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=message,
                raw_status=sm_job_status,
            )
        return SageMakerJobStatus(is_completed=False, raw_status=sm_job_status)

    def _get_upgrade_status(self):

        return self._get_job_status()

    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerMonitoringScheduleInputs,
        outputs: SageMakerMonitoringScheduleOutputs,
    ):
        # prepare component outputs (defined in the spec)

        ack_statuses = super()._get_resource()["status"]

        ############GENERATED SECTION BELOW############

        outputs.ack_resource_metadata = str(
            ack_statuses["ackResourceMetadata"]
            if "ackResourceMetadata" in ack_statuses
            else None
        )
        outputs.conditions = str(
            ack_statuses["conditions"] if "conditions" in ack_statuses else None
        )
        outputs.creation_time = str(
            ack_statuses["creationTime"] if "creationTime" in ack_statuses else None
        )
        outputs.failure_reason = str(
            ack_statuses["failureReason"] if "failureReason" in ack_statuses else None
        )
        outputs.last_modified_time = str(
            ack_statuses["lastModifiedTime"]
            if "lastModifiedTime" in ack_statuses
            else None
        )
        outputs.last_monitoring_execution_summary = str(
            ack_statuses["lastMonitoringExecutionSummary"]
            if "lastMonitoringExecutionSummary" in ack_statuses
            else None
        )
        outputs.monitoring_schedule_status = str(
            ack_statuses["monitoringScheduleStatus"]
            if "monitoringScheduleStatus" in ack_statuses
            else None
        )
        outputs.sagemaker_resource_name = self.job_name

        ############GENERATED SECTION ABOVE############


if __name__ == "__main__":
    import sys

    spec = SageMakerMonitoringScheduleSpec(sys.argv[1:])

    component = SageMakerMonitoringScheduleComponent()
    component.Do(spec)
