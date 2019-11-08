from kfp import dsl

def echo_op():
    return dsl.ContainerOp(
        name='echo',
        image='library/bash:4.4.23',
        command=['sh', '-c'],
        arguments=['echo "hello world"']
    )

@dsl.pipeline(
    name='My first pipeline',
    description='A hello world pipeline.'
)
def hello_world_pipeline():
    echo_task = echo_op()
