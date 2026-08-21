"""Sample pipeline to trigger and test ImagePullBackOff / ErrImagePull. """

from kfp import compiler
from kfp import dsl

@dsl.component(base_image='nonexistent.registry.io/kfp-diagnostics/inavlid-image:v999')
def fail_image_pull_task():
    print("This task will fail during pod provisioning with ImagePullBackOff.")
    
@dsl.pipeline(
    name='image-pull-failure-pipeline',
    description='Pipeline to test and verify ImagePullBackOff diagnostic detection in KFP UI.'
)
def image_pull_failure_pipeline():
    fail_image_pull_task()
    
if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=image_pull_failure_pipeline,
        package_path='samples/diagnostics/image_pull_failure_pipeline.yaml',
    )
    print("Compiled samples/diagnostics/image_pull_failure_pipeline.yaml successfully.")
    
    