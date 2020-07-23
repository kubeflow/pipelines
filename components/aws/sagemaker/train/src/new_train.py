from train.src.sagemaker_training_spec import SageMakerTrainingSpec
from train.src.sagemaker_training_component import SageMakerTrainingComponent

from common.component_compiler import SageMakerComponentCompiler

if __name__== "__main__":
  # import sys
  # spec = SageMakerTrainingSpec(['--region', 'us-east-1', '--job_name', 'abc1234'])
  # print(spec.inputs) # {'region': 'us-east-1', 'job_name': 'abc1234'}
  # print(spec.outputs) # {'model_artifact_url_output_path': '/tmp/model-artifact-url'}

  # component = SageMakerTrainingComponent()
  # component.Do(spec)

  SageMakerComponentCompiler.compile(SageMakerTrainingComponent, "train.py", "output.yaml")