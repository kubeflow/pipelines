
import os
import tempfile
from typing import Callable
import unittest
import yaml

from kfp import compiler, components, dsl, aws
from kfp.components import InputPath, OutputPath
def preprocess(uri: str, some_int: int, output_parameter_one: OutputPath(int),
               output_dataset_one: OutputPath('Dataset')):
    '''Dummy Preprocess Step.'''
    with open(output_dataset_one, 'w') as f:
        f.write('Output dataset')
    with open(output_parameter_one, 'w') as f:
        f.write("{}".format(1234))


def train(dataset: InputPath('Dataset'),
          model: OutputPath('Model'),
          num_steps: int = 100):
    '''Dummy Training Step.'''

    with open(dataset, 'r') as input_file:
        input_string = input_file.read()
        with open(model, 'w') as output_file:
            for i in range(num_steps):
                output_file.write("Step {}\n{}\n=====\n".format(i, input_string))


preprocess_op = components.create_component_from_func(preprocess,
                                                      base_image='python:3.9')
train_op = components.create_component_from_func(train)

@dsl.pipeline(pipeline_root='gs://output-directory/v2-artifacts',name='my-test-pipeline')
def v2_compatible_two_step_pipeline():
    preprocess_task = preprocess_op(uri='uri-to-import', some_int=12)
    train_task = train_op(
        num_steps=preprocess_task.outputs['output_parameter_one'],
        dataset=preprocess_task.outputs['output_dataset_one'])
    dsl.get_pipeline_conf().add_op_transformer(
        aws.use_aws_secret(aws_region="us-west-2"))

kfp_compiler = compiler.Compiler(
            mode=dsl.PipelineExecutionMode.V2_COMPATIBLE)
kfp_compiler.compile(v2_compatible_two_step_pipeline, package_path="/Users/xiyue/go/src/kubeflow/pipelines/sdk/python/kfp/compiler/aws.yaml")
