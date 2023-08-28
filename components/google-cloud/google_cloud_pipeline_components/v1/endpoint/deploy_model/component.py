# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from typing import Dict

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import VertexEndpoint
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Input
from kfp.dsl import OutputPath


@container_component
def model_deploy(
    model: Input[VertexModel],
    gcp_resources: OutputPath(str),
    endpoint: Input[VertexEndpoint] = None,
    deployed_model_display_name: str = '',
    traffic_split: Dict[str, str] = {},
    dedicated_resources_machine_type: str = '',
    dedicated_resources_min_replica_count: int = 0,
    dedicated_resources_max_replica_count: int = 0,
    dedicated_resources_accelerator_type: str = '',
    dedicated_resources_accelerator_count: int = 0,
    automatic_resources_min_replica_count: int = 0,
    automatic_resources_max_replica_count: int = 0,
    service_account: str = '',
    disable_container_logging: bool = False,
    enable_access_logging: bool = False,
    explanation_metadata: Dict[str, str] = {},
    explanation_parameters: Dict[str, str] = {},
):
  # fmt: off
  """`Deploys <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/deployModel>`_ a Google Cloud Vertex Model to an `Endpoint <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints>`_ creating a
  `DeployedModel <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints#deployedmodel>`_ within it.

  See the `deploy Model <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/deployModel>`_ method for more information.

  Args:
      model: The model to be deployed.
      endpoint: The Endpoint to be deployed
        to.
      deployed_model_display_name: The display name of the
        DeployedModel. If not provided upon creation, the Model's display_name
        is used.
      traffic_split:
        A map from a DeployedModel's
        ID to the percentage of this Endpoint's traffic that should be
        forwarded to that DeployedModel.  If this field is non-empty, then the
        Endpoint's trafficSplit will be overwritten with it. To refer to the
        ID of the just being deployed Model, a "0" should be used, and the
        actual ID of the new DeployedModel will be filled in its place by this
        method. The traffic percentage values must add up to 100.  If this
        field is empty, then the Endpoint's trafficSplit is not updated.
      dedicated_resources_machine_type: The specification of a
        single machine used by the prediction.  This field is required if
        ``automatic_resources_min_replica_count`` is not specified.  See `more information <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints#dedicatedresources>`_.
      dedicated_resources_accelerator_type: Hardware
        accelerator type. Must also set accelerator_count if used. See `available options <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec#AcceleratorType>`_.  This field is required if
        ``dedicated_resources_machine_type`` is specified.
      dedicated_resources_accelerator_count: The number of
        accelerators to attach to a worker replica.
      dedicated_resources_min_replica_count: The minimum
        number of machine replicas this DeployedModel will be always deployed
        on. This value must be greater than or equal to 1. If traffic against
        the DeployedModel increases, it may dynamically be deployed onto more
        replicas, and as traffic decreases, some of these extra replicas may
        be freed.
      dedicated_resources_max_replica_count: The maximum
        number of replicas this deployed model may the larger value of
        min_replica_count or 1 will be used. If value provided is smaller than
        min_replica_count, it will automatically be increased to be
        min_replica_count. The maximum number of replicas this deployed model
        may be deployed on when the traffic against it increases. If requested
        value is too large, the deployment will error, but if deployment
        succeeds then the ability to scale the model to that many replicas is
        guaranteed (barring service outages). If traffic against the deployed
        model increases beyond what its replicas at maximum may handle, a
        portion of the traffic will be dropped. If this value is not provided,
        will use ``dedicated_resources_min_replica_count`` as the default value.
      automatic_resources_min_replica_count: The minimum
        number of replicas this DeployedModel will be always deployed on. If
        traffic against it increases, it may dynamically be deployed onto more
        replicas up to ``automatic_resources_max_replica_count``, and as traffic
        decreases, some of these extra replicas may be freed. If the requested
        value is too large, the deployment will error.  This field is required
        if ``dedicated_resources_machine_type`` is not specified.
      automatic_resources_max_replica_count: The maximum
        number of replicas this DeployedModel may be deployed on when the
        traffic against it increases. If the requested value is too large, the
        deployment will error, but if deployment succeeds then the ability to
        scale the model to that many replicas is guaranteed (barring service
        outages). If traffic against the DeployedModel increases beyond what
        its replicas at maximum may handle, a portion of the traffic will be
        dropped. If this value is not provided, a no upper bound for scaling
        under heavy traffic will be assume, though Vertex AI may be unable to
        scale beyond certain replica number.
      service_account: The service account that the
        DeployedModel's container runs as. Specify the email address of the
        service account. If this service account is not specified, the
        container runs as a service account that doesn't have access to the
        resource project.  Users deploying the Model must have the
        ``iam.serviceAccounts.actAs`` permission on this service account.
      disable_container_logging: For custom-trained Models
        and AutoML Tabular Models, the container of the DeployedModel
        instances will send stderr and stdout streams to Stackdriver Logging
        by default. Please note that the logs incur cost, which are subject to
        Cloud Logging pricing.  User can disable container logging by setting
        this flag to true.
      enable_access_logging: These logs are like standard
        server access logs, containing information like timestamp and latency
        for each prediction request.  Note that Stackdriver logs may incur a
        cost, especially if your project receives prediction requests at a
        high queries per second rate (QPS). Estimate your costs before
        enabling this option.
      explanation_metadata: Metadata describing the Model's
        input and output for explanation. See `more information <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#explanationmetadata>`_.
      explanation_parameters: Parameters that configure
        explaining information of the Model's predictions. See `more information <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#explanationmetadata>`_.

  Returns:
      gcp_resources: Serialized JSON of ``gcp_resources`` `proto <https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto>`_ which tracks the deploy Model's long-running operation.
  """
  # fmt: on
  return ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.endpoint.deploy_model.launcher',
      ],
      args=[
          '--type',
          'DeployModel',
          '--payload',
          ConcatPlaceholder([
              '{',
              '"endpoint": "',
              endpoint.metadata['resourceName'],
              '"',
              ', "traffic_split": ',
              traffic_split,
              ', "deployed_model": {',
              '"model": "',
              model.metadata['resourceName'],
              '"',
              ', "dedicated_resources": {',
              '"machine_spec": {',
              '"machine_type": "',
              dedicated_resources_machine_type,
              '"',
              ', "accelerator_type": "',
              dedicated_resources_accelerator_type,
              '"',
              ', "accelerator_count": ',
              dedicated_resources_accelerator_count,
              '}',
              ', "min_replica_count": ',
              dedicated_resources_min_replica_count,
              ', "max_replica_count": ',
              dedicated_resources_max_replica_count,
              '}',
              ', "automatic_resources": {',
              '"min_replica_count": ',
              automatic_resources_min_replica_count,
              ', "max_replica_count": ',
              automatic_resources_max_replica_count,
              '}',
              ', "service_account": "',
              service_account,
              '"',
              ', "disable_container_logging": ',
              disable_container_logging,
              ', "enable_access_logging": ',
              enable_access_logging,
              ', "explanation_spec": {',
              '"parameters": ',
              explanation_parameters,
              ', "metadata": ',
              explanation_metadata,
              '}',
              '}',
              '}',
          ]),
          '--project',
          '',
          '--location',
          '',
          '--gcp_resources',
          gcp_resources,
      ],
  )
