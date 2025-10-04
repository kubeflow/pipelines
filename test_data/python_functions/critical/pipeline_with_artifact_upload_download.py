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
"""Pipeline using ExitHandler with PipelineTaskFinalStatus."""

from kfp import compiler
from kfp import dsl
from kfp.dsl import Dataset, Output

@dsl.component(packages_to_install=["datasets==4.0.0"])
def download_dataset_and_upload_as_artifact(dataset_repo: str, dataset_name: str, output_dataset: Output[Dataset]):
    from datasets import load_dataset
    # Load data set from hugging face
    ds = load_dataset(dataset_repo, dataset_name)
    print("Downloaded Hugging Face data")
    print(f"Now saving to {output_dataset.path}")
    ds.save_to_disk(output_dataset.path)
    print(f"Saved to {output_dataset.path}")


@dsl.component
def print_dataset_info(dataset: Dataset):
    print('Information about the artifact')
    print('Name:', dataset.name)
    print('URI:', dataset.uri)
    assert "download-dataset-and-upload-as-artifact" in dataset.uri, "The URI of the downloaded artifact does not match the expected function's name that generated it"



@dsl.pipeline(name='pipeline-with-datasets', description="Download Hugging Face Data Set, upload it as an artifact and print its metadata")
def my_pipeline(dataset_repo: str = "google/frames-benchmark", dataset_name: str = ""):
    downloaded_dataset = download_dataset_and_upload_as_artifact(dataset_repo=dataset_repo, dataset_name=dataset_name)
    print_dataset_info(dataset=downloaded_dataset.output)

if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
