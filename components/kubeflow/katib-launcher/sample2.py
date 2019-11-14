import json
from kfp import components
import kfp.dsl as dsl

@dsl.pipeline(
    name="Launch katib experiment",
    description="An example to launch katib experiment."
)
def mnist_hpo(
        name="mnist",
        namespace="kubeflow",
        goal=0.99,
        parallelTrialCount=3,
        maxTrialCount=12,
        experimentTimeoutMinutes=60,
        deleteAfterDone=True):
    objectiveConfig = {
      "type": "maximize",
      "goal": goal,
      "objectiveMetricName": "Validation-accuracy",
      "additionalMetricNames": ["accuracy"]
    }
    algorithmConfig = {"algorithmName" : "random"}
    parameters = [
      {"name": "--lr", "parameterType": "double", "feasibleSpace": {"min": "0.01","max": "0.03"}},
      {"name": "--num-layers", "parameterType": "int", "feasibleSpace": {"min": "2", "max": "5"}},
      {"name": "--optimizer", "parameterType": "categorical", "feasibleSpace": {"list": ["sgd", "adam", "ftrl"]}}
    ]
    rawTemplate = {
      "apiVersion": "batch/v1",
      "kind": "Job",
      "metadata": {
         "name": "{{.Trial}}",
         "namespace": "{{.NameSpace}}"
      },
      "spec": {
        "template": {
          "spec": {
            "restartPolicy": "Never",
            "containers": [
              {"name": "{{.Trial}}",
               "image": "docker.io/katib/mxnet-mnist-example",
               "command": [
                   "python /mxnet/example/image-classification/train_mnist.py --batch-size=64 {{- with .HyperParameters}} {{- range .}} {{.Name}}={{.Value}} {{- end}} {{- end}}"
               ]
              }
            ]
          }
        }
      }
    }
    trialTemplate = {
      "goTemplate": {
        "rawTemplate": json.dumps(rawTemplate)
      }
    }
    katib_experiment_launcher_op = components.load_component_from_file("./component.yaml")
    # katib_experiment_launcher_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/kubeflow/katib-launcher/component.yaml')
    op1 = katib_experiment_launcher_op(
            experiment_name=name,
            experiment_namespace=namespace,
            parallel_trial_count=parallelTrialCount,
            max_trial_count=maxTrialCount,
            objective=str(objectiveConfig),
            algorithm=str(algorithmConfig),
            trial_template=str(trialTemplate),
            parameters=str(parameters),
            experiment_timeout_minutes=experimentTimeoutMinutes,
            delete_finished_experiment=deleteAfterDone)

    op_out = dsl.ContainerOp(
        name="my-out-cop",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo hyperparameter: %s" % op1.output],
    )

if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(mnist_hpo, __file__ + ".tar.gz")
