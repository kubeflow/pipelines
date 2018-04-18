# ML Pipeline User Guideline

This document guides you through the steps to install Machine Learning Pipelines and run your first pipeline sample. 

## Requirements
- Install Kubernetes v1.8 or later
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
 
If you don't want to install [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) and its dependencies(such as VirtualBox), you can use [GKE](https://cloud.google.com/kubernetes-engine/) instead. See [Appendix](#setup-gke) for more information about setting up an GKE cluster.


## Install

To install ML pipeline, just create a bootstrapper job using [bootstrapper.yaml](https://github.com/googleprivate/ml/blob/master/bootstrapper.yaml)
```
TODO(yangpa): Use github path when project is public.  
$ kubectl create -f bootstrapper.yaml
```
And you should see a job created that deploys all ML pipeline components in the cluster.
```
$ kubectl get job
NAME                      DESIRED   SUCCESSFUL   AGE
deploy-ml-pipeline-wjqwt  1         1            9m
```

By default, the ML pipeline will be deployed to the default namespace, with usage collection turned on. 
We use [Spartakus](https://github.com/kubernetes-incubator/spartakus) which does not report any personal identifiable information (PII).

If you want to turn off the usage report, or deploy to a different namespace, you can pass in additional arguments to the deployment job.
See [bootstrapper.yaml](https://github.com/googleprivate/ml/blob/master/bootstrapper.yaml) file for example arguments.

Once deployment is completed, you can check the resource created
```
$  kubectl get all

NAME                         DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
deploy/argo-ui               1         1         1            1           8m
deploy/minio                 1         1         1            1           8m
deploy/ml-pipeline           1         1         1            1           8m
deploy/ml-pipeline-ui        1         1         1            1           8m
deploy/mysql                 1         1         1            1           8m
deploy/spartakus-volunteer   1         1         1            1           8m
deploy/workflow-controller   1         1         1            1           8m

NAME                                      READY     STATUS    RESTARTS   AGE
po/argo-ui-864b4dbc9-wb7rx                1/1       Running   0          8m
po/minio-5bd8bcd4d5-628lf                 1/1       Running   0          8m
po/ml-pipeline-58d679fbd4-6j27x           1/1       Running   0          8m
po/ml-pipeline-ui-64f7bcb9c8-sbgf9        1/1       Running   0          8m
po/mysql-5d496ddbb4-jvztd                 1/1       Running   0          8m
po/spartakus-volunteer-6bc9d9f958-j4z8f   1/1       Running   0          8m
po/workflow-controller-69b6c74c7b-njdlt   1/1       Running   0          8m

NAME                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
svc/argo-ui          NodePort    10.15.247.19    <none>        80:30613/TCP   8m
svc/minio-service    ClusterIP   10.15.253.71    <none>        9000/TCP       8m
svc/ml-pipeline      ClusterIP   10.15.242.187   <none>        8888/TCP       8m
svc/ml-pipeline-ui   ClusterIP   10.15.251.241   <none>        80/TCP         8m
svc/mysql            ClusterIP   10.15.252.169   <none>        3306/TCP       8m
```

By default, the services don't expose any external IP. To visit the ML pipeline UI, forward its port.
```
$ kubectl port-forward $(kubectl get pods -l app=ml-pipeline-ui -o jsonpath='{.items[0].metadata.name}') 8080:3000
```
You can now access the ML pipeline UI at [localhost:8080](http://localhost:8080).

## Uninstall
To uninstall ML pipeline, create a job following the same steps as installation, with additional uninstall argument. 
Check [bootstrapper.yaml](https://github.com/googleprivate/ml/blob/master/bootstrapper.yaml) for details.

## Samples
Checkout [here](https://github.com/googleprivate/ml/blob/master/samples) for a list of samples. 

## Appendix

### Setup Minikube
See the Kubernetes official [doc](https://kubernetes.io/docs/tasks/tools/install-minikube/) to set up your Minikube.

### Setup GKE

Follow the [instruction](https://cloud.google.com/resource-manager/docs/creating-managing-projects) and create a GCP project. 
Once created, enable the GKE API in this [page](https://console.developers.google.com/apis/enabled). You can also find more details about enabling the [billing](https://cloud.google.com/billing/docs/how-to/modify-project?visit_id=1-636559671979777487-508867449&rd=1#enable-billing), as well as activating [GKE API](https://cloud.google.com/kubernetes-engine/docs/quickstart#before-you-begin).

Also, follow the instruction and install google [Cloud SDK](https://cloud.google.com/sdk/) and [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#download-as-part-of-the-google-cloud-sdk) if you haven’t done so yet. Once installed, [authorize](https://cloud.google.com/sdk/gcloud/reference/auth/login) gcloud with your credential, and [set your default project](https://cloud.google.com/sdk/gcloud/reference/config/set).
```
$ gcloud auth login
$ gcloud config set project [your-project-id]
```

The ML pipeline will be running on a GKE cluster. To start a new GKE cluster, first set a default compute zone (us-west1-a in this case):
```
$ gcloud config set compute/zone us-west1-a
```
Then start a GKE cluster. 
```
$ gcloud container clusters create ml-pipeline --scopes cloud-platform
```
Here we choose cloud-platform scope so it can invoke Dataproc cluster. You can find all options for creating a cluster in [here](https://cloud.google.com/sdk/gcloud/reference/container/clusters/create). 

For example following would create a cluster with 3 nodes and n1-standard-2 [machine type](https://cloud.google.com/compute/docs/machine-types), with logging and monitoring enabled.
```
$ gcloud container clusters create ml-pipeline \
  --zone us-west1-a \
  --scopes cloud-platform \
  --enable-cloud-logging \
  --enable-cloud-monitoring \
  --machine-type n1-standard-2 \
  --num-nodes 3
```
Set the newly created GKE cluster as default.
```
$ gcloud config set container/cluster ml-pipeline
```

ML Pipeline uses Argo as its underlying pipeline orchestration system. When installing ML pipeline, a few new cluster roles will be generated for Argo and Pipeline’s API server, in order to allow them creating and retrieving resources such as pods, Argo workflow CRDs etc. 

On GKE with RBAC enabled, you might need to grant your account the ability to create such new cluster roles.

```
$ kubectl create clusterrolebinding BINDING_NAME --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com
```

Now the GKE cluster is ready to use.