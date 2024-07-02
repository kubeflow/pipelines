# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import datetime
import json

import kfp
from kfp.v2 import compiler
from kfp.v2 import dsl
from kfp.v2.dsl import (
    Artifact,
    ClassificationMetrics,
    Input,
    InputPath,
    Metrics,
    Output,
    OutputPath,
)
from kfp.v2.dsl import component
from kfp.v2.google.client import AIPlatformClient  # noqa: F811

TIMESTAMP = datetime.datetime.now().strftime("%Y%m%d%H%M%S")

PIPELINE_ROOT = f"gs://model-evaluation-test-data/pipeline_root/"
evaluation_op = kfp.components.load_component_from_file("component.yaml")
batch_prediction_op = kfp.components.load_component_from_file(
    "batch_prediction_component.yaml")

PROBLEM_SPEC = {
    "classification": {
        "type": 1,
        "class_names": ["1", "0"],
        "ground_truth_column_spec": {
            "name": "instance",
            "sub_column_spec": {
                "name": "male",
            },
        },
        "prediction_score_column_spec": {
            "name": "prediction",
            "sub_column_spec": {
                "name": "scores",
            },
        },
        "prediction_label_column_spec": {
            "name": "prediction",
            "sub_column_spec": {
                "name": "scores",
            },
        },
    },
}


@component(packages_to_install=["google-cloud-aiplatform"])
def read_gcs_destination_from_batch_prediction_job(
    batch_prediction_job: Input[Artifact]) -> str:
  from google.cloud import aiplatform

  batch_prediction_job_uri = batch_prediction_job.uri

  aiplatform.init(project="model-evaluation-dev")

  bp_resource_path = batch_prediction_job_uri.replace("aiplatform://v1/", "")
  bp_job = aiplatform.BatchPredictionJob(bp_resource_path)

  return bp_job.output_info.gcs_output_directory


@component
def save_config_to_gcs(project_id: str, region: str, staging_dir: str,
                       problem_spec: str, gcs_source: str,
                       config_path: OutputPath("EvaluationConfig")):

  import json

  config = {
      "name": "xwx-test",
      "data_source": {
          "gcs_source": {
              "format": 1,
              "files": [gcs_source + "/prediction.results-*"],
          },
      },
      "problem": json.loads(problem_spec),
      "execution": {
          "dataflow_beam": {
              "project_id":
                  project_id,
              "region":
                  region,
              "dataflow_job_prefix":
                  "dataflow-test-",
              "service_account":
                  "977012026409-compute@developer.gserviceaccount.com",
              "dataflow_staging_dir":
                  staging_dir + "dataflow_staging/",
              "dataflow_temp_dir":
                  staging_dir + "dataflow_temp/",
          },
      },
  }

  with open(config_path, "w") as writer:
    writer.write(json.dumps(config))


@component
def read_output_file(
    output_path: InputPath("Metrics"), metrics: Output[ClassificationMetrics]):

  import json

  def numberfy(obj):
    if isinstance(obj, dict):
      return {
          key: numberfy(value) if key not in ["id", "displayName"] else value
          for (key, value) in obj.items()
      }
    if isinstance(obj, list):
      return [numberfy(value) for value in obj]
    if isinstance(obj, str):
      return int(obj) if obj.isnumeric() else obj
    return obj

  with open(output_path + "/metrics", "rb") as f:
    metrics.metadata = numberfy(
        json.loads(f.read())["slicedMetrics"][0]["metrics"]["classification"])


@dsl.pipeline(
    name="bp-dataflow-pipeline",
    pipeline_root=PIPELINE_ROOT,
)
def my_pipeline(
    problem_spec: str = json.dumps(PROBLEM_SPEC),
    project_id: str = "model-evaluation-dev",
    location: str = "us-central1",
    staging_dir: str = "gs://model-evaluation-test-data/dataflow-staging/",
    model_name:
    str = "projects/model-evaluation-dev/locations/us-central1/models/2539089007883583488",
    gcs_source: list = ["gs://xwx-us-central1-test/Heart_study_dataset_4k.csv"],
    gcs_destination_prefix: str = "gs://model-evaluation-dev-aip-20210831002219",
    input_format: str = "csv",
    output_format: str = "jsonl"):

  model_importer = dsl.importer(
      artifact_uri=model_name,
      artifact_class=dsl.Model,
      reimport=False)

  batch_prediction_task = batch_prediction_op(
      project=project_id,
      model=model_importer.output,
      job_display_name="import-model-batch-predict",
      gcs_source=gcs_source,
      instances_format=input_format,
      predictions_format=output_format,
      gcs_destination_prefix=gcs_destination_prefix)

  gcs_destination = read_gcs_destination_from_batch_prediction_job(
      batch_prediction_task.outputs["batchpredictionjob"])

  config_file = save_config_to_gcs(project_id, location, staging_dir,
                                   problem_spec, gcs_destination.output)

  job = evaluation_op(
      project_id=project_id,
      location=location,
      staging_dir=staging_dir,
      config_uri=config_file.output)

  read_output_file(output_path=job.outputs["output_uri"])


compiler.Compiler().compile(
    pipeline_func=my_pipeline,
    package_path="pipeline-" + TIMESTAMP + ".json",
    type_check=False)

api_client = AIPlatformClient(
    project_id="model-evaluation-dev", region="us-central1")

response = api_client.create_run_from_job_spec(
    "pipeline-" + TIMESTAMP + ".json",
    pipeline_root=PIPELINE_ROOT,
)
