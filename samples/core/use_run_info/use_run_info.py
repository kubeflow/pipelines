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
import kfp.dsl as dsl


def get_run_info(run_id: str):
    '''Example of getting run info for current pipeline run'''
    import kfp
    import json
    print(f'Current run ID is {run_id}.')
    client = kfp.Client(host='http://ml-pipeline:8888')
    run_info = client.get_run(run_id=run_id)
    # Hide verbose info
    run_info.run.pipeline_spec.workflow_manifest = None
    print(run_info.run)


get_run_info_component = kfp.components.create_component_from_func(
    func=get_run_info,
    packages_to_install=['kfp'],
)


@dsl.pipeline(
    name="pipeline_with_run_info",
    description=
    "A pipeline that demonstrates how to use run information, including run ID etc."
)
def pipeline_with_run_info(run_id: str = '[[RunUUID]]'):
    '''[[RunUUID]] macro inside a pipeline level parameter will be populated with KFP Run ID at runtime.'''
    run_info_op = get_run_info_component(run_id=run_id)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(pipeline_with_run_info, __file__ + '.yaml')
