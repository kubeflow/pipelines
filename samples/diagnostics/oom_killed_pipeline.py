"""Sample pipeline to trigger and test OOMKilled (Exit status 137)."""

from kfp import compiler
from kfp import dsl

@dsl.component(base_image='python:3.9-slim')
def fail_oom_task():
    print("Starting task: allocating 500MB of RAM to exceed the container 50Mi limit...")
    memory_hog = []
    for _ in range(50):
        memory_hog.append(b'X' * (10 * 1024 * 1024))
    print("Done (this should not be reached).")
    
@dsl.pipeline(
    name='oom-killed-pipeline',
    description='Pipeline to test and verify OOMKilled diagnostic detection in KFP UI.',
)
def oom_killed_pipeline():
    task = fail_oom_task()
    task.set_memory_limit('50Mi')
    task.set_memory_request('50Mi')
    
    
if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=oom_killed_pipeline,
        package_path='samples/diagnostics/oom_killed_pipeline.yaml'
    )
    print("Compiled successfully")
    