"""SageMaker component for deploy."""
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

from dataclasses import dataclass
import logging
from typing import Dict

from botocore.exceptions import ClientError

from deploy.src.sagemaker_deploy_spec import (
    SageMakerDeploySpec,
    SageMakerDeployInputs,
    SageMakerDeployOutputs,
)
from common.sagemaker_component import (
    SageMakerComponent,
    ComponentMetadata,
    SageMakerJobStatus,
)


@dataclass(frozen=True)
class EndpointRequests:
    """Holds the request types for creating each of the deploy steps."""

    config_request: Dict
    endpoint_request: Dict


@dataclass(frozen=True)
class EndpointResponses:
    """Holds the responses for creating each of the deploy step."""

    config_response: Dict
    endpoint_response: Dict


@ComponentMetadata(
    name="SageMaker - Deploy Model",
    description="Deploy Machine Learning Model Endpoint in SageMaker",
    spec=SageMakerDeploySpec,
)
class SageMakerDeployComponent(SageMakerComponent):
    """SageMaker component for deploy."""

    def Do(self, spec: SageMakerDeploySpec):
        # Manually invoke AWS client configuration so we can use it before
        # starting the reconciliation loop
        self._configure_aws_clients(spec.inputs)

        name_suffix = SageMakerComponent._generate_unique_timestamped_id()

        self._endpoint_name = (
            spec.inputs.endpoint_name
            if spec.inputs.endpoint_name
            else f"Endpoint{name_suffix}"
        )

        self._should_update_existing = (
            spec.inputs.update_endpoint
            and self._endpoint_name_exists(spec.inputs.endpoint_name)
        )

        # Fetch existing config to delete after creation
        if self._should_update_existing:
            self._existing_endpoint_config_name = self._get_endpoint_config(
                spec.inputs.endpoint_name
            )

        self._endpoint_config_name = (
            spec.inputs.endpoint_config_name
            # Only use the predefined name if we are not updating, otherwise could conflict
            if (spec.inputs.endpoint_config_name and not self._should_update_existing)
            else f"EndpointConfig{name_suffix}"
        )

        super().Do(spec.inputs, spec.outputs, spec.output_paths)

    def _get_job_status(self) -> SageMakerJobStatus:
        # Wait for endpoint creation to complete
        response = self._sm_client.describe_endpoint(EndpointName=self._endpoint_name)
        status = response["EndpointStatus"]

        if status == "InService":
            return SageMakerJobStatus(
                is_completed=True, has_error=False, raw_status=status
            )
        if status == "Failed":
            message = response["FailureReason"]
            return SageMakerJobStatus(
                is_completed=True,
                has_error=True,
                error_message=message,
                raw_status=status,
            )

        return SageMakerJobStatus(is_completed=False, raw_status=status)

    def _after_job_complete(
        self,
        job: object,
        request: EndpointRequests,
        inputs: SageMakerDeployInputs,
        outputs: SageMakerDeployOutputs,
    ):
        outputs.endpoint_name = self._endpoint_name

    def _create_endpoint_config_request(
        self, inputs: SageMakerDeployInputs, outputs: SageMakerDeployOutputs
    ):
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_endpoint_config
        request = self._get_request_template("endpoint_config")

        if not inputs.model_name_1:
            logging.error("Must specify at least one model (model name) to host.")
            raise Exception("Could not create endpoint config.")

        request["EndpointConfigName"] = self._endpoint_config_name

        if inputs.resource_encryption_key:
            request["KmsKeyId"] = inputs.resource_encryption_key
        else:
            request.pop("KmsKeyId")

        for i in range(len(request["ProductionVariants"]), 0, -1):
            if inputs.__dict__["model_name_" + str(i)]:
                request["ProductionVariants"][i - 1]["ModelName"] = inputs.__dict__[
                    "model_name_" + str(i)
                ]
                if inputs.__dict__["variant_name_" + str(i)]:
                    request["ProductionVariants"][i - 1][
                        "VariantName"
                    ] = inputs.__dict__["variant_name_" + str(i)]
                if inputs.__dict__["initial_instance_count_" + str(i)]:
                    request["ProductionVariants"][i - 1][
                        "InitialInstanceCount"
                    ] = inputs.__dict__["initial_instance_count_" + str(i)]
                if inputs.__dict__["instance_type_" + str(i)]:
                    request["ProductionVariants"][i - 1][
                        "InstanceType"
                    ] = inputs.__dict__["instance_type_" + str(i)]
                if inputs.__dict__["initial_variant_weight_" + str(i)]:
                    request["ProductionVariants"][i - 1][
                        "InitialVariantWeight"
                    ] = inputs.__dict__["initial_variant_weight_" + str(i)]
                if inputs.__dict__["accelerator_type_" + str(i)]:
                    request["ProductionVariants"][i - 1][
                        "AcceleratorType"
                    ] = inputs.__dict__["accelerator_type_" + str(i)]
                else:
                    request["ProductionVariants"][i - 1].pop("AcceleratorType")
            else:
                request["ProductionVariants"].pop(i - 1)

        ### Update tags
        for key, val in inputs.endpoint_config_tags.items():
            request["Tags"].append({"Key": key, "Value": val})

        return request

    def _create_endpoint_request(
        self, inputs: SageMakerDeployInputs, outputs: SageMakerDeployOutputs
    ):
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_endpoint
        request = {}

        request["EndpointName"] = self._endpoint_name
        request["EndpointConfigName"] = self._endpoint_config_name

        self._enable_tag_support(request, inputs)

        return request

    def _create_update_endpoint_request(self):
        """Creates an UpdateEndpoint request object.

        Returns:
            An UpdateEndpoint request object.
        """
        ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.update_endpoint
        request = {}

        request["EndpointName"] = self._endpoint_name
        request["EndpointConfigName"] = self._endpoint_config_name

        return request

    def _create_job_request(
        self, inputs: SageMakerDeployInputs, outputs: SageMakerDeployOutputs,
    ) -> EndpointRequests:
        # The endpoint request represents an UpdateEndpoint request if we are updating
        endpoint_request = (
            self._create_update_endpoint_request()
            if self._should_update_existing
            else self._create_endpoint_request(inputs, outputs)
        )

        return EndpointRequests(
            config_request=self._create_endpoint_config_request(inputs, outputs),
            endpoint_request=endpoint_request,
        )

    def _submit_job_request(self, request: EndpointRequests) -> EndpointResponses:
        config_response = self._sm_client.create_endpoint_config(
            **request.config_request
        )

        if self._should_update_existing:
            endpoint_response = self._sm_client.update_endpoint(
                **request.endpoint_request
            )

            # Also delete the existing endpoint config
            if self._delete_endpoint_config(self._existing_endpoint_config_name):
                logging.info(
                    f"Deleted old endpoint config: {self._existing_endpoint_config_name}"
                )
            else:
                logging.info(
                    f"Unable to delete old endpoint config: {self._existing_endpoint_config_name}"
                )
        else:
            endpoint_response = self._sm_client.create_endpoint(
                **request.endpoint_request
            )

        return EndpointResponses(
            config_response=config_response, endpoint_response=endpoint_response
        )

    def _after_submit_job_request(
        self,
        job: EndpointResponses,
        request: EndpointRequests,
        inputs: SageMakerDeployInputs,
        outputs: SageMakerDeployOutputs,
    ):
        region = inputs.region
        logging.info(
            "Endpoint configuration in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/endpointConfig/{}".format(
                region, region, request.config_request["EndpointConfigName"]
            )
        )
        logging.info(f"Endpoint Config Arn: {job.config_response['EndpointConfigArn']}")
        logging.info(f"Created endpoint with name: {self._endpoint_name}")
        logging.info(
            "Endpoint in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/endpoints/{}".format(
                region, region, self._endpoint_name
            )
        )
        logging.info(
            "CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/Endpoints/{};streamFilter=typeLogStreamPrefix".format(
                region, region, self._endpoint_name
            )
        )

    def _endpoint_name_exists(self, endpoint_name: str):
        """Determine whether an endpoint already exists by a given name.

        Args:
            endpoint_name: The name of the endpoint to check.

        Returns:
            True if the endpoint already exists, False otherwise.
        """
        try:
            endpoint_name = self._sm_client.describe_endpoint(
                EndpointName=endpoint_name
            )["EndpointName"]
            logging.info("Endpoint exists: " + endpoint_name)
            return True
        except ClientError as e:
            logging.debug("Endpoint does not exist")
        return False

    def _endpoint_config_name_exists(self, endpoint_config_name: str):
        """Determine whether an endpoint config already exists by a given name.

        Args:
            endpoint_config_name: The name of the endpoint config to check.

        Returns:
            True if the endpoint config already exists, False otherwise.
        """
        try:
            config_name = self._sm_client.describe_endpoint_config(
                EndpointConfigName=endpoint_config_name
            )["EndpointConfigName"]
            logging.info("Endpoint Config exists: " + config_name)
            return True
        except ClientError as e:
            logging.info("Endpoint Config does not exist:" + endpoint_config_name)
        return False

    def _get_endpoint_config(self, endpoint_name: str):
        """Gets the name of the current config for a given endpoint.

        Args:
            endpoint_name: The name of the endpoint whose config to reference.

        Returns:
            The name of an endpoint configuration currently assigned to the given endpoint.
        """
        endpoint_config_name = None
        try:
            endpoint_config_name = self._sm_client.describe_endpoint(
                EndpointName=endpoint_name
            )["EndpointConfigName"]
            logging.info("Current Endpoint Config Name: " + endpoint_config_name)
        except ClientError as e:
            logging.info("Endpoint Config does not exist")
            ## This is not an error, end point may not exist
        return endpoint_config_name

    def _delete_endpoint_config(self, endpoint_config_name: str):
        """Deletes an endpoint config.

        Args:
            endpoint_config_name: The name of the endpoint config to delete.

        Returns:
            True if the endpoint was deleted, False otherwise.
        """
        try:
            self._sm_client.delete_endpoint_config(
                EndpointConfigName=endpoint_config_name
            )
            return True
        except ClientError as e:
            logging.info(
                "Endpoint config may not exist to be deleted: "
                + e.response["Error"]["Message"]
            )
        return False


if __name__ == "__main__":
    import sys

    spec = SageMakerDeploySpec(sys.argv[1:])

    component = SageMakerDeployComponent()
    component.Do(spec)
