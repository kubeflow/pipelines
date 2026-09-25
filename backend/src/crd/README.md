## CRD controller
This directory contains code for custom Kubernetes CRDs and controllers used by
the Kubeflow pipelines system. Currently there are 2 such systems:

* ScheduledWorkflow
* Viewer

The following are guidelines on developing and running these controllers.

### Prerequisites

First, we need to generate the API client code from the API specification.

Get the dependencies:

```
dep ensure
go get -u ./...
```

Generate the API client code from the API specification:

```
./hack/update-codegen.sh
```

### Running ScheduledWorkflow controller from the command line.

The following assumes that your Kubernetes configuration file is located at '$HOME/.kube/config'.

To create the resource for the CRD, execute:

```
kubectl create -f ./install/manifests/scheduledworkflow-crd.yaml
```

Output:

```
customresourcedefinition.apiextensions.k8s.io "scheduledworkflows.kubeflow.org" created
```

To run the controller locally, execute:
configuration file

```
go run ./controller/scheduledworkflow/*.go -kubeconfig=$HOME/.kube/config -alsologtostderr=true
```

Output:

```
Starting workers
Started workers
Wait for shut down
```

Create recurring runs using the v2beta1 RecurringRunService or the KFP SDK.
The API server accepts pipeline IR or pipeline-version references and creates
ScheduledWorkflow resources. Embedded Argo templates are execution details of
IR compilation; submitting raw user-authored Workflow templates is not supported.

Inspect the schedules with `kubectl get swf` and `kubectl describe swf <name>`.
Kubernetes CRD API versions such as `kubeflow.org/v1beta1` are independent of the
removed KFP v1 API and remain required by the controllers.

### Running Viewer controller from the command line.

The following assumes that your Kubernetes configuration file is located at '$HOME/.kube/config'.

To create the resource for the CRD, execute:

```
$ kubectl create -f ./install/manifests/viewer-crd.yaml
customresourcedefinition.apiextensions.k8s.io/viewers.kubeflow.org created
```

To run the controller locally, execute:

```
go run ./controller/viewer/ -kubecfg=$HOME/.kube/config -alsologtostderr=true
```

Now, let's create a simple Tensorboard viewer using the supplied sample:
```
$ kubectl create -f samples/viewer/mnist.yaml
viewer.kubeflow.org/viewer-75tkf created

$ kubectl get viewers -n kubeflow
NAME           AGE
viewer-75tkf   108s
```

Verify that the viewer created a deployment and service to house the Tensorboard instance:
```
$ kubectl -n kubeflow get deployments -l app=viewer
NAME                      READY   UP-TO-DATE   AVAILABLE   AGE
viewer-75tkf-deployment   1/1     1            1           3m31s

$ kubectl -n kubeflow get services -l app=viewer
NAME                   TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE
viewer-75tkf-service   ClusterIP   10.35.249.82   <none>        80/TCP    3m45s
```

There are two ways one can access the running Tensorboard instance, via
[Ambassador](https://www.getambassador.io/), or via direct port forwarding,
as described below.

#### Access via Ambassador
The Tensorboard instance's service has the right annotations to allow it to
be routed via Ambassador. Set up port forwarding to the Ambassador instance
in your cluster:
```
$ kubectl port-forward -n kubeflow \
  $(kubectl get pods -n kubeflow --selector service=ambassador -o jsonpath='{.items[0].metadata.name}')  \
  8000:80
Forwarding from 127.0.0.1:8000 -> 80
Forwarding from [::1]:8000 -> 80
```

Note: The above assumes ambassador is installed under namespace `kubeflow`.

The Tensorboard instance should now be accessible at
http://localhost:8000/tensorboard/viewer-75tkf/. Note that the last path
corresponds to the new viewer name, and the URL must end with the trailing
slash.

#### Access via port-forwarding
You can also access the instance directly by setting up port-forwarding to
the Pod running the Tensorboard instance. Note that the serving path will
still be under `/tensorboard/viewer-75tkf`, similar to the Ambassador routing
scenario above.

To set up port-forwarding to the viewer named `viewer-75ktf`, run the
following command:
```
kubectl port-forward -n kubeflow \
  $(kubectl get pods -n kubeflow --selector=viewer=viewer-t6qst -o jsonpath='{.items[0].metadata.name}') \
  8000:6006
```

Notice that we port-forward the remote port 6006, which is the port that
Tensorboard viewers are started on.

The Tensorboard instance should now be accessible at
http://localhost:8000/tensorboard/viewer-75tkf/.
