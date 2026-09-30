# Run and Recurring Run

A *run* is a single execution of a pipeline. Runs comprise an immutable log of
all experiments that you attempt, and are designed to be self-contained to allow
for reproducibility. You can track the progress of a run by looking at its
details page on the Kubeflow Pipelines UI, where you can see the runtime graph,
output artifacts, and logs for each step in the run.

<a id=recurring-run></a>
A *recurring run*, or job in the Kubeflow Pipelines [backend APIs](https://github.com/kubeflow/pipelines/tree/06e4dc660498ce10793d566ca50b8d0425b39981/backend/api/go_http_client/job_client), is a repeatable run of
a pipeline. The configuration for a recurring run includes a copy of a pipeline
with all parameter values specified and a 
[run trigger](run-trigger.md).
You can start a recurring run inside any experiment, and it will periodically
start a new copy of the run configuration. You can enable/disable the recurring
run from the Kubeflow Pipelines UI. You can also specify the maximum number of
concurrent runs, to limit the number of runs launched in parallel. This can be
helpful if the pipeline is expected to run for a long period of time and is
triggered to run frequently.

## Recurring run tags

Recurring runs support the same key-value tags as pipelines and pipeline versions.
Use tags to organize schedules by team, environment, or other metadata and filter
recurring runs through the API or Python SDK. Each recurring run can have up to
20 tags. Keys must be nonempty and cannot contain a period (`.`); keys and values
can each contain up to 63 Unicode characters.

```python
import json
import kfp

client = kfp.Client()
recurring_run = client.create_recurring_run(
    experiment_id=experiment_id,
    job_name="nightly",
    pipeline_id=pipeline_id,
    version_id=version_id,
    interval_second=86400,
    tags={"team": "ml", "environment": "production"},
)

# Replace the complete tag map.
client.update_recurring_run_tags(recurring_run.recurring_run_id, {"team": "platform"})

# Tag predicates use EQUALS and can be combined with other list filters.
matching_runs = client.list_recurring_runs(
    experiment_id=experiment_id,
    filter=json.dumps({"predicates": [
        {"key": "tags.team", "operation": "EQUALS", "string_value": "platform"}
    ]}),
)

# Remove all tags.
client.update_recurring_run_tags(recurring_run.recurring_run_id, {})
```

Tags belong to the recurring run. They are not inherited from its pipeline or
pipeline version, and are not copied to the individual runs it creates. Enabling,
disabling, and scheduler updates preserve tags.

For REST clients, use `PATCH /apis/v2beta1/recurringruns/{recurring_run_id}` with
`{"tags": {"team": "ml"}}` to replace tags or `{"tags": {}}` to clear them. Omitting
`tags` leaves them unchanged. gRPC clients should set `update_mask` to `tags` when
clearing tags, since protobuf maps do not distinguish an empty map from an absent
one. An explicit `update_mask=tags` also works for REST clients. Other update-mask
paths are not supported.

In multi-user deployments, updating tags requires the `update` verb on the
`jobs` resource in `pipelines.kubeflow.org`. The Pipelines edit role includes this
permission; existing deployments must apply the updated role manifest.

## Next steps

* Read an [overview of Kubeflow Pipelines](../overview.md).
* Follow the [pipelines quickstart guide](../getting-started.md) 
  to deploy Kubeflow and run a sample pipeline directly from the Kubeflow 
  Pipelines UI.
