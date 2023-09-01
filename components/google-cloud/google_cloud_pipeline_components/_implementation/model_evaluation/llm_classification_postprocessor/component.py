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
"""LLM Classification Postprocessor component."""

from typing import List, NamedTuple, Optional

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from google_cloud_pipeline_components._implementation.model_evaluation import version
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath


@dsl.component
def add_json_escape_class_labels(class_labels: list) -> str:
  import json

  json_escaped_class_labels = json.dumps(class_labels).replace('"', '\\"')
  return json_escaped_class_labels


@dsl.container_component
def llm_classification_predictions_postprocessor_internal(
    gcp_resources: OutputPath(str),
    postprocessed_predictions_gcs_source: Output[Artifact],
    postprocessed_class_labels: OutputPath(list),
    project: str,
    location: str,
    batch_prediction_results: Input[Artifact],
    class_labels: str,
    display_name: str = 'llm-classification-predictions-postprocessor',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    reserved_ip_ranges: Optional[List[str]] = None,
    encryption_spec_key_name: str = '',
):
  """Postprocesses LLM predictions for evaluating classification task.

  For each output string, find the first appearance of a class label in the
  list of classes, and output the index of this class in a one-hot encoding
  format for evaluation. If the output string does not contain any class labels
  from the list, label it as “UNKNOWN”.

  Constraints
    1. In rare cases, if the model outputs verbose answers like "The topic of
    the text is not business, but is health". In this case, the first answer in
    the list the model outputs isn't what the model actually chose, and the
    postprocessor output would be incorrect.
    2. Cannot handle cases where class names are substrings of each other. For
    example, "toxic, nontoxic".

  Args:
    project: The GCP project that runs the pipeline component.
    location: The GCP region that runs the pipeline component.
    batch_prediction_results: An Artifact pointing toward a GCS directory with
      prediction files to be used for this component.
    class_labels: String that is JSON array escapedclass names for the
      target_field, in the same order they appear in the batch predictions input
      file.
    display_name: The name of the custom job.
    machine_type: The machine type of this custom job. If not set, defaulted to
      `e2-highmem-16`. More details:
        https://cloud.google.com/compute/docs/machine-resource
    service_account: Sets the default service account for workload run-as
      account. The service account running the pipeline
      (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
      submitting jobs must have act-as permission on this run-as account.
    network: The full name of the Compute Engine network to which the job should
      be peered. For example, `projects/12345/global/networks/myVPC`. Format is
      of the form `projects/{project}/global/networks/{network}`. Where
      `{project}` is a project number, as in `12345`, and `{network}` is a
      network name, as in `myVPC`. To specify this field, you must have already
      configured VPC Network Peering for Vertex AI
      (https://cloud.google.com/vertex-ai/docs/general/vpc-peering). If left
      unspecified, the job is not peered with any network.
    reserved_ip_ranges: A list of names for the reserved ip ranges under the VPC
      network that can be used for this job. If set, we will deploy the job
      within the provided ip ranges. Otherwise, the job will be deployed to any
      ip ranges under the provided VPC network.
    encryption_spec_key_name:  Customer-managed encryption key options. If set,
      resources created by this Custom Job will be encrypted with the provided
      encryption key. Has the form:
      `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`.
      The key needs to be in the same region as where the compute resource is
      created.

  Returns:
    postprocessed_predictions_gcs_source: A string URI pointing toward a GCS
      directory with postprocessed prediction files to be used for Evaluation
      component.
    postprocessed_class_labels: The list of class names for the target_field
      with an additional field named "UNKNOWN", in the same order they appear in
      the batch predictions input file.
    gcp_resources: Serialized gcp_resources proto tracking the custom job.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_custom_job_payload(
          display_name=display_name,
          machine_type=machine_type,
          image_uri=version.LLM_EVAL_IMAGE_TAG,
          args=[
              '--postprocessor',
              'true',
              '--batch_prediction_results',
              batch_prediction_results.path,
              '--postprocessed_predictions_gcs_source',
              postprocessed_predictions_gcs_source.path,
              '--class_labels',
              class_labels,
              '--postprocessed_class_labels',
              postprocessed_class_labels,
              '--executor_input',
              '{{$.json_escape[1]}}',
          ],
          service_account=service_account,
          network=network,
          reserved_ip_ranges=reserved_ip_ranges,
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )


@dsl.pipeline(name='ModelEvaluationLLMClassificationPredictionsPostprocessorOp')
def llm_classification_predictions_postprocessor_graph_component(
    project: str,
    location: str,
    batch_prediction_results: Input[Artifact],
    class_labels: List[str],
    display_name: str = 'llm-classification-predictions-postprocessor',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    reserved_ip_ranges: Optional[List[str]] = None,
    encryption_spec_key_name: str = '',
) -> NamedTuple(
    'outputs',
    postprocessed_predictions_gcs_source=Artifact,
    postprocessed_class_labels=List[str],
):
  """Graph component to postprocess LLM predictions for evaluating classification task.

  For each output string, find the first appearance of a class label in the
  list of classes, and output the index of this class in a one-hot encoding
  format for evaluation. If the output string does not contain any class labels
  from the list, label it as “UNKNOWN”.

  Constraints
    1. In rare cases, if the model outputs verbose answers like "The topic of
    the text is not business, but is health". In this case, the first answer in
    the list the model outputs isn't what the model actually chose, and the
    postprocessor output would be incorrect.
    2. Cannot handle cases where class names are substrings of each other. For
    example, "toxic, nontoxic".

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    batch_prediction_results: An Artifact pointing toward a GCS directory with
      prediction files to be used for this component.
    class_labels: The JSON array of class names for the target_field, in the
      same order they appear in the batch predictions input file.
    display_name: The name of the custom job.
    machine_type: The machine type of this custom job. If not set, defaulted to
      `e2-highmem-16`. More details:
        https://cloud.google.com/compute/docs/machine-resource
    service_account: Sets the default service account for workload run-as
      account. The service account running the pipeline
      (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
      submitting jobs must have act-as permission on this run-as account.
    network: The full name of the Compute Engine network to which the job should
      be peered. For example, `projects/12345/global/networks/myVPC`. Format is
      of the form `projects/{project}/global/networks/{network}`. Where
      `{project}` is a project number, as in `12345`, and `{network}` is a
      network name, as in `myVPC`. To specify this field, you must have already
      configured VPC Network Peering for Vertex AI
      (https://cloud.google.com/vertex-ai/docs/general/vpc-peering). If left
      unspecified, the job is not peered with any network.
    reserved_ip_ranges: A list of names for the reserved ip ranges under the VPC
      network that can be used for this job. If set, we will deploy the job
      within the provided ip ranges. Otherwise, the job will be deployed to any
      ip ranges under the provided VPC network.
    encryption_spec_key_name:  Customer-managed encryption key options. If set,
      resources created by this Custom Job will be encrypted with the provided
      encryption key. Has the form:
      `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`.
      The key needs to be in the same region as where the compute resource is
      created.

  Returns:
    NamedTuple:
      postprocessed_predictions_gcs_source: A string URI pointing toward a GCS
        directory with postprocessed prediction files to be used for Evaluation
        component.
      postprocessed_class_labels: The list of class names for the target_field
        with an additional field named "UNKNOWN", in the same order they appear
        in the batch predictions input file.
  """
  outputs = NamedTuple(
      'outputs',
      postprocessed_predictions_gcs_source=Artifact,
      postprocessed_class_labels=List[str],
  )

  postprocessor_task = llm_classification_predictions_postprocessor_internal(
      project=project,
      batch_prediction_results=batch_prediction_results,
      class_labels=add_json_escape_class_labels(
          class_labels=class_labels
      ).output,
      location=location,
      display_name=display_name,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      reserved_ip_ranges=reserved_ip_ranges,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  return outputs(
      postprocessed_predictions_gcs_source=postprocessor_task.outputs[
          'postprocessed_predictions_gcs_source'
      ],
      postprocessed_class_labels=postprocessor_task.outputs[
          'postprocessed_class_labels'
      ],
  )
