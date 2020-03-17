# MetadataStore Upgrade Tool

As per this [published contract](https://docs.google.com/document/d/1gF-mx3lMyU9h7MAAOXP-KGV-BF-UabDsAlFrWNNhKBo/edit?usp=sharing) shared with the Kubeflow Pipelines community this tool provides Metadata Store upgrade support in KFP clusters.

To run the tool execute the following command from this folder:

```
go run main.go --image_tag=<image-tag> --kubeconfig=<kubeconfig-path> --namespace=<namespace-name>

 -- image_tag(Requried) - Image tag of a published MetadataStore image in the gcr repository - gcr.io/tfx-oss-public/ml_metadata_store_server
 -- kubeconfig(Optional) - Absolute path to configured kubeconfig file. If not specified .kubecofing at users home directory is useds
 -- namespace(Optional) - Namespace where MetadataStore deployment is deployed. Defaults to 'kubeflow' namespace
```
