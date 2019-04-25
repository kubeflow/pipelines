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

To run a sample workflow on a schedule, execute:

```
kubectl create -f ./samples/scheduledworkflow/every-minute-cron.yaml
```

Output:

```
scheduledworkflow.kubeflow.org "every-minute-cron" created
```

To see the current list of ScheduledWorkflows, execute:

```
kubectl get swf
```

Output:

```
NAME                AGE
every-minute-cron   1m
```

To see the current status of the ScheduledWorklfow named 'every-minute-cron', execute:

```
kubectl describe swf every-minute-cron
```

Output:

```
Name:         every-minute-cron
Namespace:    default
Labels:       scheduledworkflows.kubeflow.org/enabled=true
              scheduledworkflows.kubeflow.org/status=Enabled
Annotations:  <none>
API Version:  kubeflow.org/v1beta1
Kind:         ScheduledWorkflow
Metadata:
  Cluster Name:
  Creation Timestamp:  2018-06-06T01:24:55Z
  Generation:          0
  Initializers:        <nil>
  Resource Version:    3056202
  Self Link:           /apis/kubeflow.org/v1beta1/namespaces/default/scheduledworkflows/every-minute-cron
  UID:                 6b11874e-6928-11e8-9fd5-42010a8a0021
Spec:
  Enabled:      true
  Max History:  10
  Trigger:
    Cron Schedule:
      Cron:  1 * * * * *
  Workflow:
    Spec:
      Arguments:
        Parameters:
          Name:    message
          Value:   hello world
      Entrypoint:  whalesay
      Templates:
        Container:
          Args:
            {{inputs.parameters.message}}
          Command:
            cowsay
          Image:  docker/whalesay
          Name:
          Resources:
        Inputs:
          Parameters:
            Name:  message
        Metadata:
        Name:  whalesay
        Outputs:
Status:
  Conditions:
    Last Heartbeat Time:   2018-06-06T01:41:40Z
    Last Transition Time:  2018-06-06T01:41:40Z
    Message:               The schedule is enabled.
    Reason:                Enabled
    Status:                True
    Type:                  Enabled
  Trigger:
    Last Index:           17
    Last Triggered Time:  2018-06-06T01:41:01Z
    Next Triggered Time:  2018-06-06T01:42:01Z
  Workflow History:
    Completed:
      Phase:         Succeeded
      Created At:    2018-06-06T01:41:10Z
      Finished At:   2018-06-06T01:41:13Z
      Index:         17
      Name:          every-minute-cron-17-2173648469
      Namespace:     default
      Scheduled At:  2018-06-06T01:41:01Z
      Self Link:     /apis/argoproj.io/v1alpha1/namespaces/default/workflows/every-minute-cron-17-2173648469
      Started At:    2018-06-06T01:41:10Z
      UID:           b0b63a82-692a-11e8-9fd5-42010a8a0021
      [...]
```

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
