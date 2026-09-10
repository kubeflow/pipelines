# Version Compatibility

The following table presents a comprehensive overview of the version compatibility between the Kubeflow Pipelines (KFP) Runtime and the KFP SDK.

| KFP Runtime | KFP SDK | Notes |
|---|---|---|
| v2.0.* | v2.0.* | Active development. Support for certain features may be staged between the Runtime and the SDK. |
| v2.0.* | v1.8.* | Backward compatibility maintained for v1 features. No v2 features support. |
| v1.8.* | v1.8.* | Maintenance mode. Full compatibility for v1 features. No v2 features support. |
| v1.7.* | * | Not recommended due to the age of the release. |
| * | v1.7.* | Not recommended due to the age of the release. |

## Notes

* **v1 features** refer to the features available when running v1 pipelines--these are pipelines produced by v1 versions of the KFP SDK (excluding the v2 compiler available in KFP SDK v1.8), they are persisted as Argo workflow in YAML format.

* **v2 features** refer to the features available when running v2 pipelines--these are pipelines produced using v2 versions of the KFP SDK, they are persisted as Intermediate Representation (IR) in YAML format.

* Pipelines produced using the v2 namespace (`kfp.v2`) in the SDK v1.8 were partially and momentarily supported by the KFP Runtime v1.8 via *v2-compatible* mode. The support for v2-compatible mode has been discontinued.

* The KFP Runtime v2.0.* supports v2 features and v1 features at the same time. Depending on whether users are running v1 pipelines vs. v2 pipelines, the KFP Runtime behaves differently. The most noticeable difference users can perceive is v1 pipelines are rendered using the old v1-style DAG UI, while v2 pipelines are rendered in the v2 modern DAG UI.

Please note that while we aim to sure backward compatibility when possible, it is always recommended to use the latest version of both KFP Runtime and KFP SDK to take advantage of the full range of features and improvements.

For more detailed information on feature support, please refer to the version-specific user documentation:

* [Kubeflow Pipelines v1][kfp-v1-doc]
* [Kubeflow Pipelines v2][kfp-v2-doc]

[kfp-v1-doc]: https://www.kubeflow.org/docs/components/pipelines/legacy-v1
[kfp-v2-doc]: ../index.md
