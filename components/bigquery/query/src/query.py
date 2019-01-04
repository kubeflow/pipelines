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

import argparse
import json
import logging

from google.cloud import bigquery


def _query(query, project_id, dataset_id, table_id, output):

  client = bigquery.Client(project=project_id)

  job_config = bigquery.QueryJobConfig()
  table_ref = client.dataset(dataset_id).table(table_id)
  job_config.create_disposition = bigquery.job.CreateDisposition.CREATE_IF_NEEDED
  job_config.write_disposition = bigquery.job.WriteDisposition.WRITE_TRUNCATE
  job_config.destination = table_ref

  query_job = client.query(query, job_config=job_config)

  job_result = query_job.result()  # Wait for the query to finish

  result = {
      'destination_table': table_ref.path,
      'total_rows': job_result.total_rows,
      'total_bytes_processed': query_job.total_bytes_processed,
      'schema': [f.to_api_repr() for f in job_result.schema]
  }

  # If a GCS output location is provided, export to CSV
  if output:
    extract_job = client.extract_table(table_ref, output)
    extract_job.result()  # Wait for export to finish

  return result


def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--output',
      type=str,
      required=False,
      help='GCS URL where results will be saved as a CSV.')
  parser.add_argument(
      '--query',
      type=str,
      required=True,
      help='The SQL query to be run in BigQuery')
  parser.add_argument(
      '--dataset_id',
      type=str,
      required=True,
      help='Dataset of the destination table.')
  parser.add_argument(
      '--table_id',
      type=str,
      required=True,
      help='Name of the destination table.')
  parser.add_argument(
      '--project',
      type=str,
      required=True,
      help='The GCP project to run the query.')

  args = parser.parse_args()
  return args


def run_query(query, project_id, dataset_id, table_id, output):
  results = _query(query, project_id, dataset_id, table_id, output)
  results['output'] = output
  with open('/output.json', 'w+') as f:
    json.dump(results, f)


def main():
  logging.getLogger().setLevel(logging.INFO)
  args = parse_arguments()

  run_query(args.query, args.project, args.dataset_id, args.table_id,
            args.output)


if __name__ == '__main__':
  main()
