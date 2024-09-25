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

import argparse
import json
import logging
import time
import re

import boto3


def get_client(region=None):
    """Builds a client to the AWS Athena API."""
    client = boto3.client("athena", region_name=region)
    return client


def query(client, query, database, output, workgroup=None):
    """Executes an AWS Athena query."""
    params = dict(
        QueryString=query,
        QueryExecutionContext={"Database": database},
        ResultConfiguration={"OutputLocation": output,},
    )
    if workgroup:
        params.update(dict(WorkGroup=workgroup))

    response = client.start_query_execution(**params)

    execution_id = response["QueryExecutionId"]
    logging.info("Execution ID: %s", execution_id)

    # Athena query is aync call, we need to fetch results and wait for execution
    state = "RUNNING"
    max_execution = (
        5  # TODO: this should be an optional parameter from users. or use timeout
    )

    while max_execution > 0 and state in ["RUNNING"]:
        max_execution = max_execution - 1
        response = client.get_query_execution(QueryExecutionId=execution_id)

        if (
            "QueryExecution" in response
            and "Status" in response["QueryExecution"]
            and "State" in response["QueryExecution"]["Status"]
        ):
            state = response["QueryExecution"]["Status"]["State"]
            if state == "FAILED":
                raise Exception("Athena Query Failed")
            elif state == "SUCCEEDED":
                s3_path = response["QueryExecution"]["ResultConfiguration"][
                    "OutputLocation"
                ]
                # could be multiple files?
                filename = re.findall(r".*\/(.*)", s3_path)[0]
                logging.info("S3 output file name %s", filename)
                break
        time.sleep(5)

    # TODO:(@Jeffwan) Add more details.
    result = {
        "total_bytes_processed": response["QueryExecution"]["Statistics"][
            "DataScannedInBytes"
        ],
        "filename": filename,
    }

    return result


def main():
    logging.getLogger().setLevel(logging.INFO)
    parser = argparse.ArgumentParser()
    parser.add_argument("--region", type=str, help="Athena region.")
    parser.add_argument(
        "--database", type=str, required=True, help="The name of the database."
    )
    parser.add_argument(
        "--query",
        type=str,
        required=True,
        help="The SQL query statements to be executed in Athena.",
    )
    parser.add_argument(
        "--output",
        type=str,
        required=False,
        help="The location in Amazon S3 where your query results are stored, such as s3://path/to/query/bucket/",
    )
    parser.add_argument(
        "--workgroup",
        type=str,
        required=False,
        help="Optional argument to provide Athena workgroup",
    )

    args = parser.parse_args()

    client = get_client(args.region)
    results = query(client, args.query, args.database, args.output, args.workgroup)

    results["output"] = args.output
    logging.info("Athena results: %s", results)
    with open("/output.txt", "w+") as f:
        json.dump(results, f)


if __name__ == "__main__":
    main()
