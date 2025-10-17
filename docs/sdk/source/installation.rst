.. _open-source-deployment:

Deploying Kubeflow Pipelines 
========

As an alternative to deploying Kubeflow Pipelines (KFP) as part of the
`Kubeflow deployment <https://www.kubeflow.org/docs/started/installing-kubeflow/>`_,
you also have the option to deploy only Kubeflow Pipelines.

Follow the instructions below to deploy Kubeflow Pipelines standalone using the supplied Kustomize manifests.

You should be familiar with the following tools:

- `Kubernetes <https://kubernetes.io/docs/home/>`_
- `kubectl <https://kubernetes.io/docs/reference/kubectl/overview/>`_
- `kustomize <https://kustomize.io/>`_

Deploying Kubeflow Pipelines
----------------------------

1. Deploy Kubeflow Pipelines:

   .. code-block:: bash

      export PIPELINE_VERSION={{% pipelines/latest-version %}}
      kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources?ref=$PIPELINE_VERSION"
      kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io
      kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/env/dev?ref=$PIPELINE_VERSION"

   The Kubeflow Pipelines deployment takes approximately 3 minutes to complete. During this time, it is normal for pods to crash in the `kubeflow` namespace until the deployment completes.

2. Port-forward the Kubeflow Pipelines UI:

   .. code-block:: bash

      kubectl port-forward -n kubeflow svc/ml-pipeline-ui 8080:80

3. Open the following URL in your browser to access the UI:

   `http://localhost:8080 <http://localhost:8080>`_

