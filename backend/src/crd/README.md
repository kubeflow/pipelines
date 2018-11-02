## Scheduled Workflow CRD/controller

### How to generate the API client code from the API specification? 

Get the dependencies:

```
go get -u ./...
go get -u k8s.io/client-go/...
go get -u k8s.io/code-generator/...
```

Generate the API client code from the API specification:

```
./hack/update-codegen.sh
```

### How to run the ScheduledWorkflow controller from the command line? 

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
API Version:  kubeflow.org/v1alpha1
Kind:         ScheduledWorkflow
Metadata:
  Cluster Name:        
  Creation Timestamp:  2018-06-06T01:24:55Z
  Generation:          0
  Initializers:        <nil>
  Resource Version:    3056202
  Self Link:           /apis/kubeflow.org/v1alpha1/namespaces/default/scheduledworkflows/every-minute-cron
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
### 