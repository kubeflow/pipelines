from kfp import dsl


@dsl.component
def sum_numbers(a: int, b: int) -> int:
    return a + b


@dsl.pipeline
def pipeline(
    cpu_limit: str = '4000m',
    memory_limit: str = '15G',
    accelerator_type: str = 'NVIDIA_TESLA_P4',
    accelerator_limit: str = '1',
):
    sum_numbers_task = sum_numbers(a=1, b=2)
    sum_numbers_task.set_cpu_limit(cpu_limit)
    sum_numbers_task.set_memory_limit(memory_limit)
    sum_numbers_task.set_accelerator_type(accelerator_type)
    sum_numbers_task.set_accelerator_limit(accelerator_limit)


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace('.py', '.yaml'))
