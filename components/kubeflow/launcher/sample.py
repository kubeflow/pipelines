import json
from kfp import components
import kfp.dsl as dsl

@dsl.pipeline(
    name="Launch kubeflow tfjob",
    description="An example to launch tfjob."
)
def mnist_train(
        name="mnist",
        namespace="kubeflow",
        workerNum=3,
        ttlSecondsAfterFinished=-1,
        tfjobTimeoutMinutes=60,
        deleteAfterDone=False):
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
                  "--tf-train-steps=6000"
                ],
                "image": "liuhougangxa/tf-estimator-mnist",
                "name": "tensorflow",
              }
            ]
          }
       }
    }
    worker = {}
    if workerNum > 0:
      worker = {
       "replicas": workerNum,
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
                  "--tf-train-steps=6000"
                ],
                "image": "liuhougangxa/tf-estimator-mnist",
                "name": "tensorflow",
              }
            ]
          }
       }
    }
    tfjob_launcher_op(
        name=name,
        namespace=namespace,
        ttl_seconds_after_finished=ttlSecondsAfterFinished,
        worker_spec=worker,
        chief_spec=chief,
        tfjob_timeout_minutes=tfjobTimeoutMinutes,
        delete_finished_tfjob=deleteAfterDone
    )

if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(mnist_train, __file__ + ".tar.gz")
