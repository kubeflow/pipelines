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
import logging
import os

from google.cloud import bigquery
from google.cloud.bigquery.job import ExtractJobConfig, DestinationFormat
from google.api_core import exceptions

from kfp_component.core import KfpExecutionContext, display
from .. import common as gcp_common

# TODO(hongyes): make this path configurable as a environment variable
KFP_OUTPUT_PATH = '/tmp/kfp/output/'


def query(query, project_id, dataset_id=None, table_id=None, 
    output_gcs_path=None, dataset_location='US', job_config=None,
    output_path=None, output_filename=None, output_destination_format="CSV",
    job_object_output_path='/tmp/kfp/output/bigquery/query-job.json',
    output_gcs_path_output_path='/tmp/kfp/output/bigquery/query-output-path.txt',
    output_dataset_id_output_path='/tmp/kfp/output/bigquery/query-dataset-id.txt',
    output_table_id_output_path='/tmp/kfp/output/bigquery/query-table-id.txt',
):
    """Submit a query to Bigquery service and dump outputs to Bigquery table or 
    a GCS blob.
    
    Args:
        query (str): The query used by Bigquery service to fetch the results.
        project_id (str): The project to execute the query job.
        dataset_id (str): The ID of the persistent dataset to keep the results
            of the query. If the dataset does not exist, the operation will 
            create a new one.
        table_id (str): The ID of the table to keep the results of the query. If
            absent, the operation will generate a random id for the table.
        output_gcs_path (str): The GCS blob path to dump the query results to.
        dataset_location (str): The location to create the dataset. Defaults to `US`.
        job_config (dict): The full config spec for the query job.
        output_path (str): The path to where query result will be stored
        output_filename (str): The name of the file where the results will be stored
        output_destination_format (str): The name of the output destination format.
            Default is CSV, and you can also choose NEWLINE_DELIMITED_JSON and AVRO.
    Returns:
        The API representation of the completed query job.
    """
    client = bigquery.Client(project=project_id, location=dataset_location)
    if not job_config:
        job_config = bigquery.QueryJobConfig()
        job_config.create_disposition = bigquery.job.CreateDisposition.CREATE_IF_NEEDED
        job_config.write_disposition = bigquery.job.WriteDisposition.WRITE_TRUNCATE
    else:
        job_config = bigquery.QueryJobConfig.from_api_repr(job_config)
    job_id = None
    def cancel():
        if job_id:
            client.cancel_job(job_id)
    with KfpExecutionContext(on_cancel=cancel) as ctx:
        job_id = 'query_' + ctx.context_id()
        query_job = _get_job(client, job_id)
        table_ref = None
        if not query_job:
            dataset_ref = _prepare_dataset_ref(client, dataset_id, output_gcs_path, 
                dataset_location)
            if dataset_ref:
                if not table_id:
                    table_id = job_id
                table_ref = dataset_ref.table(table_id)
                job_config.destination = table_ref
                gcp_common.dump_file(output_dataset_id_output_path, table_ref.dataset_id)
                gcp_common.dump_file(output_table_id_output_path, table_ref.table_id)
            query_job = client.query(query, job_config, job_id=job_id)
        _display_job_link(project_id, job_id)
        if output_path != None: #Write to local file
            result = query_job.result()
            if not os.path.exists(output_path):
                os.makedirs(output_path)
            df = result.to_dataframe()
            df.to_csv(os.path.join(output_path, output_filename))
        else:
            query_job.result() 
            if output_gcs_path:
                job_id = 'extract_' + ctx.context_id()
                extract_job = _get_job(client, job_id)
                logging.info('Extracting data from table {} to {}.'.format(str(table_ref), output_gcs_path))
                if not extract_job:
                    job_config = ExtractJobConfig(destination_format=output_destination_format)
                    extract_job = client.extract_table(table_ref, output_gcs_path, job_config=job_config)
                extract_job.result()  # Wait for export to finish
            # TODO: Replace '-' with empty string when most users upgrade to Argo version which has the fix: https://github.com/argoproj/argo/pull/1653
            gcp_common.dump_file(output_gcs_path_output_path, output_gcs_path or '-')

        gcp_common.dump_file(job_object_output_path, json.dumps(query_job.to_api_repr()))
        return query_job.to_api_repr()

def _get_job(client, job_id):
    try:
        return client.get_job(job_id)
    except exceptions.NotFound:
        return None

def _prepare_dataset_ref(client, dataset_id, output_gcs_path, dataset_location):
    if not output_gcs_path and not dataset_id:
        return None
    if not dataset_id:
        dataset_id = 'kfp_tmp_dataset'
    dataset_ref = client.dataset(dataset_id)
    dataset = _get_dataset(client, dataset_ref)
    if not dataset:
        logging.info('Creating dataset {}'.format(dataset_id))
        dataset = _create_dataset(client, dataset_ref, dataset_location)
    return dataset_ref

def _get_dataset(client, dataset_ref):
    try:
        return client.get_dataset(dataset_ref)
    except exceptions.NotFound:
        return None

def _create_dataset(client, dataset_ref, location):
    dataset = bigquery.Dataset(dataset_ref)
    dataset.location = location
    return client.create_dataset(dataset)

def _display_job_link(project_id, job_id):
    display.display(display.Link(
        href= 'https://console.cloud.google.com/bigquery?project={}'
            '&j={}&page=queryresults'.format(project_id, job_id),
        text='Query Details'
    ))

def _dump_outputs(job, output_path, table_ref):
    gcp_common.dump_file(KFP_OUTPUT_PATH + 'bigquery/query-job.json', 
        json.dumps(job.to_api_repr()))
    if not output_path:
        output_path = '-'  # Replace with empty string when we upgrade to Argo version which has the fix: https://github.com/argoproj/argo/pull/1653
    gcp_common.dump_file(KFP_OUTPUT_PATH + 'bigquery/query-output-path.txt', 
        output_path)
    (dataset_id, table_id) = (table_ref.dataset_id, table_ref.table_id) if table_ref else ('-', '-')
    gcp_common.dump_file(KFP_OUTPUT_PATH + 'bigquery/query-dataset-id.txt', 
        dataset_id)
    gcp_common.dump_file(KFP_OUTPUT_PATH + 'bigquery/query-table-id.txt', 
        table_id)
