# ML Pipeline Components

ML Pipeline Components are implementation of ML Pipeline tasks. Each task takes
one or more [artifacts](../artifacts) as input and may produce one or more
[artifacts](../artifacts).


## XGBoost DataProc Components
* [Setup Cluster](dataproc/xgboost/create_cluster.py)
* [Analyze](dataproc/xgboost/analyze.py)
* [Transform](dataproc/xgboost/transform.py)
* [Distributed Train](dataproc/xgboost/train.py)
* [Delete Cluster](dataproc/xgboost/delete_cluster.py)

Each task usually includes two parts:

``Client Code``
  The code that talks to endpoints to submit jobs. For example, code to talk to Google
  Dataproc API to submit Spark job.

``Runtime Code``
  The code that does the actual job and usually run in cluster. For example, Spark code
  that transform raw data into preprocessed data.

``Container``
  A container image that runs the client code.

There is a naming convention to client code and runtime code. For a task named "mytask",
there is mytask.py including client code, and there is also a mytask directory holding
all runtime code.


