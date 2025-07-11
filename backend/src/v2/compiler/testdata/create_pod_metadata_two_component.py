from kfp import dsl
from kfp.compiler import compiler
from kfp.kubernetes import add_pod_annotation
from kfp.kubernetes import add_pod_label


@dsl.component
def print_op(input: str):
    print(input)

@dsl.pipeline
def create_pod_metadata_two_component():
    # task_a and task_b are set with different task-specific metadata and the separate pods representing each task should
    # be set with their separate, corresponding metadata.
    task_a = print_op(input='task-a')
    add_pod_label(task_a, 'task', 'a')
    add_pod_annotation(task_a, 'task', 'a')

    task_b = print_op(input='task-b')
    add_pod_label(task_b, 'task', 'b')
    add_pod_annotation(task_b, 'task', 'b')

    # task_c is set with no task-specific metadata, as a control group.
    task_c = print_op(input='task-c')

if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=create_pod_metadata_two_component,
        package_path=__file__.replace('.py', '.yaml'))
