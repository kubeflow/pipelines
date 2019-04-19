import kfp
import arena
import kfp.dsl as dsl
import argparse

FLAGS = None

@dsl.pipeline(
  name='pipeline to run mpi job',
  description='shows how to run mpi job.'
)
def sample_mpirun_pipeline(image="registry.cn-hangzhou.aliyuncs.com/tensorflow-samples/horovod:0.13.11-tf1.10.0-torch0.4.0-py3.5",
						   batch_size="64",
						   optimizer='momentum'):
  """A pipeline for end to end machine learning workflow."""
  data=["user-susan:/training"]
  env=["NCCL_DEBUG=INFO", ""]
  gpus=1

  train=arena.mpi_job_op(
  	name="all-reduce",
  	image=image,
  	env=env,
  	command="""
  	mpirun python /benchmarks/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --model resnet101 \
  	--batch_size {0}  --variable_update horovod --optimizer {1}\
  	--summary_verbosity=3 --save_summaries_steps=10
  	""".format(batch_size, optimizer)
  )

  
if __name__ == '__main__':
  parser = argparse.ArgumentParser()
  parser.add_argument('--image', type=str,
                      default="registry.cn-hangzhou.aliyuncs.com/tensorflow-samples/horovod:0.13.11-tf1.10.0-torch0.4.0-py3.5",
                      help='docker image.')
  parser.add_argument('--batch_size', type=str, default="64",
                      help='batch size.')
  parser.add_argument('--optimizer', type=str, default="momentum",
                      help='optimizer.')
  FLAGS, unparsed = parser.parse_known_args()

  image = FLAGS.image
  batch_size = FLAGS.batch_size
  optimizer = FLAGS.optimizer

  EXPERIMENT_NAME="resnet101-allreduce"
  RUN_ID="run"
  KFP_SERVICE="ml-pipeline.kubeflow.svc.cluster.local:8888"
  import kfp.compiler as compiler
  compiler.Compiler().compile(sample_mpirun_pipeline, __file__ + '.tar.gz')
  client = kfp.Client(host=KFP_SERVICE)
  try:
    experiment_id = client.get_experiment(experiment_name=EXPERIMENT_NAME).id
  except:
    experiment_id = client.create_experiment(EXPERIMENT_NAME).id
  run = client.run_pipeline(experiment_id, RUN_ID, __file__ + '.tar.gz',
                            params={'image':image,
                                     'batch_size':batch_size,
                                    'optimizer':optimizer})