# Copyright 2022 The Kubeflow Authors
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

import datetime
import json
from typing import List

from kfp import client
import requests
from tqdm import tqdm

today = datetime.datetime.now(tz=datetime.timezone.utc)
DATE_THRESHOLD = today - datetime.timedelta(days=14)


def main() -> None:
    endpoint = get_endpoint()
    print(f'Cleaning up old runs from KFP instance: {endpoint}\n')

    kfp_client = client.Client(host=endpoint)
    deleted_run_count = delete_old_runs(
        kfp_client=kfp_client, date_threshold=DATE_THRESHOLD)
    print(
        f'Successfully deleted {deleted_run_count} runs created before {DATE_THRESHOLD}.\n'
    )


def get_endpoint() -> str:
    response = requests.get(
        'https://raw.githubusercontent.com/kubeflow/testing/master/test-infra/kfp/endpoint'
    )
    response.raise_for_status()
    clean_partial_url = response.text.strip()
    return f'https://{clean_partial_url}'


def accumulate_ids_to_delete(kfp_client: client.Client,
                             date_threshold: datetime.datetime) -> List[str]:
    ids = []
    open_api_formatted_timestamp = date_threshold.strftime('%Y-%m-%dT%H:%M:%SZ')

    filter_operation = json.dumps({
        'predicates': [{
            'op': client.client._FILTER_OPERATIONS['LESS_THAN'],
            'key': 'created_at',
            'timestampValue': open_api_formatted_timestamp
        }]
    })
    initial_response = kfp_client.list_runs(
        filter=filter_operation,
        page_size=100,
    )
    ids = [run.id for run in initial_response.runs
          ] if initial_response.runs else []
    next_page_token = initial_response.next_page_token

    while True:
        response = kfp_client.list_runs(page_token=next_page_token)
        if not response.runs:
            break
        ids.extend([run.id for run in response.runs])
        next_page_token = response.next_page_token
        if not next_page_token:
            break
    return ids


def delete_old_runs(kfp_client: client.Client,
                    date_threshold: datetime.datetime) -> int:
    ids_to_delete = accumulate_ids_to_delete(
        kfp_client=kfp_client, date_threshold=date_threshold)

    for run_id in tqdm(ids_to_delete):
        kfp_client.delete_run(run_id)

    return len(ids_to_delete)


if __name__ == '__main__':
    main()
