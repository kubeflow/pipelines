# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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

import json
from . import lro_remote_runner
from .utils import artifact_util
from .utils import json_util
from typing import Dict, Optional, Tuple


from google.cloud import bigquery
from google_cloud_pipeline_components.experimental.proto.gcp_resources_pb2 import (
    GcpResources,
)
from google.protobuf import json_format


def bq_query(
    project: str,
    location: str,
    query: str,
    dry_run: Optional[bool] = None,
    use_query_cache: Optional[bool] = None,
    use_legacy_sql: Optional[bool] = None,
    labels: Optional[Dict[str, str]] = None,
    maximum_bytes_billed: Optional[int] = None,
    destination_encryption_spec_key_name: Optional[str] = None,
    destination_table_id: Optional[str] = None,
) -> Tuple[GcpResources, str]:
    """Query

    https://cloud.google.com/bigquery/docs/writing-results?hl=en
    """

    client = bigquery.Client(project=project, location=location)

    job_config = bigquery.QueryJobConfig()
    job_config.labels = {"kfp_runner": "bqml"}.update(labels or {})

    if destination_encryption_spec_key_name:
        encryption_config = bigquery.EncryptionConfiguration(
            destination_encryption_spec_key_name=destination_encryption_spec_key_name
        )
        job_config.destination_encryption_configuration = encryption_config

    if dry_run:
        job_config.dry_run = dry_run

    if maximum_bytes_billed:
        job_config.maximum_bytes_billed = maximum_bytes_billed

    if use_query_cache:
        job_config.use_query_cache = use_query_cache

    if use_legacy_sql:
        job_config.use_legacy_sql = use_legacy_sql

    if destination_table_id:
        job_config.destination = destination_table_id

    query_job = client.query(query, job_config=job_config)  # API request
    query_job.result()  # Waits for query to finish

    # Extract destination table
    destination = query_job.destination
    output_destination_table_id = (
        f"{destination.project}.{destination.dataset_id}.{destination.table_id}"
    )

    # Instantiate GCPResources Proto
    query_job_resources = GcpResources()
    query_job_resource = query_job_resources.resources.add()

    # Write the job proto to output
    query_job_resource.resource_type = "BigQueryJob"
    query_job_resource.resource_uri = query_job.self_link

    # query_job_resources_serialized = json_format.MessageToJson(query_job_resources)

    # from collections import namedtuple

    # output = namedtuple("Outputs", ["gcp_resources", "destination_table_id"])
    return (query_job_resources, output_destination_table_id)


def create_bigquery_query(
    type,
    project,
    location,
    payload,
    gcp_resources,
    executor_input,
):
    """
    Create endpoint and poll the LongRunningOperator till it reaches a final state.
    """
    # api_endpoint = location + '-aiplatform.googleapis.com'
    # vertex_uri_prefix = f"https://{api_endpoint}/v1/"
    # create_endpoint_url = f"{vertex_uri_prefix}projects/{project}/locations/{location}/endpoints"
    # endpoint_spec = json.loads(payload, strict=False)
    # # TODO(IronPan) temporarily remove the empty fields from the spec
    # create_endpoint_request = json_util.recursive_remove_empty(endpoint_spec)

    # remote_runner = lro_remote_runner.LroRemoteRunner(location)
    # create_endpoint_lro = remote_runner.create_lro(
    #     create_endpoint_url, json.dumps(create_endpoint_request), gcp_resources)
    # create_endpoint_lro = remote_runner.poll_lro(lro=create_endpoint_lro)
    query_spec = json.loads(payload, strict=False)

    # Run BQ query serially
    query_job_resources, output_destination_table_id = bq_query(
        project=project,
        location=location,
        query=query_spec.query,
        dry_run=query_spec.dry_run,
        use_query_cache=query_spec.use_query_cache,
        use_legacy_sql=query_spec.use_legacy_sql,
        labels=query_spec.labels,
        maximum_bytes_billed=query_spec.maximum_bytes_billed,
        destination_encryption_spec_key_name=query_spec.destination_encryption_spec_key_name,
        destination_table_id=query_spec.destination_table_id
    )

    # endpoint_resource_name = create_endpoint_lro["response"]["name"]

    artifact_util.update_output_artifact(
        executor_input,
        "bigqueryjob",
        query_job_resources.resources.get[0].resource_uri,
        {artifact_util.ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: "TODO"},
    )
