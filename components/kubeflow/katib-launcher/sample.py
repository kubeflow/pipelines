import json
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
    op1 = katib_experiment_launcher_op(
            name,
            namespace,
            parallelTrialCount=parallelTrialCount,
            maxTrialCount=maxTrialCount,
            objectiveConfig=str(objectiveConfig),
            algorithmConfig=str(algorithmConfig),
            trialTemplate=str(trialTemplate),
            parameters=str(parameters),
            experimentTimeoutMinutes=experimentTimeoutMinutes,
            deleteAfterDone=deleteAfterDone
    )

    op_out = dsl.ContainerOp(
        name="my-out-cop",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo hyperparameter: %s" % op1.output],
    )


def katib_experiment_launcher_op(
      name,
      namespace,
      maxTrialCount=100,
      parallelTrialCount=3,
      maxFailedTrialCount=3,
      objectiveConfig='{}',
      algorithmConfig='{}',
      metricsCollector='{}',
      trialTemplate='{}',
      parameters='[]',
      experimentTimeoutMinutes=60,
      deleteAfterDone=True,
      outputFile='/output.txt'):
    return dsl.ContainerOp(
        name = "mnist-hpo",
        image = 'liuhougangxa/katib-experiment-launcher:latest',
        arguments = [
            '--name', name,
            '--namespace', namespace,
            '--maxTrialCount', maxTrialCount,
            '--maxFailedTrialCount', maxFailedTrialCount,
            '--parallelTrialCount', parallelTrialCount,
            '--objectiveConfig', objectiveConfig,
            '--algorithmConfig', algorithmConfig,
            '--metricsCollector', metricsCollector,
            '--trialTemplate', trialTemplate,
            '--parameters', parameters,
            '--outputFile', outputFile,
            '--deleteAfterDone', deleteAfterDone,
            '--experimentTimeoutMinutes', experimentTimeoutMinutes,
        ],
        file_outputs = {'bestHyperParameter': outputFile}
    )

if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(mnist_hpo, __file__ + ".tar.gz")
