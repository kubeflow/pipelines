import kfp
from google_cloud_pipeline_components import aiplatform as gcc_aip
from typing import NamedTuple
from kfp.v2 import dsl
from kfp.v2.dsl import (ClassificationMetrics, Input, Metrics, Model, Output,
                        component)
from google_cloud_pipeline_components.types import artifact_types
import google.cloud.aiplatform as aip

DISPLAY_NAME = "automl-image-training"
PIPELINE_NAME = "automl-tabular-beans-training-v2"
PROJECT_ID = 'chesu-dev'

from kfp.v2.dsl import Artifact, Input

from google_cloud_pipeline_components.types.artifact_types import VertexModel


@component(
    kfp_package_path='git+https://github.com/chensun/pipelines@custom-type#egg=kfp&subdirectory=sdk/python',
    packages_to_install=['google_cloud_pipeline_components'],
    imports=[
        'from google_cloud_pipeline_components.types.artifact_types import VertexModel'
    ],
)
def dummy_op(artifact: Input[VertexModel]):
    print('artifact.type: ', type(artifact))
    print('artifact.name: ', artifact.name)
    print('artifact.uri: ', artifact.uri)
    print('artifact.metadata: ', artifact.metadata)


@kfp.dsl.pipeline(name=PIPELINE_NAME)
def pipeline():
    ds_op = gcc_aip.ImageDatasetCreateOp(
        project=PROJECT_ID,
        display_name=DISPLAY_NAME,
        gcs_source="gs://cloud-samples-data/vision/automl_classification/flowers/all_data_v2.csv",
        import_schema_uri=aip.schema.dataset.ioformat.image
        .single_label_classification,
    )

    training_job_run_op = gcc_aip.AutoMLImageTrainingJobRunOp(
        project=PROJECT_ID,
        display_name=DISPLAY_NAME,
        prediction_type="classification",
        model_type="CLOUD",
        base_model=None,
        dataset=ds_op.outputs["dataset"],
        model_display_name="train-automl-flowers",
        training_fraction_split=0.6,
        validation_fraction_split=0.2,
        test_fraction_split=0.2,
        budget_milli_node_hours=8000,
    )

    create_endpoint_op = gcc_aip.EndpointCreateOp(
        project=PROJECT_ID,
        display_name=DISPLAY_NAME,
    )

    dummy_op(training_job_run_op.outputs["model"])
    #dummy_op(create_endpoint_op.outputs['endpoint'])


if __name__ == '__main__':
    import kfp.v2.compiler as compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace('.py', '.json'))
