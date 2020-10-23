import kfp
from kfp.components import create_component_from_func


@create_component_from_func
def sleep_op(seconds: float) -> None:
    import time
    time.sleep(seconds)


@create_component_from_func
def generate_random_number_op(dummy: str = '') -> int:
    import random
    return random.randrange(0, 1000000)


@create_component_from_func
def assert_equal_op(object_1, object_2):
    if not object_1 == object_2:
        raise ValueError('Actually: "{}" != "{}"'.format(object_1, object_2))


@create_component_from_func
def assert_not_equal_op(object_1, object_2):
    if not object_1 != object_2:
        raise ValueError('Actually: "{}" == "{}"'.format(object_1, object_2))


def caching_pipeline():
    # All outputs of successfull executions are cached
    number1_task = generate_random_number_op()

    # By default, the cached execution outputs are reused
    number2_task = generate_random_number_op().after(number1_task)
    assert_equal_op(number2_task.output, number1_task.output)

    # Cached outputs are only reused when the component is the same and all input arguments are the same.
    # This execution won't be skipped the first time you run the pipeline.
    # (In subsequent pipeline runs this task will start reusing outputs, but they would still be different from number1_task.)
    number3_task = generate_random_number_op(dummy='foo').after(number1_task)
    assert_not_equal_op(number3_task.output, number1_task.output)

    
    wait_task = sleep_op(seconds=10).after(number1_task)

    # Setting max_cache_staleness control when the the cached execution outputs can be reused
    # After waiting, the data should be ~10 seconds old
    # So this task will be using cache
    number4_task = generate_random_number_op().after(wait_task)
    number4_task.execution_options.caching_strategy.max_cache_staleness = 'P30s'
    assert_equal_op(number4_task.output, number1_task.output)

    # But this task won't use the cache since it asks for the data to be not older than 5 seconds
    number5_task = generate_random_number_op().after(wait_task)
    number5_task.execution_options.caching_strategy.max_cache_staleness = 'P5s'
    assert_equal_op(number5_task.output, number1_task.output)

    
    # Setting max_cache_staleness to 'P0D' (0-day period), prevents the cached execution outputs from being reused for this task
    number6_task = generate_random_number_op().after(number1_task)
    number6_task.execution_options.caching_strategy.max_cache_staleness = 'P0D'
    assert_not_equal_op(number6_task.output, number1_task.output)


if __name__ == '__main__':
    # kfp_endpoint = None
    # kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(caching_pipeline, arguments={})
    kfp.compiler.Compiler().compile(caching_pipeline, __file__ + '.yaml')