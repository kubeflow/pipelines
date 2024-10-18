import json
import kfp.dsl as dsl
from kfp import components
from kfp.dsl.types import Integer
from typing import NamedTuple


def create_worker_spec(workerNum: int=0) -> NamedTuple(
    'CreatWorkerSpec',
    [
      ('worker_spec', dict),
    ]):
    """
    Creates tf-job worker spec
    """

    worker = {}
    
    if workerNum > 0:
      worker = {
       "replicas": workerNum ,
       "restartPolicy": "OnFailure",
       "template": {
          "spec": {
            "containers": [
              {
                "command": [
                  "python",
                  "/opt/model.py"
                ],
                "args": [
                  "--tf-train-steps=60"
                ],
                "image": "liuhougangxa/tf-estimator-mnist",
                "name": "tensorflow",
              }
            ]
          }
       }
    }
    from collections import namedtuple
    worker_spec_output = namedtuple(
        'MyWorkerOutput',
        ['worker_spec'])
    return worker_spec_output(worker)

worker_spec_op = components.func_to_container_op(
    create_worker_spec, base_image='tensorflow/tensorflow:1.11.0-py3')

@dsl.pipeline(
    name="Launch kubeflow tfjob",
    description="An example to launch tfjob."
)
def mnist_train(name: str="mnist",
        namespace: str="kubeflow",
        workerNum: int=3,
        ttlSecondsAfterFinished: int=-1,
        tfjobTimeoutMinutes: int=60,
        deleteAfterDone =False):
    tfjob_launcher_op = components.load_component_from_file("./component.yaml")
    # tfjob_launcher_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/kubeflow/launcher/component.yaml')
    
    chief = {
       "replicas": 1,
       "restartPolicy": "OnFailure",
       "template": {
          "spec": {
            "containers": [
              {
                "command": [
                  "python",
                  "/opt/model.py"
                ],
                "args": [
                  "--tf-train-steps=60"
                ],
                "image": "liuhougangxa/tf-estimator-mnist",
                "name": "tensorflow",
              }
            ]
          }
       }
    }

    worker_spec_create = worker_spec_op(workerNum)

    tfjob_launcher_op(
        name=name,
        namespace=namespace,
        ttl_seconds_after_finished=ttlSecondsAfterFinished,
        worker_spec=worker_spec_create.outputs['worker_spec'],
        chief_spec=chief,
        tfjob_timeout_minutes=tfjobTimeoutMinutes,
        delete_finished_tfjob=deleteAfterDone
    )

if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(mnist_train, __file__ + ".tar.gz")