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
import re
import signal
import string
import logging
import json
from enum import Enum, auto
from types import FunctionType
import yaml
import random
from pathlib import Path
from time import sleep, strftime, gmtime
from abc import abstractmethod
from typing import Any, Type, Dict, List, NamedTuple, Optional

from .sagemaker_component_spec import SageMakerComponentSpec
from .boto3_manager import Boto3Manager
from .common_inputs import (
    SageMakerComponentBaseOutputs,
    SageMakerComponentCommonInputs,
    SpotInstanceInputs,
)

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


class DebugRulesStatus(Enum):
    COMPLETED = auto()
    ERRORED = auto()
    INPROGRESS = auto()

    @classmethod
    def from_describe(cls, response):
        has_error = False
        for debug_rule in response["DebugRuleEvaluationStatuses"]:
            if debug_rule["RuleEvaluationStatus"] == "Error":
                has_error = True
            if debug_rule["RuleEvaluationStatus"] == "InProgress":
                return DebugRulesStatus.INPROGRESS
        if has_error:
            return DebugRulesStatus.ERRORED
        else:
            return DebugRulesStatus.COMPLETED


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
        # Global try-catch in order to allow for safe abort
        try:
            self._configure_aws_clients(inputs)

            # Successful execution
            if not self._do(inputs, outputs, output_paths):
                sys.exit(1)
        except Exception as e:
            logging.exception("An error occurred while running the component")
            raise e

    def _configure_aws_clients(self, inputs: SageMakerComponentCommonInputs):
        """Configures the internal AWS clients for the component.

        Args:
            inputs: A populated list of user inputs.
        """
        self._sm_client = Boto3Manager.get_sagemaker_client(
            self._get_component_version(),
            inputs.region,
            endpoint_url=inputs.endpoint_url,
            assume_role_arn=inputs.assume_role,
        )
        self._cw_client = Boto3Manager.get_cloudwatch_client(
            inputs.region, assume_role_arn=inputs.assume_role
        )

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

        request = self._create_job_request(inputs, outputs)
        try:
            job = self._submit_job_request(request)
        except Exception as e:
            logging.exception(
                "An error occurred while attempting to submit the request"
            )
            return False

        self._after_submit_job_request(job, request, inputs, outputs)

        status: SageMakerJobStatus = SageMakerJobStatus(
            is_completed=False, raw_status="No Status"
        )
        try:
            while True:
                status = self._get_job_status()
                # Continue until complete
                if status and status.is_completed:
                    break

                sleep(self.STATUS_POLL_INTERVAL)
                logging.info(f"Job is in status: {status.raw_status}")
        except Exception as e:
            logging.exception("An error occurred while polling for job status")
            return False
        finally:
            self._print_logs_for_job()

        if status.has_error:
            logging.error(status.error_message)
            return False

        self._after_job_complete(job, request, inputs, outputs)
        self._write_all_outputs(output_paths, outputs)

        return True

    @abstractmethod
    def _get_job_status(self) -> SageMakerJobStatus:
        """Waits for the current job to complete.

        Returns:
            SageMakerJobStatus: A status object.
        """
        pass

    @abstractmethod
    def _create_job_request(
        self,
        inputs: SageMakerComponentCommonInputs,
        outputs: SageMakerComponentBaseOutputs,
    ) -> Dict:
        """Creates the boto3 request object to execute the component.

        Args:
            inputs: A populated list of user inputs.
            outputs: An unpopulated list of component output variables.

        Returns:
            dict: A dictionary object representing the request.
        """
        pass

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

    @abstractmethod
    def _print_logs_for_job(self):
        """Print the associated logs for the current job."""
        pass

    @staticmethod
    def _generate_unique_timestamped_id(
        prefix: str = "",
        size: int = 4,
        chars: str = string.ascii_uppercase + string.digits,
        max_length: int = 63,
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
        unique = "".join(random.choice(chars) for _ in range(size))
        return f'{prefix}{"-" if prefix else ""}{strftime("%Y%m%d%H%M%S", gmtime())}-{unique}'[
            -max_length:
        ]

    @staticmethod
    def _enable_spot_instance_support(
        request: Dict, inputs: SpotInstanceInputs,
    ) -> Dict:
        """Modifies a request object to add support for spot instance fields.

        Args:
            request: A request object to modify.
            inputs: A populated list of user inputs.

        Returns:
            dict: The modified dictionary
        """
        if inputs.max_run_time:
            request["StoppingCondition"]["MaxRuntimeInSeconds"] = inputs.max_run_time

        if inputs.spot_instance:
            request["EnableManagedSpotTraining"] = inputs.spot_instance
            if (
                inputs.max_wait_time
                >= request["StoppingCondition"]["MaxRuntimeInSeconds"]
            ):
                request["StoppingCondition"][
                    "MaxWaitTimeInSeconds"
                ] = inputs.max_wait_time
            else:
                logging.error(
                    "Max wait time must be greater than or equal to max run time."
                )
                raise Exception("Could not create job request.")

            if inputs.checkpoint_config and "S3Uri" in inputs.checkpoint_config:
                request["CheckpointConfig"] = inputs.checkpoint_config
            else:
                logging.error(
                    "EnableManagedSpotTraining requires checkpoint config with an S3 uri."
                )
                raise Exception("Could not create job request.")
        else:
            # Remove any artifacts that require spot instance support
            del request["StoppingCondition"]["MaxWaitTimeInSeconds"]
            del request["CheckpointConfig"]

        return request

    @staticmethod
    def _validate_hyperparameters(hyperparam_args: Dict) -> Dict:
        """Validates hyperparameters and returns the dictionary used for a
        request.

        Args:
            hyperparam_args: HyperParameters as passed in by the user.

        Returns:
            dict: A validated set of HyperParameters.
        """
        # Validate all values are strings
        for key, value in hyperparam_args.items():
            if not isinstance(value, str):
                raise Exception(
                    f"Could not parse hyperparameters. Value for {key} was not a string."
                )

        return hyperparam_args

    @staticmethod
    def _enable_tag_support(
        request: Dict, inputs: SageMakerComponentCommonInputs
    ) -> Dict:
        """Modifies a request object to add support for tag fields.

        Args:
            request: A request object to modify.
            inputs: A populated list of user inputs.

        Returns:
            dict: The modified dictionary
        """
        for key, val in inputs.tags.items():
            request["Tags"].append({"Key": key, "Value": val})

        return request

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

    @staticmethod
    def _get_component_version() -> str:
        """Get component version from the first line of License file.

        Returns:
            str: The string version as specified in the License file.
        """
        component_version = "NULL"

        # Get license file using known common directory
        license_file_path = os.path.abspath(
            os.path.join(
                SageMakerComponent._get_common_path(), "../THIRD-PARTY-LICENSES.txt"
            )
        )
        with open(license_file_path, "r") as license_file:
            version_match = re.search(
                "Amazon SageMaker Components for Kubeflow Pipelines; version (([0-9]+[.])+[0-9]+)",
                license_file.readline(),
            )
            if version_match is not None:
                component_version = version_match.group(1)

        return component_version

    @staticmethod
    def _get_common_path() -> str:
        """Gets the path of the common directory in the project.

        Returns:
            str: The `realpath` representation of the common directory.
        """
        return os.path.realpath(os.path.join(os.getcwd(), os.path.dirname(__file__)))

    @staticmethod
    def _get_request_template(template_name: str) -> Dict[str, Any]:
        """Loads and returns a template file as a Python construct.

        Args:
            template_name: The name corresponding to the template file to load.

        Returns:
            dict: A Python construct created by loading the template file.
        """
        with open(
            os.path.join(
                SageMakerComponent._get_common_path(),
                "templates",
                f"{template_name}.template.yaml",
            ),
            "r",
        ) as f:
            request = yaml.safe_load(f)
        return request

    def _print_log_header(self, header_len, title=""):
        """Prints a header section for logs.

        Args:
            header_len: The maximum length of the header line.
            title: The header title.
        """
        logging.info(f"{title:*^{header_len}}")

    def _print_cloudwatch_logs(self, log_grp: str, job_name: str):
        """Gets the CloudWatch logs for SageMaker jobs.

        Args:
            log_grp: The name of a CloudWatch log group.
            job_name: The name of the job as defined in CloudWatch.
        """

        CW_ERROR_MESSAGE = "Error in fetching CloudWatch logs for SageMaker job"

        try:
            logging.info(
                "\n******************** CloudWatch logs for {} {} ********************\n".format(
                    log_grp, job_name
                )
            )

            log_streams = self._cw_client.describe_log_streams(
                logGroupName=log_grp, logStreamNamePrefix=job_name + "/"
            )["logStreams"]

            for log_stream in log_streams:
                logging.info("\n***** {} *****\n".format(log_stream["logStreamName"]))
                response = self._cw_client.get_log_events(
                    logGroupName=log_grp, logStreamName=log_stream["logStreamName"]
                )
                for event in response["events"]:
                    logging.info(event["message"])

            logging.info(
                "\n******************** End of CloudWatch logs for {} {} ********************\n".format(
                    log_grp, job_name
                )
            )
        except Exception as e:
            logging.error(CW_ERROR_MESSAGE)
            logging.error(e)

    def _get_model_artifacts_from_job(self, job_name: str):
        """Loads training job model artifact results from a completed job.

        Args:
            job_name: The name of the completed training job.

        Returns:
            str: The S3 model artifacts of the job.
        """
        info = self._sm_client.describe_training_job(TrainingJobName=job_name)
        model_artifact_url = info["ModelArtifacts"]["S3ModelArtifacts"]
        return model_artifact_url

    def _get_image_from_job(self, job_name: str):
        """Gets the training image URL from a training job.

        Args:
            job_name: The name of a training job.

        Returns:
            str: A training image URL.
        """
        info = self._sm_client.describe_training_job(TrainingJobName=job_name)
        if "TrainingImage" in info["AlgorithmSpecification"]:
            image = info["AlgorithmSpecification"]["TrainingImage"]
        else:
            algorithm_name = info["AlgorithmSpecification"]["AlgorithmName"]
            image = self._sm_client.describe_algorithm(AlgorithmName=algorithm_name)[
                "TrainingSpecification"
            ]["TrainingImage"]

        return image
