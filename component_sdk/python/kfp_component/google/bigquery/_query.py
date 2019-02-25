# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json

from google.cloud import bigquery
from google.api_core import exceptions

from kfp_component.core import KfpExecutionContext, display
from .. import common as gcp_common

def query(query, project_id, dataset_id, table_id=None, 
    output_gcs_path=None, job_config=None):
    client = bigquery.Client(project=project_id)
    if not job_config:
        job_config = bigquery.QueryJobConfig()
    job_config.create_disposition = bigquery.job.CreateDisposition.CREATE_IF_NEEDED
    job_config.write_disposition = bigquery.job.WriteDisposition.WRITE_TRUNCATE
    job_id = None
    def cancel():
        if job_id:
            client.cancel_job(job_id)
    with KfpExecutionContext(on_cancel=cancel) as ctx:
        job_id = 'query_' + ctx.context_id()
        if not table_id:
            table_id = 'table_' + ctx.context_id()
        table_ref = client.dataset(dataset_id).table(table_id)
        query_job = _get_job(client, job_id)
        if not query_job:
            job_config.destination = table_ref
            query_job = client.query(query, job_config, job_id=job_id)
        _display_job_link(project_id, job_id)
        query_job.result()
        if output_gcs_path:
            job_id = 'extract_' + ctx.context_id()
            extract_job = _get_job(client, job_id)
            if not extract_job:
                extract_job = client.extract_table(table_ref, output_gcs_path)
            extract_job.result()  # Wait for export to finish
        return query_job.to_api_repr()

def _get_job(client, job_id):
    try:
        return client.get_job(job_id)
    except exceptions.NotFound:
        return None

def _display_job_link(project_id, job_id):
    display.display(display.Link(
        href= 'https://console.cloud.google.com/bigquery?project={}'
            '&j={}&page=queryresults'.format(project_id, job_id),
        text='Query Details'
    ))

def _dump_outputs(job):
    gcp_common.dump_file('/tmp/outputs/bigquery-job.json', 
        json.dumps(job.to_api_repr()))