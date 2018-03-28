Samples in this directory involve two systems

1. A GKE cluster to run [argo](https://github.com/argoproj/argo)
2. A dataproc cluster to run the actual steps

The first GKE cluster needs to be set up manually:

  ```gcloud container clusters create [your-gke-cluster-name] --zone us-west1-a --scopes cloud-platform```

The second dataproc cluster is created and shut down automatically during sample run. Which project to run it is a parameter.


The requirements:

* Two clusters should run in the same cloud project. Otherwise, one needs to grant compute service account of first project access to the second project.

* Argo GKE Cluster needs to have cloud-platform scope so it can start dataproc cluster. So "--scopes cloud-platform" is needed in setting up the argo GKE cluster.

* Your project should also have Dataproc API enabled.


To run a classification training pipeline sample with SFPD data:

argo submit xgboost-training-roc.yaml \
     -p project=MY_GCP_PROJECT \
     -p output="gs://my-bucket/sfpdmodel" \
     -p train="gs://ml-pipeline-playground/sfpd/train.csv" \
     -p eval="gs://ml-pipeline-playground/sfpd/eval.csv" \
     -p schema="gs://ml-pipeline-playground/sfpd/schema.json" \
     -p target=resolution \
     -p trueclass=ACTION \
     -p workers=2 \
     -p rounds=100 \
     -p conf=gs://ml-pipeline-playground/trainconfcla.json \
     --entrypoint xgboost-training

To run a classification training pipeline sample with 20NewsGroups data:

argo submit xgboost-training-cm.yaml \
     -p project=MY_GCP_PROJECT \
     -p output="gs://my-bucket/newsmodel" \
     -p train="gs://ml-pipeline-playground/newsgroup/train.csv" \
     -p eval="gs://ml-pipeline-playground/newsgroup/eval.csv" \
     -p schema="gs://ml-pipeline-playground/newsgroup/schema.json" \
     -p target=news_label \
     -p workers=2 \
     -p rounds=200 \
     -p conf=gs://ml-pipeline-playground/trainconfcla.json \
     --entrypoint xgboost-training


To run an evaluation pipeline sample with 20NewsGroups model & testdata:

argo submit xgboost-evaluation.yaml \
     -p project=MY_GCP_PROJECT \
     -p output="gs://my-bucket/newsmodel" \
     -p eval="gs://ml-pipeline-playground/newsgroup/eval.csv" \
     -p model="gs://ml-pipeline-playground/newsgroup/model/model" \
     -p target=news_label \
     -p trueclass=talk.politics.mideast \
     -p analysis=gs://ml-pipeline-playground/newsgroup/analysis/ \
     --entrypoint xgboost-evaluation



