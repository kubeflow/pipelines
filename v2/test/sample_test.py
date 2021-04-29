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

# %%
import yaml
import os

REPO_ROOT = os.path.join('..', '..')
SAMPLES_CONFIG_PATH = os.path.join(REPO_ROOT, 'samples', 'test', 'config.yaml')
SAMPLES_CONFIG = None
with open(SAMPLES_CONFIG_PATH, 'r') as stream:
    SAMPLES_CONFIG = yaml.safe_load(stream)

import kfp
import kfp.components as comp
import json
from typing import Optional

MINUTE = 60  # seconds

download_gcs_tgz = kfp.components.load_component_from_file(
    'components/download_gcs_tgz.yaml'
)
run_sample = kfp.components.load_component_from_file(
    'components/run_sample.yaml'
)
kaniko = kfp.components.load_component_from_file('components/kaniko.yaml')


@kfp.dsl.pipeline(name='v2 sample test')
def v2_sample_test(
    context: 'URI' = 'gs://your-bucket/path/to/context.tar.gz',
    launcher_destination: 'URI' = 'gcr.io/gongyuan-pipeline-test/kfp-launcher',
    gcs_root: 'URI' = 'gs://gongyuan-test/v2',
    samples_destination: 'URI' = 'gcr.io/gongyuan-pipeline-test/v2-sample-test',
    kfp_host: 'URI' = 'http://ml-pipeline:8888',
    samples_config: list = SAMPLES_CONFIG,
):
    download_src_op = download_gcs_tgz(gcs_path=context)
    download_src_op.set_display_name('download_src')
    download_src_op.execution_options.caching_strategy.max_cache_staleness = "P0D"
    build_kfp_launcher_op = kaniko(
        context_artifact=download_src_op.outputs['folder'],
        context_sub_path='v2',
        destination=launcher_destination,
        dockerfile='launcher_container/Dockerfile',
    )
    build_kfp_launcher_op.set_display_name('build_kfp_launcher')
    build_samples_image_op = kaniko(
        context_artifact=download_src_op.outputs['folder'],
        destination=samples_destination,
        dockerfile='v2/test/Dockerfile',
    )
    build_samples_image_op.execution_options.caching_strategy.max_cache_staleness = "P0D"
    build_samples_image_op.set_display_name('build_samples_image')
    with kfp.dsl.ParallelFor(samples_config) as sample:
        run_sample_op = run_sample(
            name=sample.name,
            sample_path=sample.path,
            gcs_root=gcs_root,
            external_host=kfp_host,
            launcher_image=build_kfp_launcher_op.outputs['digest']
        )
        run_sample_op.container.image = build_samples_image_op.outputs['digest']
        run_sample_op.set_display_name(f'sample_{sample.name}')


def main(
    context: str,
    host: str,
    gcr_root: str,
    gcs_root: str,
    experiment: str = 'v2_sample_test'
):
    client = kfp.Client(host=host)
    client.create_experiment(
        name=experiment,
        description='An experiment with Kubeflow Pipelines v2 sample test runs.'
    )
    run_result = client.create_run_from_pipeline_func(
        v2_sample_test, {
            'context': context,
            'launcher_destination': f'{gcr_root}/kfp-launcher',
            'gcs_root': gcs_root,
            'samples_destination': f'{gcr_root}/v2-sample-test',
            'kfp_host': host,
        },
        experiment_name=experiment
    )
    print("Run details page URL:")
    print(f"{host}/#/runs/details/{run_result.run_id}")
    run_response = run_result.wait_for_run_completion(20 * MINUTE)
    run = run_response.run
    from pprint import pprint
    # Hide verbose content
    run_response.run.pipeline_spec.workflow_manifest = None
    pprint(run_response.run)
    print("Run details page URL:")
    print(f"{host}/#/runs/details/{run_result.run_id}")
    assert run.status == 'Succeeded'
    # TODO(Bobgy): print debug info


# %%
if __name__ == "__main__":
    import fire
    fire.Fire(main)
