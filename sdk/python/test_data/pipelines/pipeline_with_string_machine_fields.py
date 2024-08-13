from kfp import dsl


@dsl.component
def sum_numbers(a: int, b: int) -> int:
    return a + b


@dsl.pipeline
def pipeline():
    sum_numbers_task = sum_numbers(a=1, b=2)
    sum_numbers_task.set_cpu_limit('4000m')
    sum_numbers_task.set_memory_limit('15G')
    sum_numbers_task.set_accelerator_type('NVIDIA_TESLA_P4')
    sum_numbers_task.set_accelerator_limit('1')


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace('.py', '.yaml'))
