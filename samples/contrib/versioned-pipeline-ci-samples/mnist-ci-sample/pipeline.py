import kfp.dsl as dsl
import kfp.components as components
from kfp.gcp import use_gcp_secret

@dsl.pipeline(
   name='mnist pipeline',
   description='A pipeline to train a model on mnist dataset and start a tensorboard.'
)
def mnist_pipeline(
   storage_bucket: str,
   ):
   import os
   train_step = dsl.ContainerOp(
       name='train mnist model',
       image = os.path.join(args.gcr_address, 'mnist_train:latest'),
       command = ['python', '/mnist.py'],
       arguments = ['--storage_bucket', storage_bucket],
       file_outputs = {'logdir': '/logdir.txt'},
   ).apply(use_gcp_secret('user-gcp-sa'))

   visualize_op = components.load_component_from_file('./tensorboard/component.yaml')
   visualize_step = visualize_op(tensorboard_image='gcr.io/alert-ability-264507/mnist_tensorboard:latest', logdir=train_step.outputs['logdir']).apply(use_gcp_secret('user-gcp-sa'))

if __name__ == '__main__':
   import argparse
   parser = argparse.ArgumentParser()
   parser.add_argument('--gcr_address', type = str)
   args = parser.parse_args()
   
   import kfp.compiler as compiler
   compiler.Compiler().compile(mnist_pipeline, __file__ + '.zip')