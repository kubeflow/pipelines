# Copyright 2025 The Kubeflow Authors
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

"""A pipeline that uses importer with download_to_workspace and reads the artifact."""
from kfp import dsl, compiler
from kfp.dsl import importer


@dsl.component
def read_imported(data: dsl.Input[dsl.Dataset]) -> str:
    with open(data.path, "r") as f:
        content = f.read()
    print(f"Imported content length: {len(content)}")
    return content


@dsl.pipeline(
    name="pipeline-with-importer-workspace",
    description="Importer downloads an artifact into workspace; downstream reads it",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
            size='1Gi',
            kubernetes=dsl.KubernetesWorkspaceConfig(
                pvcSpecPatch={'storageClassName': 'standard'}
            ),
        ),
    ),
)
def pipeline_with_importer_workspace() -> str:
    ds = importer(
        artifact_uri='minio://mlpipeline/sample/sample.txt',
        artifact_class=dsl.Dataset,
        reimport=False,
        metadata={'source': 'sample'},
        download_to_workspace=True,
    )

    r = read_imported(data=ds.output)
    return r.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_importer_workspace,
        package_path=__file__.replace('.py', '.yaml'))