"""Base class for all SageMaker components."""
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

import os
import sys
import signal
import string
import logging
import json
from types import FunctionType
import yaml
import random
from pathlib import Path
from time import sleep, strftime, gmtime
from abc import abstractmethod
from typing import Any, Dict, List, NamedTuple, Optional
from kubernetes import client, config
from kubernetes.client.api_client import ApiClient
from kubernetes.client.rest import ApiException

from commonv2.sagemaker_component_spec import SageMakerComponentSpec

from commonv2.common_inputs import (
    SageMakerComponentBaseOutputs,
    SageMakerComponentCommonInputs,
)

from commonv2 import snake_to_camel, is_ack_requeue_error

# This handler is called whenever the @ComponentMetadata is applied.
# It allows the command line compiler to detect every component spec class.
_component_decorator_handler: Optional[FunctionType] = None


def ComponentMetadata(name: str, description: str, spec: object):
    """Decorator for SageMaker components.

    Used to define necessary metadata attributes about the component which will
    be used for logging output and for the component specification file.

    Usage:
    ```python
    @ComponentMetadata(
        name="SageMaker - Component Name",
        description="A cool new component we made!",
        spec=MyComponentSpec
    )
    """

    def _component_metadata(cls):
        cls.COMPONENT_NAME = name
        cls.COMPONENT_DESCRIPTION = description
        cls.COMPONENT_SPEC = spec

        # Add handler for compiler
        if _component_decorator_handler:
            return _component_decorator_handler(cls) or cls
        return cls

    return _component_metadata


class SageMakerJobStatus(NamedTuple):
    """Generic representation of a job status."""

    is_completed: bool
    raw_status: str
    has_error: bool = False
    error_message: Optional[str] = None


class SageMakerComponent:
    """Base class for a KFP SageMaker component.

    An instance of a subclass of this component represents an instantiation of the
    component within a pipeline run. Use the `@ComponentMetadata` decorator to
    modify the component attributes listed below.

    Attributes:
        COMPONENT_NAME: The name of the component as displayed to the user.
        COMPONENT_DESCRIPTION: The description of the component as displayed to
            the user.
        COMPONENT_SPEC: The correspending spec associated with the component.

        STATUS_POLL_INTERVAL: Number of seconds between polling for the job
            status.
    """

    COMPONENT_NAME = ""
    COMPONENT_DESCRIPTION = ""
    COMPONENT_SPEC = SageMakerComponentSpec

    STATUS_POLL_INTERVAL = 30
    UPDATE_PROCESS_INTERVAL = 10

    # parameters that will be filled by Do().
    # assignment statements in Do() will be genereated
    job_name: str
    group: str
    version: str
    plural: str
    spaced_out_resource_name: str # Used for Logs
    namespace: Optional[str] = None
    resource_upgrade: bool = False
    initial_status: dict
    update_supported: bool

    job_request_outline_location: str
    job_request_location: str

    def __init__(self):
        """Initialize a new component."""
        self._initialize_logging()

    def _initialize_logging(self):
        """Initializes the global logging structure."""
        logging.getLogger().setLevel(logging.INFO)

    def Do(
        self,
        inputs: SageMakerComponentCommonInputs,
        outputs: SageMakerComponentBaseOutputs,
        output_paths: SageMakerComponentBaseOutputs,
    ):
        """The main execution entrypoint for a component at runtime.

        Args:
            inputs: A populated list of user inputs.
            outputs: An unpopulated list of component output variables.
            output_paths: Paths to the respective output locations.
        """

        # Verify that the kubernetes cluster is available
        try:
            self._init_configure_k8s()
        except Exception as e:
            logging.exception("Failed to initialize k8s client: %s", e)
            sys.exit(1)

        # Global try-catch in order to allow for safe abort
        try:
            # Successful execution
            if not self._do(inputs, outputs, output_paths):
                sys.exit(1)
        except Exception as e:
            logging.exception("An error occurred while running the component")
            raise e

    def _init_configure_k8s(self):
        """Initializes the kubernetes client and configures the namespace."""

        _test_client = self._get_k8s_api_client()
        _test_api = client.CoreV1Api(_test_client)
        _test_api.list_namespaced_pod(namespace=self.namespace)

    def _get_k8s_api_client(self) -> ApiClient:
        """Create new client everytime to avoid token refresh issues."""

        # when run in k8s cluster
        config.load_incluster_config()
        return ApiClient()

    def _get_current_namespace(self):
        """
        Get the current namespace.
        """
        ns_path = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
        if os.path.exists(ns_path):
            with open(ns_path) as f:
                return f.read().strip()
        try:
            _, active_context = config.list_kube_config_contexts()
            return active_context["context"]["namespace"]
        except KeyError:
            return "default"

    def _do(
        self,
        inputs: SageMakerComponentCommonInputs,
        outputs: SageMakerComponentBaseOutputs,
        output_paths: SageMakerComponentBaseOutputs,
    ) -> bool:
        # Set up SIGTERM handling
        def signal_term_handler(signalNumber, frame):
            self._on_job_terminated()

        signal.signal(signal.SIGTERM, signal_term_handler)

        self.resource_upgrade = self._is_upgrade()
        if self.resource_upgrade and not self.update_supported:
            logging.error(
                f"Resource update is not supported for {self.spaced_out_resource_name}"
            )
            return False
        request = self._create_job_request(inputs, outputs)

        try:
            job = self._submit_job_request(request)
        except Exception as e:
            logging.exception(
                "An error occurred while attempting to submit the request"
            )
            return False

        created = self._verify_resource_consumption()
        if not created:
            return False

        self._after_submit_job_request(job, request, inputs, outputs)

        status: SageMakerJobStatus = SageMakerJobStatus(
            is_completed=False, raw_status="No Status"
        )
        try:
            while True:
                cr_condition = self._check_resource_conditions()
                if cr_condition:
                    sleep(self.STATUS_POLL_INTERVAL)
                    continue
                elif (
                    cr_condition == False
                ):  # ACK.Terminal or special errors (Validation Exception/Invalid Input)
                    return False

                status = (
                    self._get_job_status()
                    if not self.resource_upgrade
                    else self._get_upgrade_status()
                )
                # Continue until complete
                if status and status.is_completed:
                    if self.resource_upgrade:
                        logging.info(
                            f"{self.spaced_out_resource_name} Update complete, final status: {status.raw_status}"
                        )
                    else:
                        logging.info(
                            f"{self.spaced_out_resource_name} Creation complete, final status: {status.raw_status}"
                        )
                    break

                sleep(self.STATUS_POLL_INTERVAL)
                logging.info(
                    f"{self.spaced_out_resource_name} is in status: {status.raw_status}"
                )
        except Exception as e:
            logging.exception(
                f"An error occurred while polling for {self.spaced_out_resource_name} status"
            )
            return False

        if status.has_error:
            logging.error(status.error_message)
            return False

        self._after_job_complete(job, request, inputs, outputs)
        self._write_all_outputs(output_paths, outputs)

        return True

    def _get_conditions_of_type(self, condition_type):
        resource_conditions = self._get_resource()["status"]["conditions"]
        filtered_conditions = filter(
            lambda condition: (condition["type"] == condition_type), resource_conditions
        )
        return list(filtered_conditions)

    def _verify_resource_consumption(self) -> bool:
        """Verify that the resource has been successfully consumed by the controller.
            In the case of an update verify that the job arn exists.

        Returns:
            bool: Whether the resource consumed by the controller.
        """
        submission_ack_printed = False
        ERROR_NOT_CREATED_MESSAGE = "An error occurred while getting resource arn, ACK CR created but Sagemaker resource not created."
        ERROR_UPDATE_MESSAGE = "An error occured when getting the resource arn. Check the ACK Sagemaker Controller logs."

        try:
            while True:
                cr_condition = self._check_resource_conditions()
                if cr_condition:  # ACK.Recoverable
                    sleep(self.STATUS_POLL_INTERVAL)
                    continue
                elif cr_condition == False:
                    if (
                        self.resource_upgrade
                        and not self.is_update_consumed_by_controller()
                    ):
                        sleep(self.UPDATE_PROCESS_INTERVAL)
                        continue
                    return False

                # Retrieve Sagemaker ARN
                arn = self.check_resource_initiation(submission_ack_printed)

                # Continue until complete
                if arn:
                    submission_ack_printed = True
                    if (
                        self.resource_upgrade
                        and not self.is_update_consumed_by_controller()
                    ):
                        sleep(self.UPDATE_PROCESS_INTERVAL)
                        continue
                    break

                sleep(self.STATUS_POLL_INTERVAL)
                logging.info(f"Getting arn for {self.job_name}")
        except Exception as e:
            err_msg = (
                ERROR_UPDATE_MESSAGE
                if self.resource_upgrade
                else ERROR_NOT_CREATED_MESSAGE
            )
            logging.exception(err_msg)
            return False
        return True

    def check_resource_initiation(self, submission_ack_printed: bool):
        """ Check if resource has been initiated in Sagemaker. 
            A resource is considered to be initiated if the resource ARN is present in the ack resource metadata.
            If the resource ARN is present in the ack resource metadata, the resource has been successfully
            created in Sagemaker.

        Args:
            submission_ack_printed (bool): Parameter to avoid printing the resource consumed message
                multiple times.

        Returns:
            str: The ARN of the resource. If the resource ARN is not present in the ack resource metadata,
                the resource has not been created in Sagemaker.
        """
        ack_status = self._get_resource()["status"]
        ack_resource_meta = ack_status.get("ackResourceMetadata", None)
        if ack_resource_meta:
            arn = ack_resource_meta.get("arn", None)
            if arn is not None:
                if submission_ack_printed:
                    resource_consumed_message = (
                        f"Created Sagemaker {self.spaced_out_resource_name} with ARN: {arn}"
                        if not self.resource_upgrade
                        else f"Submitting update for Sagemaker {self.spaced_out_resource_name} with ARN: {arn}"
                    )
                    logging.info(resource_consumed_message)
                return arn
        return None

    @abstractmethod
    def _get_job_status(self) -> SageMakerJobStatus:
        """Waits for the current job to complete.

        Returns:
            SageMakerJobStatus: A status object.
        """
        pass

    @abstractmethod
    def _get_upgrade_status(self) -> SageMakerJobStatus:
        """Waits for the resource upgrade to complete

        Returns:
            SageMakerJobStatus: A status object.
        """
        pass

    def is_update_consumed_by_controller(self):
        """Check if update has been consumed by the controller, in this case it is done by
        checking whether
        """
        current_resource = self._get_resource()
        current_status = current_resource.get("status", None)
        ## Python == is deep equal between dicts.
        if current_status == self.initial_status:
            return False
        return True

    def _get_resource(self):
        """Get the custom resource detail similar to: kubectl describe
        trainingjob JOB_NAME -n NAMESPACE.

        Returns:
            None or object: None if the resource doesnt exist in server, otherwise the
                custom object.
        """

        _api_client = self._get_k8s_api_client()
        _api = client.CustomObjectsApi(_api_client)

        if self.namespace is None:
            job_description = _api.get_cluster_custom_object(
                self.group.lower(),
                self.version.lower(),
                self.plural.lower(),
                self.job_name.lower(),
            )
        else:
            job_description = _api.get_namespaced_custom_object(
                self.group.lower(),
                self.version.lower(),
                self.namespace.lower(),  # "default",
                self.plural.lower(),
                self.job_name.lower(),
            )

        return job_description

    @abstractmethod
    def _create_job_request(
        self,
        inputs: SageMakerComponentCommonInputs,
        outputs: SageMakerComponentBaseOutputs,
    ) -> Dict:
        """Creates the ACK custom object.

        Args:
            inputs: A populated list of user inputs.
            outputs: An unpopulated list of component output variables.

        Returns:
            dict: A dictionary object representing the custom object.
        """
        pass

    def _create_job_yaml(
        self,
        inputs: SageMakerComponentCommonInputs,
        outputs: SageMakerComponentBaseOutputs,
    ) -> Dict:
        """Creates the ACK request object to execute the component.

        Args:
            inputs: A populated list of user inputs.
            outputs: An unpopulated list of component output variables.

        Returns:
            dict: A dictionary object representing the request.
        """
        with open(self.job_request_outline_location) as job_request_outline:
            job_request_dict = yaml.load(job_request_outline, Loader=yaml.FullLoader)
            job_request_spec = job_request_dict["spec"]

            # populate meta data
            job_request_dict["metadata"]["name"] = self.job_name
            job_request_dict["metadata"]["annotations"][
                "services.k8s.aws/region"
            ] = getattr(inputs, "region")

            # populate spec from inputs
            for para in vars(inputs):
                camel_para = snake_to_camel(para)
                if camel_para in job_request_spec:
                    value = getattr(inputs, para)
                    if value not in [{}, []]:
                        job_request_spec[camel_para] = value

            # clean up empty fields in job_request_spec
            filtered = {k: v for k, v in job_request_spec.items() if v is not None}
            job_request_spec.clear()
            job_request_spec.update(filtered)
            job_request_dict["spec"] = job_request_spec

            logging.info(f"Custom resource: {json.dumps(job_request_dict, indent=2)}")

        return job_request_dict

    @abstractmethod
    def _submit_job_request(self, request: Dict) -> Dict:
        """Submits a pre-defined request object to SageMaker.

        The `request` argument should be provided as the result of the
        `_create_job_request` method.

        Args:
            request: A request object to execute the component.

        Returns:
            dict: The job object that was created.

        Raises:
            Exception: If SageMaker responded with an error during the request.
        """
        pass

    def _patch_custom_resource(self, custom_resource: dict):
        """Patch a custom resource in ACK

        Args:
            custom_resource: A dictionary object representing the custom object.
        Returns:
            dict: The job object that was patched

        """

        _api_client = self._get_k8s_api_client()
        _api = client.CustomObjectsApi(_api_client)

        if self.namespace is None:
            return _api.patch_cluster_custom_object(
                self.group.lower(),
                self.version.lower(),
                self.plural.lower(),
                self.job_name.lower(),
                custom_resource,
            )
        return _api.patch_namespaced_custom_object(
            self.group.lower(),
            self.version.lower(),
            self.namespace.lower(),
            self.plural.lower(),
            self.job_name.lower(),
            custom_resource,
        )

    def _create_custom_resource(self, custom_resource: dict):
        """Submit a custom_resource to the ACK cluster.

        Args:
            custom_resource: A dictionary object representing the custom object.
        """

        _api_client = self._get_k8s_api_client()
        _api = client.CustomObjectsApi(_api_client)

        if self.namespace is None:
            return _api.create_cluster_custom_object(
                self.group.lower(),
                self.version.lower(),
                self.plural.lower(),
                custom_resource,
            )
        return _api.create_namespaced_custom_object(
            self.group.lower(),
            self.version.lower(),
            self.namespace.lower(),
            self.plural.lower(),
            custom_resource,
        )

    def _wait_resource_consumed_by_controller(
        self,
        wait_periods,
        period_length,
    ):
        """Wait for the custom resource to be consumed by the controller.

        Args:
            wait_periods: The number of times to wait for the resource to be consumed.
            period_length: The length of time to wait between polling.
        """
        if not self._get_resource_exists():
            logging.error(
                f"Resource %s does not exist",
                (self.job_name),
            )
            return None

        for _ in range(wait_periods):
            resource = self._get_resource()

            if "status" in resource:
                return resource

            sleep(period_length)

        logging.error(
            f"Wait for resource %s to be consumed by controller timed out",
            (self.job_name),
        )
        return None

    def _get_resource_exists(self) -> bool:
        """Check if the custom resource exists.

        Returns:
            bool: True if the resource exists, False otherwise.
        """
        try:
            return self._get_resource() is not None
        except ApiException:
            return False

    def _create_resource(
        self,
        cr_spec: object,
        wait_periods=6,
        period_length=10,
    ):
        """Create a resource from the spec and wait to be consumed by
        controller.

        Args:
            cr_spec: A dictionary object representing the custom object.
            wait_periods: The number of times to wait for the resource to be created.
            period_length: The length of time to wait between polling.
        """

        resource = self._create_custom_resource(cr_spec)
        resource = self._wait_resource_consumed_by_controller(
            wait_periods, period_length
        )

        if resource is None:
            logging.error(f"Resource {self.job_name} is not created.")
            logging.error(
                f"Possible reason: ACK controller may not have been configured properly."
            )
            raise Exception(f"Resource {self.job_name} is not created.")
        else:
            logging.info(f"Created custom resource with name: {self.job_name}")

        return resource

    def _check_resource_conditions(self):
        """Check the status of the custom resource.
        * loop through all conditions
            * if recoverable and condition set to true, print out message and return true
            (let outside polling loop goes on forever and let user decide if should stop)
            * if terminal and condition set up true, print out message and return false
        * Returns None if there are no error conditions.
        """
        status_conditions = self._get_resource()["status"]["conditions"]

        for condition in status_conditions:
            condition_type = condition["type"]
            condition_status = condition["status"]
            condition_message = condition.get("message", "No error message found.")

            # If the controller has not consumed the update, any existing error will not representative of the new state.
            if self.resource_upgrade and not self.is_update_consumed_by_controller():
                continue
            if condition_type == "ACK.Terminal" and condition_status == "True":
                logging.error(json.dumps(condition, indent=2))
                logging.error(
                    "Terminating the run because resource encountered a Terminal condition. Please describe the resource for further debugging and retry with correct parameters."
                )
                return False
            if condition_type == "ACK.Recoverable" and condition_status == "True":
                # ACK requeue errors are not real errors.
                if is_ack_requeue_error(condition_message):
                    continue
                logging.error(json.dumps(condition, indent=2))
                if "ValidationException" in condition_message:
                    logging.error(
                        "Terminating the run because resource encountered a Validation Exception. Please describe the resource for further debugging and retry with correct parameters."
                    )
                    return False
                elif "InvalidParameter" in condition_message:
                    logging.error(
                        "Terminating the run because resource encountered InvalidParameters. Please describe the resource for further debugging and retry with correct parameters."
                    )
                    return False
                else:
                    logging.error(
                        "Waiting for error to be resolved . . . Please fix the error or terminate the job and retry with correct parameters if this is not a transient error"
                    )
                    return True

        return None

    def _get_resource_synced_status(self, ack_statuses: Dict):
        """ Retrieve the resource sync status
        """
        conditions = ack_statuses.get("conditions", None)  # Conditions has to be there
        if conditions == None:
            return None
        for condition in conditions:
            if condition["type"] == "ACK.ResourceSynced":
                if condition["status"] == "True":
                    return True
                else:
                    return False
        return False

    @abstractmethod
    def _after_submit_job_request(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerComponentCommonInputs,
        outputs: SageMakerComponentBaseOutputs,
    ):
        """Handles any events required after submitting a job to SageMaker.

        Args:
            job: The job returned after creation.
            request: The request submitted prior.
            inputs: A populated list of user inputs.
            outputs: An unpopulated list of component output variables.
        """
        pass

    @abstractmethod
    def _after_job_complete(
        self,
        job: object,
        request: Dict,
        inputs: SageMakerComponentCommonInputs,
        outputs: SageMakerComponentBaseOutputs,
    ):
        """Handles any events after the job has been completed.

        Args:
            job: The job object that was created.
            request: The request object used to execute the component.
            inputs: A populated list of user inputs.
            outputs: An unpopulated list of component output variables.
        """
        pass

    @abstractmethod
    def _on_job_terminated(self):
        """Handles any SIGTERM events."""
        pass

    def _delete_custom_resource(self):
        """Delete custom resource from cluster and wait for it to be removed by
        the server.

        for wait_periods * period_length seconds.
        Returns:
            response, bool:
            response is APIserver response for the operation.
            bool is true if resource was removed from the server and false otherwise
        """

        _api_client = self._get_k8s_api_client()
        _api = client.CustomObjectsApi(_api_client)

        if self.resource_upgrade:
            logging.info("Recieved termination signal, stopping component but resource update will still proceed if started. Please rerun the component with the desired configuration to revert the update.")
            return _response, True

        logging.info("Recieved termination signal, deleting custom resource %s", (self.job_name))
        _response = None
        if self.namespace is None:
            _response = _api.delete_cluster_custom_object(
                self.group.lower(),
                self.version.lower(),
                self.plural.lower(),
                self.job_name.lower(),
            )
        else:
            _response = _api.delete_namespaced_custom_object(
                self.group.lower(),
                self.version.lower(),
                self.namespace.lower(),
                self.plural.lower(),
                self.job_name.lower(),
            )

        return _response, True

    @staticmethod
    def _generate_unique_timestamped_id(
        prefix: str = "",
        size: int = 4,
        chars: str = string.ascii_uppercase + string.digits,
        max_length: int = 32,
    ) -> str:
        """Generate a pseudo-random string of characters appended to a
        timestamp.

        Format of the ID is as follows: `prefix-YYYYMMDDHHMMSS-unique`. If the
        length of the total ID exceeds `max_length`, it will be truncated from
        the beginning (prefix will be trimmed).

        Args:
            prefix: A prefix to append to the random suffix.
            size: The number of unique characters to append to the ID.
            chars: A list of characters to use in the random suffix.
            max_length: The maximum length of the generated ID.

        Returns:
            string: A pseudo-random string with included timestamp and prefix.
        """
        unique = "".join(random.choice(chars) for _ in range(size)).lower()
        return f'{prefix}{"-" if prefix else ""}{strftime("%Y%m%d%H%M%S", gmtime())}-{unique}'[
            -max_length:
        ]

    def _write_all_outputs(
        self,
        output_paths: SageMakerComponentBaseOutputs,
        outputs: SageMakerComponentBaseOutputs,
    ):
        """Writes all of the outputs specified by the component to their
        respective file paths.

        Args:
            output_paths: A populated list of output paths.
            outputs: A populated list of output values.
        """
        for output_key, output_value in outputs.__dict__.items():
            if output_value is None:
                output_value = "N/A"

            output_path = output_paths.__dict__.get(output_key)

            if not output_path:
                logging.error(f"Could not find output path for {output_key}")
                continue

            # Encode it if it's a List or Dict (not primitive)
            encoded_types = (List, Dict)
            self._write_output(
                output_path,
                output_value,
                json_encode=isinstance(output_value, encoded_types),
            )

            logging.info(f"Wrote output '{output_key}' to '{output_path}'")

    def _write_output(
        self, output_path: str, output_value: Any, json_encode: bool = False
    ):
        """Write an output value to the associated path, dumping as a JSON
        object if specified.

        Args:
            output_path: The file path of the output.
            output_value: The output value to write to the file.
            json_encode: True if the value should be encoded as a JSON object.
        """

        write_value = json.dumps(output_value) if json_encode else output_value

        Path(output_path).parent.mkdir(parents=True, exist_ok=True)
        Path(output_path).write_text(write_value)

    def _is_upgrade(self):
        """If the resource already exists the component assumes that the user wants to upgrade
        Returns:
            Bool: If the resource is being upgraded or not.
        Raises:
            Exception
        """
        try:
            resource = self._get_resource()
            if resource is None:
                return False
            logging.info("Existing resource detected. Starting Update.")
        except client.exceptions.ApiException as error:
            if error.status == 404:
                logging.info("Resource does not exist. Creating a new resource.")
                return False
            else:
                raise error
        return True

