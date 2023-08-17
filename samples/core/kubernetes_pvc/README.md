## Read/write to a Kubernetes PVC using kfp-kubernetes

This sample uses [kfp-kubernetes](https://pypi.org/project/kfp-kubernetes/) to
demonstrate typical usage of a plugin library. Specifically, we will use
`kfp-kubernetes` to create a [PersistentVolumeClaim
(PVC)](https://kubernetes.io/docs/concepts/storage/persistent-volumes/), use the
PVC to pass data between tasks, and delete the PVC after using it.

See the [kfp-kubernetes documentation](https://kfp-kubernetes.readthedocs.io/)
and [Kubeflow Pipeline
documentation](https://www.kubeflow.org/docs/components/pipelines/v2/platform-specific-features/#example-readwrite-to-a-kubernetes-pvc-using-kfp-kubernetes)
for more information.
