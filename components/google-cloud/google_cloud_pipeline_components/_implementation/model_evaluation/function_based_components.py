"""Python function-based components used in KFP pipelines."""

from kfp.dsl import Artifact
from kfp.dsl import component
from kfp.dsl import Input
from kfp.dsl import OutputPath


@component
def convert_artifact_to_string(
    output: OutputPath(str), input_artifact: Input[Artifact]
):
  path = input_artifact.path
  if path.startswith('/gcs/'):
    path = 'gs://' + path[len('/gcs/') :]
  with open(output, 'w') as f:
    f.write(path)


@component
def add_json_escape_class_labels(output: OutputPath(str), class_labels: list):
  import json

  json_escaped_class_labels = json.dumps(class_labels).replace('"', '\\"')
  print(json_escaped_class_labels)
  with open(output, 'w') as f:
    f.write(json_escaped_class_labels)
