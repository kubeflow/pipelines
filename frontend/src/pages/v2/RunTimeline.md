# Run Timeline

The **Timeline** tab charts component task executions in chronological order, including individual loop iterations. It reuses the run-details task query for completed and in-progress runs.

Click anywhere on a row to inspect task status, timestamps, elapsed time, and state history. Component-name and bar buttons also support keyboard selection. The details panel links back to the corresponding task in the graph.

## Timing semantics

- Task elapsed time runs from `create_time` to `end_time`. It includes startup, waiting, and retries—not just user-code execution.
- Active tasks in an unfinished run grow with wall time. Completed tasks remain fixed. Missing, invalid, or reversed timestamps do not become inferred durations.
- Cached task spans describe cache-resolution overhead unless the task also has earlier attempts. Enclosing DAG/loop rows are excluded to avoid double-counting.
- Retry indicators use failure/recovery history, repeated starts, or distinct executor pod identities. A second `RUNNING` entry is not required, and a normal driver/executor pair is not considered a retry.
- Components are ordered by creation time, with missing creation times last and ties resolved by task identity.
- Task selection survives query refreshes. The view does not add a second API poller.

This view does not separate driver/executor pods, pending versus executing phases, or individual retry-attempt durations. It relies only on existing task API data.

On wide layouts, the Timeline owns its scroll area and the selected-task inspector stays visible alongside the chart. Details are limited to the available height and scroll internally when needed. Selecting another task resets the details scroll position; refreshing the same task preserves it. Narrow layouts keep the inspector below the chart in normal document flow. Column headers scroll with the chart.

## Development

Use the [frontend development setup](../../../README.md#local-development) with a KFP backend, then open a run's **Timeline** tab. New links use `?tab=timeline`; existing `?tab=waterfall` links remain supported. The existing ten-second run/task pollers discover state changes and newly created tasks; active bars update every second.
