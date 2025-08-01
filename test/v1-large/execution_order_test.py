from kfp import compiler
from kfp import components
from kfp import dsl

from conftest import runner


def echo1_op(text1: str):
  print(text1)


def echo2_op(text2: str):
  print(text2)


echo1_component = components.create_component_from_func(echo1_op)
echo2_component = components.create_component_from_func(echo2_op)


@dsl.pipeline(
  name='execution-order-pipeline',
  description='A pipeline to demonstrate execution order management.'
)
def execution_order_pipeline(text1: str='message 1', text2: str='message 2'):
  """A two step pipeline with an explicitly defined execution order."""
  step1_task = echo1_component(text1=text1)
  step2_task = echo2_component(text2=text2)
  step2_task.after(step1_task)

def test_execution_order_pipeline():
    runner(execution_order_pipeline)


if __name__ == '__main__':
  compiler.Compiler().compile(execution_order_pipeline, __file__ + '.yaml')
