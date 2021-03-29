#!/usr/bin/env python3
# Copyright 2021 Google LLC
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

import kfp
from .use_run_info import pipeline_with_run_info

MINUTE = 60  # seconds


def main(
    pipeline_root: str = 'gs://your-bucket/path/to/workdir',
    host: str = 'http://ml-pipeline:8888',
    launcher_image: 'URI' = None,
    experiment: str = 'v2_sample_test_samples'
):
    client = kfp.Client(host=host)
    run_result = client.create_run_from_pipeline_func(
        pipeline_with_run_info,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
        arguments={},
        experiment_name=experiment,
    )
    print("Run details page URL:")
    print(f"{host}/#/runs/details/{run_result.run_id}")
    run_response = run_result.wait_for_run_completion(timeout=10 * MINUTE)
    run = run_response.run
    from pprint import pprint
    pprint(run_response.run)
    print("Run details page URL:")
    print(f"{host}/#/runs/details/{run.id}")
    assert run.status == 'Succeeded'
    # TODO: add more MLMD verification


if __name__ == '__main__':
    import fire
    fire.Fire(main)
