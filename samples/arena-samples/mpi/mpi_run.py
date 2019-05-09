import kfp
import arena
import kfp.dsl as dsl
import argparse

FLAGS = None

@dsl.pipeline(
  name='pipeline to run mpi job',
  description='shows how to run mpi job.'
)
def mpirun_pipeline(image="uber/horovod:0.13.11-tf1.10.0-torch0.4.0-py3.5",
						   batch_size="64",
						   optimizer='momentum',
               sync_source='https://github.com/tensorflow/benchmarks.git',
               data='user-susan:/training',
               env='NCCL_DEBUG=INFO,GIT_SYNC_BRANCH=cnn_tf_v1.9_compatible',
               gpus=1,
               workers=2,
               cpu_limit=0,
               memory_limit=0):
  """A pipeline for end to end machine learning workflow."""

  train=arena.mpi_job_op(
  	name="all-reduce",
  	image=image,
  	env=env.spllit(','),
    data=data.split(','),
    workers=workers,
    cpu_limit=cpu_limit,
    memory_limit=memory_limit,
  	command="""
  	mpirun python /benchmarks/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --model resnet101 \
  	--batch_size {0}  --variable_update horovod --optimizer {1}\
  	--summary_verbosity=3 --save_summaries_steps=10
  	""".format(batch_size, optimizer)
  )

  
if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(sample_mpirun_pipeline, __file__ + '.tar.gz')
