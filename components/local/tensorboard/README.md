# Pipeline Tensorboard Local Component

For now, the output artifact cannot take inputs from the local file-system or Minio.
This is crucial for on-prem / local setup.

This is a workaround step until the features for the Output Artifact Viewer is finished.

```python
# Define the tensorboard step op
def tensorboard_op(tensorboard_dir, step_name='tensorboard'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'gcr.io/ml-pipeline/ml-pipeline-local-tensorboard:latest',
        arguments = [
            '--logdir', tensorboard_dir,
        ]
    )

# Define the volume, pvc, volume mount to be used for mounting TFEvents
tensorboard_pvc = k8s_client.V1PersistentVolumeClaimVolumeSource(claim_name='tensorboard-pvc')
tensorboard_volume = k8s_client.V1Volume(name='tensorboard', persistent_volume_claim=tensorboard_pvc)
tensorboard_volume_mount = k8s_client.V1VolumeMount(mount_path='/mnt/tensorboard/', name='tensorboard')

# Ensure tensorboard summaries are stored to this location from the training step
# Or modify to match location of stored summaries
tensorboard_dir = '/mnt/tensorboard/{{.workflow.name}}'

# Create tensorboard step and attach volumes, volume mount
tensorboard = bolts_tensorboard_op(tensorboard_dir)
tensorboard.add_volume(tensorboard_volume)
tensorboard.add_volume_mount(tensorboard_volume_mount)
```

`Port-forward to the 6006 port of the Tensorboard pod`
