# ML Metadata

> **Note:** [KFP 2.15](https://github.com/kubeflow/pipelines/releases/tag/2.15.0) includes a major upgrade to the underlying Gorm backend, necessitating an automated database index migration for users upgrading from versions prior to 2.15.0 (the migration logic can be reviewed [here](https://github.com/kubeflow/pipelines/blob/release-2.15/backend/src/apiserver/client_manager/client_manager.go#L367)). Given that this migration does not support rollback functionality, it is strongly advised that production databases be backed up before initiating the upgrade process.

**Note:** Kubeflow Pipelines has moved from using [kubeflow/metadata](https://github.com/kubeflow/metadata)
to using [google/ml-metadata](https://github.com/google/ml-metadata) for Metadata dependency.

Kubeflow Pipelines backend stores runtime information of a pipeline run in Metadata store.
Runtime information includes the status of a task, availability of artifacts, custom properties associated
with Execution or Artifact, etc. Learn more at [ML Metadata Get Started](https://github.com/google/ml-metadata/tree/master).

You can view the connection between Artifacts and Executions across Pipeline Runs, if
one Artifact is being used by multiple Executions in different Runs. This connection visualization
is called a *Lineage Graph*.

## Next steps

* Learn about [output Artifact](output-artifact.md).
