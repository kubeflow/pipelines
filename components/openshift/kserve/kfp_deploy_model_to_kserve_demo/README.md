# Using Data Science Pipelines to deploy a model to KServe in OpenShift AI

This example is based on https://github.com/kubeflow/pipelines/tree/b4ecbabbba1ac3c7cf0e762a48e9b8fcde239911/components/kserve.

In a cluster with the following operators installed:

* Red Hat OpenShift AI
  * Create a `DataScienceCluster` instance
* Red Hat Authorino
* Red Hat OpenShift Service Mesh
* Red Hat OpenShift Serverless

1. Set a namespace and deploy the manifests:

    ```shell
    export NAMESPACE=<your-namespace>
    kustomize build manifests | envsubst | oc apply -f -
    ```

2. Install the required Python dependencies

    ```shell
    pip install -r requirements.txt
    ```
      
3. Compile the pipeline

    ```shell
    kfp dsl compile --py pipeline.py --output pipeline.yaml
    ```

4. Deploy the compiled pipeline (`pipeline.yaml`) in the Red Hat OpenShift AI console
5. Run the pipeline in the Red Hat OpenShift AI console
6. When the pipeline completes you should be able to see the `example-precictor` pod and the `InferenceService`

    ```shell
    oc get pods | grep 'example-predictor'
    example-predictor-00001-deployment-7c5bf67574-p6rrs               2/2     Running     0          8m18s
    ```
   
    ```shell
    oc get inferenceservice
    NAME      URL                                   READY   PREV   LATEST   PREVROLLEDOUTREVISION   LATESTREADYREVISION       AGE
    example   https://something.openshiftapps.com   True           100                              example-predictor-00001   12m
    ```