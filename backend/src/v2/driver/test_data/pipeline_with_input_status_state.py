
from kfp import dsl


@dsl.component
def echo_state(status: dsl.PipelineTaskFinalStatus):
    print(status)
    assert(status.state == 'SUCCEEDED')
    assert('status-state-pipeline' in status.pipeline_job_resource_name)
    assert(status.pipeline_task_name == 'exit-handler-1')

@dsl.component
def some_task():
    print('Executing some_task()...')

@dsl.pipeline
def status_state_pipeline():
    echo_state_task = echo_state()
    with dsl.ExitHandler(exit_task=echo_state_task):
        some_task()

if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=status_state_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
