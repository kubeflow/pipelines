# Frontend integration test

This test gets triggered by the end-to-end testing workflows.

## Local run

1. Deploy KFP on a k8s cluster and enable port forwarding. Replace the default namespace `kubeflow` below if needed.

    ```bash
    POD=`kubectl get pods -n kubeflow -l app=ml-pipeline-ui -o jsonpath='{.items[0].metadata.name}'`
    kubectl port-forward -n kubeflow ${POD} 3000:3000 &
    ```

1. Build the container with the tests:

    ```bash
    docker build . -t kfp-frontend-integration-test:local
    ```

1. Run the test with enabled networking (**this exposes your local networking stack to the testing container**):

    ```bash
    docker run --net=host kfp-frontend-integration-test:local --remote-run true
    ```

1. Once completed, you can kill the background process of port-forwarding:

    ```
    ps aux | grep '[k]ubectl port-forward'
    kill <PID>
    ```
