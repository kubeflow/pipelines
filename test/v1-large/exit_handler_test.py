import json

import kfp
from kfp import compiler
from kfp import components
from kfp import dsl


@components.create_component_from_func
def success_op():
    """Prints a message."""
    print("Success")


@components.create_component_from_func
def fail_op():
    """Fails."""
    import sys
    print("Fail")
    sys.exit(1)

@components.create_component_from_func
def exit_handler():
    print("Exit handler executed")


@dsl.pipeline(name='pipeline-with-exit-handler')
def pipeline_exit_handler():
    exit_handler_op = exit_handler()

    with dsl.ExitHandler(exit_handler_op):
        success_op()
        fail_op()


def test_exit_handler():

    client = kfp.Client(host="http://localhost:8888")
    run = client.create_run_from_pipeline_func(
        pipeline_exit_handler,
        arguments={},
    )
    result = run.wait_for_run_completion(timeout=600)
    assert result.run.status.lower() == "failed"
    workflow_manifest_str = result.pipeline_runtime.workflow_manifest
    workflow_manifest = json.loads(workflow_manifest_str)
    nodes = workflow_manifest.get('status', {}).get('nodes', {})
    for node in nodes.values():
        if node.get("templateName") == "exit-handler":
            assert node.get("phase") == "Succeeded"


if __name__ == '__main__':
  compiler.Compiler().compile(pipeline_exit_handler, __file__ + '.yaml')