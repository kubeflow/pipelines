
from kfp import dsl


@dsl.component
def echo_state(status: dsl.PipelineTaskFinalStatus):
    assert(status.state == 'COMPLETE')
    assert('status-state-pipeline' in status.pipeline_job_resource_name)
    assert(status.pipeline_task_name == 'exit-handler-1')
    #TODO: Add assert statements to validate status.error_code and status.error_message values once those fields have been implemented.

@dsl.component
def some_task():
    print('Executing some_task()...')

@dsl.pipeline
def status_state_pipeline():
    echo_state_task = echo_state()
    with dsl.ExitHandler(exit_task=echo_state_task):
        some_task()
