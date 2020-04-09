# GPU

## GPU for Kubeflow Pipeline Standalone

```
export GPU_POOL_NAME=p100pool
export CLUSTER_NAME=mpdev-verify

# It stops at "Provisioning" status for almost 1 hour in my test.
gcloud container node-pools create ${GPU_POOL_NAME} \
  --accelerator type=nvidia-tesla-p100,count=1 \
  --zone us-west1-a --cluster ${CLUSTER_NAME} \
  --num-nodes=0 --machine-type=n1-standard-4 --min-nodes=0 --max-nodes=5 --enable-autoscaling
```

## GPU for Kubeflow 