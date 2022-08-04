from kfp import compiler
from kfp import dsl


@dsl.component
def producer_op() -> str:
    return 'a'


@dsl.component
def dummy_op(msg: str = '') -> str:
    return msg


@dsl.pipeline(name='test-pipeline')
def my_pipeline(text: bool = True):
    first_exit_task = producer_op()
    with dsl.ExitHandler(first_exit_task):
        second_exit_task = producer_op()
        with dsl.ExitHandler(second_exit_task):
            third_exit_task = producer_op()
            with dsl.ExitHandler(third_exit_task):
                dummy_op(msg='first hello')

        dummy_task = dummy_op(msg='second hello')
        with dsl.Condition(dummy_task.output == 'second hello'):
            fouth_exit_task = producer_op()
            with dsl.ExitHandler(fouth_exit_task):
                dummy_op(msg='third hello')


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
