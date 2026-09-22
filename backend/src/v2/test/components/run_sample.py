# Copyright 2021 The Kubeflow Authors
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

from pathlib import Path

from kfp import compiler
from kfp import dsl

SHELL_COMMAND = r'''backend_compiler_path="$0"
sample_path="$1"
output_dir="$2"
export KF_PIPELINES_ENDPOINT="$3"
export KF_PIPELINES_UI_ENDPOINT="$4"
launcher_v2_image="$5"
driver_image="$6"

# install kfp v2 backend compiler
chmod +x "$backend_compiler_path"
cp "$backend_compiler_path" /usr/local/bin/kfp-v2-compiler
# Install the native SDK and sample runner dependencies from this checkout.
(cd backend/src/v2/test && pip install -r requirements.txt)
# run test sample
python3 \
  -u \
  -m "$sample_path" \
  --pipeline_root "$output_dir" \
  --launcher_v2_image "$launcher_v2_image" \
  --driver_image "$driver_image"
'''


@dsl.container_component
def run_sample(
    name: str,
    sample_path: str,
    gcs_root: str,
    external_host: str,
    backend_compiler: dsl.Input[dsl.Artifact],
    host: str = 'http://ml-pipeline:8888',
    launcher_v2_image: str = 'gcr.io/ml-pipeline/kfp-launcher-v2:latest',
    driver_image: str = 'gcr.io/ml-pipeline/kfp-driver:latest',
):
    """Run a v2 sample using the compiler artifact and native SDK
    dependencies."""
    return dsl.ContainerSpec(
        image='python:3.11-alpine',
        command=[
            'sh',
            '-exc',
            SHELL_COMMAND,
            backend_compiler.path,
            sample_path,
            dsl.ConcatPlaceholder([gcs_root, '/', name]),
            host,
            external_host,
            launcher_v2_image,
            driver_image,
        ],
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        run_sample, package_path=str(Path(__file__).with_suffix('.yaml')))
