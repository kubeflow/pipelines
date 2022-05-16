from kfp import dsl


@dsl.component
def component_1(x: str) -> str:
    return x


@dsl.component
def component_2(x: str) -> str:
    return x


@dsl.component
def component_3(x: str) -> str:
    return x


@dsl.pipeline(name='pipeline-1')
def pipeline_1(x: str) -> str:
    dag_4 = pipeline_4(x=x)
    return dag_4.output


@dsl.pipeline(name='pipeline-2')
def pipeline_2(x: str) -> str:
    dag_5 = pipeline_4(x=x)
    return dag_5.output


@dsl.pipeline(name='pipeline-3')
def pipeline_3(x: str) -> str:
    task_5 = component_2(x=x)
    task_6 = component_3(x=task_5.output)
    return task_6.output


@dsl.pipeline(name='pipeline-4')
def pipeline_4(x: str) -> str:
    task_1_3 = component_1(x=x)
    task_2_4 = component_2(x=task_1_3.output)
    return task_2_4.output


@dsl.pipeline(name='pipeline-0')
def pipeline_0(x: str):
    dag_1 = pipeline_1(x=x)
    dag_2 = pipeline_2(x=dag_1.output)
    dag_3 = pipeline_3(x=dag_2.output)
