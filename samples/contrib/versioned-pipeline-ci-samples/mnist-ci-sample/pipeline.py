import kfp.dsl as dsl
from kfp.gcp import use_gcp_secret
import argparse

# Get commit id to tag image
parser = argparse.ArgumentParser()
parser.add_argument('--commit_id', help='Commit Id', type=str)
args = parser.parse_args()


@dsl.pipeline(
   name='mnist pipeline',
   description='A pipeline to train a model on mnist dataset and start a tensorboard.'
)
def mnist_pipeline(
   storage_bucket: str,
   gcr_address: str
   ):
   import os
   train_step = dsl.ContainerOp(
       name='train mnist model',
       image = os.path.join(gcr_address, 'mnist_train:latest'),
       command = ['python', '/mnist.py'],
       arguments = ['--storage_bucket', storage_bucket],
       file_outputs = {'logdir': '/logdir.txt'},
   ).apply(use_gcp_secret('user-gcp-sa'))

   visualize_step = dsl.ContainerOp(
      name = 'visualize training result with tensorboard',
      image = os.path.join(gcr_address, 'mnist_tensorboard:latest'),
      command = ['python', '/tensorboard.py'],
      arguments = ['--logdir', '%s' % train_step.outputs['logdir']],
      output_artifact_paths={'mlpipeline-ui-metadata': '/mlpipeline-ui-metadata.json'}
   ).apply(use_gcp_secret('user-gcp-sa'))

if __name__ == '__main__':
   import kfp.compiler as compiler
   compiler.Compiler().compile(mnist_pipeline, __file__ + '.zip')