# Run Timeline (2.18)

The **Timeline** tab charts component executions in chronological order, including individual loop iterations and importers. This release-specific adaptation reuses the existing run-context **MLMD query**; it does not require the native task APIs or the backend migration used on master.

Click a row, or use its component-name/bar button with the keyboard, to inspect the latest state and metadata timestamps. **Open task in graph** resolves parent execution IDs to the corresponding DAG/iteration and checks the exact execution identity before opening its details. Missing ancestry disables navigation; if the graph represents a different same-name execution, Timeline stays open with a warning rather than selecting the wrong task.

## Timing semantics and limitations

- Elapsed time is **approximate**: MLMD execution creation to last metadata update for a terminal execution, or to the current wall time for a running execution in an unfinished run. The inspector labels the endpoint **Last updated**, not Finished. Later metadata edits can change a terminal execution's displayed interval.
- These are not exact component start/finish or CPU timings. They may include waiting, metadata writes, and retries. Cached executions are not component computation time and may include earlier attempts.
- Missing, invalid, reversed, and zero-sentinel timestamps do not produce inferred durations. Skipped executions have no duration. Stale RUNNING executions stop growing when the run becomes terminal and remain untimed until terminal metadata arrives.
- MLMD exposes only the latest execution state and pod identity. State-transition history, reliable retry detection, and per-attempt durations are unavailable. No retry count or history is inferred from names or metadata timestamps. Separate execution records remain separate rows, while retries that reuse an execution share its interval.
- MLMD's `CANCELED` state means “not triggered” in KFP's writer and is displayed as **Skipped**. Tasks that never produced an MLMD execution are absent.
- Root, DAG, condition, and loop-wrapper executions are excluded. When a response lacks its execution type name, only leaves proven by the pipeline spec and parent ancestry are included.
- Creation-time ordering uses execution ID as a stable tie-breaker. Selection survives MLMD refreshes and refresh failures retain the last successful snapshot. No second API poller is added.

On wide layouts, Timeline owns its scroll area and the selected-task inspector stays visible alongside the chart. Long details scroll internally; changing selection resets the inspector scroll while refreshing the same task preserves it. Narrow layouts put the inspector below the chart. Column headers scroll with the chart.

## Development

Use the [frontend development setup](../../../README.md#local-development) with a 2.18 KFP backend, then open a run's **Timeline** tab. New links use `?tab=timeline`; `?tab=waterfall` is also accepted. Existing ten-second polling discovers new MLMD snapshots; active bars update every second.
