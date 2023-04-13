from typing import Dict, List
import kfp.v2.components.experimental as components
from kfp.v2.compiler.experimental import compiler
import kfp.v2.dsl.experimental as dsl


@dsl.component(
    # Usually this is not needed. By default, it will install kfp from PyPi using the same version as used for compiling the pipeline.
    kfp_package_path='git+https://github.com/kubeflow/pipelines#egg=kfp&subdirectory=sdk/python',
)
def iter_gen_op(iters: dsl.OutputPath(str)):
    with open(iters, 'w') as f:
        f.write('[[1,2,3],[4,5]]')


@dsl.component(
    # Usually this is not needed. By default, it will install kfp from PyPi using the same version as used for compiling the pipeline.
    kfp_package_path='git+https://github.com/kubeflow/pipelines#egg=kfp&subdirectory=sdk/python',
)
def print_str_op(text: str):
    print(text)


@dsl.pipeline(
    name='conditional-execution-pipeline-v1-yaml-1029',
    description='Shows how to use dsl.Condition().')
def condition_pipeline():
    iter_gen = iter_gen_op()
    with dsl.ParallelFor(iter_gen.outputs['iters']) as it:
        with dsl.ParallelFor(it) as it2:
            with dsl.Condition(it2 == '5', 'equals to 5'):
                print_str_op(text=it)


compiler.Compiler().compile(
    pipeline_func=condition_pipeline, package_path='condition_pipeline.json')
