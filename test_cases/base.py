from typing import Dict, List

from kfp import compiler
from kfp import components
from kfp import dsl
from kfp.dsl import component


@component
def print_op(text: str) -> List[str]:
    print(text)
    return [text, text]


@component
def print_struct(d: list):
    print(d)


@dsl.pipeline(name='pipeline-with-pipelineparam-containing-format')
def my_pipeline(name: str = 'KFP'):
    print_task = print_op(text='Hello {}'.format(name))
    print_struct(d=print_task.outputs)
    # print_op(text='{}, again.'.format(print_task.output))

    # new_value = f' and {name}.'
    # with dsl.ParallelFor(['1', '2']) as item:
    #     print_op2(text1=item, text2=new_value)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.json'))

if __name__ == '__main__':
    import datetime
    import warnings

    from google.cloud import aiplatform
    warnings.filterwarnings("ignore")

    ir_file = __file__.replace('.py', '_list.json')
    compiler.Compiler().compile(pipeline_func=my_pipeline, package_path=ir_file)
    aiplatform.PipelineJob(
        template_path=ir_file,
        pipeline_root='gs://cjmccarthy-kfp-default-bucket',
        display_name=str(datetime.datetime.now())).submit()

    # "inputs": {
    #   "parameters": {
    #     "pipelinechannel--args-generator-op-Output-loop-item": {
    #       "componentInputParameter": "pipelinechannel--args-generator-op-Output-loop-item"
    #     },
    #     "pipelinechannel--args-generator-op-Output-loop-item-subvar-B_b": {
    #       "componentInputParameter": "pipelinechannel--args-generator-op-Output-loop-item",
    #       "parameterExpressionSelector": "parseJson(string_value)[\"B_b\"]"
    #     },
    #     "pipelinechannel--flip-coin-op-Output": {
    #       "componentInputParameter": "pipelinechannel--flip-coin-op-Output"
    #     }
    #   }
    # },
    # "parameterIterator": {
    #   "itemInput": "pipelinechannel--args-generator-op-Output-loop-item-subvar-B_b-loop-item",
    #   "items": {
    #     "inputParameter": "pipelinechannel--args-generator-op-Output-loop-item-subvar-B_b"
    #   }
    # },
