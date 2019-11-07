# Kubeflow pipeline components

Kubeflow pipeline components are implementations of Kubeflow pipeline tasks. Each task takes
one or more [artifacts](https://www.kubeflow.org/docs/pipelines/overview/concepts/output-artifact/)
as input and may produce one or more
[artifacts](https://www.kubeflow.org/docs/pipelines/overview/concepts/output-artifact/) as output.


**Example: XGBoost DataProc components**
* [Set up cluster](deprecated/dataproc/create_cluster/src/create_cluster.py)
* [Analyze](deprecated/dataproc/analyze/src/analyze.py)
* [Transform](deprecated/dataproc/transform/src/transform.py)
* [Distributed train](deprecated/dataproc/train/src/train.py)
* [Delete cluster](deprecated/dataproc/delete_cluster/src/delete_cluster.py)

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

See how to [use the Kubeflow Pipelines SDK](https://www.kubeflow.org/docs/pipelines/sdk/sdk-overview/)
and [build your own components](https://www.kubeflow.org/docs/pipelines/sdk/build-component/).
