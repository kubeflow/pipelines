# Kubeflow pipeline components

Kubeflow pipeline components are implementations of Kubeflow pipeline tasks. Each task takes
one or more [artifacts](https://github.com/kubeflow/pipelines/wiki/Concepts#step-output-artifacts)
as input and may produce one or more
[artifacts](https://github.com/kubeflow/pipelines/wiki/Concepts#step-output-artifacts) as output.


**Example: XGBoost DataProc components**
* [Set up cluster](dataproc/xgboost/create_cluster.py)
* [Analyze](dataproc/xgboost/analyze.py)
* [Transform](dataproc/xgboost/transform.py)
* [Distributed train](dataproc/xgboost/train.py)
* [Delete cluster](dataproc/xgboost/delete_cluster.py)

Each task usually includes two parts:

``Client code``
  The code that talks to endpoints to submit jobs. For example, code to talk to Google
  Dataproc API to submit a Spark job.

``Runtime code``
  The code that does the actual job and usually runs in the cluster. For example, Spark code
  that transforms raw data into preprocessed data.

``Container``
  A container image that runs the client code.

Note the naming convention for client code and runtime code&mdash;for a task named "mytask":

* The `mytask.py` program contains the client code.
* The `mytask` directory contains all the runtime code.

See [how to build your own components](https://github.com/kubeflow/pipelines/wiki/Build-Your-Own-Component)
