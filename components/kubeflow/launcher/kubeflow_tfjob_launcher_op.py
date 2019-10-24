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

from kfp import dsl

def kubeflow_tfjob_launcher_op(container_image, command, number_of_workers: int, number_of_parameter_servers: int, tfjob_timeout_minutes: int, output_dir=None, step_name='TFJob-launcher'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf:e9b96de317989a9673ef88d88fb9dab9dac3005f',
        arguments = [
            '--workers', number_of_workers,
            '--pss', number_of_parameter_servers,
            '--tfjob-timeout-minutes', tfjob_timeout_minutes,
            '--container-image', container_image,
            '--output-dir', output_dir,
            '--ui-metadata-type', 'tensorboard',
            '--',
        ] + command,
        file_outputs = {'train': '/output.txt'},
        output_artifact_paths={
            'mlpipeline-ui-metadata': '/mlpipeline-ui-metadata.json',
        },
    )
