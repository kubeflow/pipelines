# MetadataStore Upgrade Tool

**Note: There are two available options to upgrade Metadata store in the cluster**

1. upgrade whole KFP system (via CLI guide/doc) that incldues Metadata store upgrade.
2. Use this tool to upgrade only the Metadata store in the cluster.


Upgrade tool provides a mechanism for KFP users to upgrade MetadataStore component in their KFP cluster. A MetadataStore upgrade is composed of two aspects:
* Upgrading the image used in `metadata-grpc-deployment` [K8s Deployment](https://github.com/kubeflow/pipelines/blob/master/manifests/kustomize/base/metadata/metadata-grpc-deployment.yaml). The user is expected to provide an image-tag for this upgrade.
* Upgrading the MYSQL database schema to adhere to MLMD library used in the `metadata-grpc-deployment` image. The tool automatically handles schema upgrade by setting `--enable_database_upgrade=true` [flag](https://github.com/kubeflow/pipelines/blob/master/manifests/kustomize/base/metadata/metadata-grpc-deployment.yaml) while invoking the metadata gRPC server.

**Note: This upgrade tool should be the only client interacting with the MetadataStore during upgrade**

The contract for this tool was published and shared with Kubeflow Pipelines community in this [doc](https://docs.google.com/document/d/1gF-mx3lMyU9h7MAAOXP-KGV-BF-UabDsAlFrWNNhKBo/edit?usp=sharing)

To run the tool execute the following command from this folder:

```
go run main.go --new_image_tag=<image-tag> --kubeconfig=<kubeconfig-path> --namespace=<namespace-name>
```

Arguments:
* `--new_image_tag`(Required) - The image tag for the gRPC server version to upgrade to. The list of available images can be found [here](gcr.io/tfx-oss-public/ml_metadata_store_server)
* `--kubeconfig`(Optional) - Absolute path to a kubeconfig file. If this argument is not specified `.kubecofing` in user's home directory is used.
* `--namespace`(Optional) - Namespace where `metadata-deployment` is deployed in the KFP cluster. Defaults to `kubeflow`.

**Note:** 
1. Upgrade is supported from version [ml-metadata v0.21.0](https://github.com/google/ml-metadata/releases/tag/v0.21.0) onwards.
2. The ML Metadata Store Server image version used in the `metadata-grpc-deployment` deployment of a KFP cluster can be found  in the `Active revisions` section of the deployment details page. 

## Execution Flow

The tool using the K8's [client-go](https://github.com/kubernetes/client-go) library performs upgrade in following steps:

1. Queries the KFP cluster to get the `metadata-grpc-deployment` K8 Deployment resource.
2. Updates the deployment's Spec Image value using the image tag provided as argument and adds `--enable_database_upgrade=true` to the deployment's container arguments.
3. Uses  [client-go's](https://github.com/kubernetes/client-go) `RetryOnConflict` API to update the Deployment.
4. If the update is successful, `metadata-grpc-deployment` deployment is updated again to remove the `--enable_database_upgrade=true` argument. If this update fails, the tool logs the failure message to `stdout` with error details.
5. If update in Step-3 fails, the tool errors out by logging the failure message to `stdout` with error details. In this state, the user is expected to fix the MetadataStore deployment in the cluster.

