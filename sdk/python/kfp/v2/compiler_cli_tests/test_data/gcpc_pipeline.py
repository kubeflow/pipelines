from tabnanny import verbose

import google.cloud.aiplatform as aip
from google_cloud_pipeline_components import aiplatform as gcc_aip
import kfp
from kfp.v2.dsl import component
from kfp.v2.dsl import Input

DISPLAY_NAME = "test-vertex-dataset"
PIPELINE_NAME = "test-vertex-dataset"
PROJECT_ID = 'cjmccarthy-kfp'

from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from kfp.v2.dsl import Input


@component(
    kfp_package_path='git+https://github.com/chensun/pipelines@custom-type#egg=kfp&subdirectory=sdk/python',
    packages_to_install=['google_cloud_pipeline_components'],
)
def dummy_op(artifact: Input[VertexDataset]):
    print('artifact.type: ', type(artifact))
    print('artifact.name: ', artifact.name)
    print('artifact.uri: ', artifact.uri)
    print('artifact.metadata: ', artifact.metadata)


@kfp.dsl.pipeline(name=PIPELINE_NAME)
def pipeline():
    ds_op = gcc_aip.ImageDatasetCreateOp(
        project=PROJECT_ID,
        display_name=DISPLAY_NAME,
        gcs_source='gs://cloud-samples-data/vision/automl_classification/flowers/all_data_v2.csv',
        import_schema_uri=aip.schema.dataset.ioformat.image
        .single_label_classification,
    )

    dummy_op(ds_op.outputs['dataset'])


if __name__ == '__main__':
    from kfp.v2 import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace('.py', '.json'))
