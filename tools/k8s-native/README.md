# Export KFP Pipelines to Kubernetes Manifests

This tool exports all **Kubeflow Pipelines (KFP)** and their **PipelineVersions**  from the KFP database and generates Kubernetes-compatible Pipeline and PipelineVersion YAML manifests. These manifests can then be deployed directly to a Kubernetes cluster.

Users may find this script useful after switching from KFP's traditional DB storage for Pipelines and Pipelineversion to [KFP's k8s native method of storage](https://github.com/kubeflow/pipelines/tree/master/manifests/kustomize/env/cert-manager/platform-agnostic-multi-user-k8s-native). After switching, users can use this script to assist with migrating their pre-existing Pipelines and PipelineVersions.

## What It Does

The script performs the following steps:

1. **Connects to the KFP API server** to retrieve all pipelines and their versions.
2. **Converts the retrieved objects** into Kubernetes `Pipeline` and `PipelineVersion` resources.
3. **Generates Kubernetes-compliant names**:
   - Pipeline names are transformed to conform to [Kubernetes DNS-1123](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) naming standards.
   - Pipeline version names are derived by combining the pipeline name and version display name. If the combined name exceeds length limits, it is truncated and a unique index is appended to avoid duplication.
4. **Writes the output YAML files** into a specified directory—one file per pipeline, including all its versions.

## Installation

This script depends on the following Python packages:

- `kfp`
- `pyyaml`
- `requests`

You can install them via pip:

```bash
pip install kfp pyyaml requests
```
## Usage

```bash
python migration.py \
  --kfp-server-host $KFP_SERVER_HOST \
  --token $KFP_BEARER_TOKEN \
  [--ca-bundle $CA_BUNDLE] \
  [--namespace your-namespace] \
  [--batch-size 10] \
  [--output ./exported-pipelines] \
  [--no-pipeline-name-prefix]
```

## Arguments

- `--kfp-server-host`: Host of the KFP Endpoint (e.g., `https://<your-kfp-server>`).
- `--token`: Bearer token for authentication to access the KFP API.
- `--ca-bundle` : Path to the CA bundle for TLS verification (if required).
- `--namespace` : The Kubernetes namespace from which the pipelines should be exported (typically used in multi-user deployments).
- `--batch-size` : Number of pipelines to process in a single batch (default is 100). 
- `--output`  : Directory path where the generated YAML files will be written (default is `/kfp-exported-pipelines`).
- `--no-pipeline-name-prefix` : Disable prefixing pipeline names to version names for uniqueness (useful for cleaner naming).

Note: In multi-user mode, pipelines are isolated per namespace. To export pipelines from all user namespaces, you need to run this script separately for each namespace by specifying the --namespace flag.

## Useful Tips

### How to Generate the Token in Multi-User Deployment

To generate a token for accessing the KFP API in a multi-user deployment, follow these steps:

1. **Create a Service Account**  
   
   Define a service account that will be used to authenticate API access:

   ```yaml
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: your-service-account
     namespace: your-namespace
   ```
2. **Assign ClusterRole with Required Permissions**
   
   Bind the service account to a ClusterRole with permissions to list and get all pipelines:

   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRole
   metadata:
     name: pipeline-viewer
   rules:
     - apiGroups: ["pipelines.kubeflow.org"]
       resources: ["pipelines"]
       verbs: ["list", "get"]
   ``` 
   Then bind the role to the service account:

   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRoleBinding
   metadata:
     name: pipeline-viewer-binding
   subjects:
     - kind: ServiceAccount
       name: your-service-account
       namespace: your-namespace
   roleRef:
     kind: ClusterRole
     name: pipeline-viewer
     apiGroup: rbac.authorization.k8s.io
   ```
3. **Mount the ServiceAccount Token in Your Pod**
   
   In your Pod manifest, set serviceAccountName to the one you created:

   ```yaml
   apiVersion: v1
   kind: Pod
   metadata:
     name: access-kfp-example
   spec:
     containers:
     - image: hello-world:latest
       name: hello-world
       env:
         - ## this environment variable is automatically read by `kfp.Client()`
           ## this is the default value, but we show it here for clarity
           name: KF_PIPELINES_SA_TOKEN_PATH
           value: /var/run/secrets/kubeflow/pipelines/token
       volumeMounts:
         - mountPath: /var/run/secrets/kubeflow/pipelines
           name: volume-kf-pipeline-token
           readOnly: true
     volumes:
       - name: volume-kf-pipeline-token
         projected:
           sources:
             - serviceAccountToken:
                 path: token
                 expirationSeconds: 7200
                 ## defined by the `TOKEN_REVIEW_AUDIENCE` environment variable on the `ml-pipeline` deployment
                 audience: pipelines.kubeflow.org      
    ```
4. **Extract the Service Account Token**

   Run the following command to retrieve the service account token from the pod and save it locally. This token can then be used to authenticate with the KFP REST API:

    ```bash
    kubectl exec -n <your-namespace> <your-pod-name> -- cat /var/run/secrets/kubeflow/pipelines/token > ./sa-token.txt
    ```
Note: Refer to [Kubeflow documentation](https://www.kubeflow.org/docs/components/pipelines/user-guides/core-functions/connect-api/#serviceaccount-token-volume) for more details.