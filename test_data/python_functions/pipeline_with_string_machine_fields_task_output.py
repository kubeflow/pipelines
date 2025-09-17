from kfp import dsl


@dsl.component
def cpu_limit() -> str:
    return '4000m'


@dsl.component
def memory_limit() -> str:
    return '15G'


@dsl.component
def accelerator_type() -> str:
    return 'NVIDIA_TESLA_P4'


@dsl.component
def accelerator_limit() -> str:
    return '1'


@dsl.component
def sum_numbers(a: int, b: int) -> int:
    return a + b


@dsl.pipeline
def pipeline():
    sum_numbers_task = sum_numbers(a=1, b=2)
    sum_numbers_task.set_cpu_limit(cpu_limit().output)
    sum_numbers_task.set_memory_limit(memory_limit().output)
    sum_numbers_task.set_accelerator_type(accelerator_type().output)
    sum_numbers_task.set_accelerator_limit(accelerator_limit().output)


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace('.py', '.yaml'))
