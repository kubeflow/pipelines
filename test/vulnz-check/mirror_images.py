# %%
from typing import NamedTuple
import os

import kfp
from kfp.components import func_to_container_op, InputPath, OutputPath

# %%
# Mirror Image


def mirror_image(
        image: str,
        source_registry: str,
        destination_registry: str,
        tag: str = '',
):
    source_image = '{}/{}:{}'.format(source_registry, image, tag)
    destination_image = '{}/{}:{}'.format(destination_registry, image, tag)
    import subprocess
    subprocess.run([
        'gcloud', 'container', 'images', 'add-tag', source_image,
        destination_image
    ])


mirror_image_component = kfp.components.create_component_from_func(
    mirror_image_imp, base_image='google/cloud-sdk:alpine'
)


# Combining all pipelines together in a single pipeline
def mirror_images_pipeline(
        version: str = '1.3.0',
        source_registry: str = 'gcr.io/ml-pipeline',
        destination_registry: str = 'gcr.io/gongyuan-pipeline-test/dev'
):
    images = [
        'persistenceagent', 'scheduledworkflow', 'frontend',
        'viewer-crd-controller', 'visualization-server', 'inverse-proxy-agent',
        'metadata-writer', 'cache-server', 'cache-deployer', 'metadata-envoy'
    ]
    with kfp.dsl.ParallelFor(images) as image:
        mirror_image_task = mirror_image_component(
            image=image,
            source_registry=source_registry,
            destination_registry=destination_registry,
            tag=version,
        )


# %%
if __name__ == '__main__':
    # Submit the pipeline for execution:
    if os.getenv('KFP_HOST') is None:
        print('KFP_HOST env var is not set')
        exit(1)
    kfp.Client(host=os.getenv('KFP_HOST')).create_run_from_pipeline_func(
        mirror_images_pipeline, arguments={'version': '1.3.0'}
    )

    # Compiling the pipeline
    # kfp.compiler.Compiler().compile(mirror_images_pipeline, __file__ + '.yaml')

# %%
