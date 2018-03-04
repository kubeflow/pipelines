The sample involves two systems

1. A GKE cluster to run argo
2. A dataproc cluster to run the actual steps

The first GKE cluster needs to be set up manually:

  ```gcloud container clusters create [your-gke-cluster-name] --zone us-west1-a --scopes cloud-platform```

The second dataproc cluster is created and shut down automatically during sample run. Which project to run it is supplied in the run.sh as a command line parameter.


The requirements:

* Two clusters should run in the same cloud project. Otherwise, one needs to grant compute service account of first project access to the second project.

* Argo GKE Cluster needs to have cloud-platform scope so it can start dataproc cluster. So "--scopes cloud-platform" is needed in setting up the argo GKE cluster.

* Your project should also have Dataproc API enabled.


Current run.sh includes hard-coded parameters:

* project=bradley-playground --- this needs to be replaced with your project that runs argo GKE cluster.
* output, train, eval, schema, package, conf --- You can copy the files to your own project to make sure the steps can access them, and then replace the paths.
* cluster=mycluster --- just pick a unique name. This is the dataproc cluster name to use in running the sample.



