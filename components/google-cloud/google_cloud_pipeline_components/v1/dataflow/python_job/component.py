from typing import List

from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import OutputPath


@container_component
def dataflow_python(
    project: str,
    python_module_path: str,
    temp_location: str,
    # TODO(b/243411151): misalignment of arguments in documentation vs function signature.
    gcp_resources: OutputPath(str),
    location: str = 'us-central1',
    requirements_file_path: str = '',
    args: List[str] = [],
):
  """Launch a self-executing beam python file on Google Cloud using the DataflowRunner.

    Args:
        project (str): Required. Project to create the Dataflow job in.
        location (Optional[str]): Location for creating the Dataflow job. If not
          set, default to us-central1.
        python_module_path (str): The GCS path to the python file to run.
        temp_location (str): A GCS path for Dataflow to stage temporary job
          files created during the execution of the pipeline.
        requirements_file_path (Optional[str]): The GCS path to the pip
          requirements file. args(Optional[List[str]]): The list of args to pass
          to the python file. Can include additional parameters for the beam
          runner.

    Returns:
        gcp_resources (str):
            Serialized gcp_resources proto tracking the Dataflow job.
            For more details, see
            https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:2.0.0b1',
      command=[
          'python3', '-u', '-m',
          'google_cloud_pipeline_components.container.v1.dataflow.dataflow_launcher'
      ],
      args=[
          '--project',
          project,
          '--location',
          location,
          '--python_module_path',
          python_module_path,
          '--temp_location',
          temp_location,
          '--requirements_file_path',
          requirements_file_path,
          '--args',
          args,
          '--gcp_resources',
          gcp_resources,
      ])
