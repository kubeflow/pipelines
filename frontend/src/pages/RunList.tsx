/*
 * Copyright 2018 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { NavigationProps } from 'src/lib/Navigation';
import * as React from 'react';
import CustomTable, { Column, Row, CustomRendererProps } from 'src/components/CustomTable';
import { ExperimentInfo } from 'src/lib/ExperimentInfo';
import { V2Run, V2RuntimeState, V2RunStorageState } from 'src/apisv2/run';
import { V2ListExperimentsResponse } from 'src/apisv2/experiment';
import { ResponseError, V2PipelineVersion } from 'src/apisv2/pipeline';
import { Apis, RunSortKeys, ListRequest } from 'src/lib/Apis';
import { Link } from 'react-router';
import { V2Filter, V2PredicateOperation } from 'src/apisv2/filter';
import { RoutePageFactory, QUERY_PARAMS } from 'src/components/Router';
import { URLParser } from 'src/lib/URLParser';
import { formatDateString, logger, errorToMessage, getRunDurationV2 } from 'src/lib/Utils';
import { RunsTable } from 'src/components/tables/RunsTable';
import { RunStatus } from 'src/components/runs/RunStatus';

import { DateListFilter, ColumnFilter, FilterChip } from 'src/components/tables/ColumnFilter';

interface PipelineVersionInfo {
  displayName?: string;
  versionId?: string;
  runId?: string;
  recurringRunId?: string;
  pipelineId?: string;
  usePlaceholder: boolean;
  unavailable?: boolean;
}

interface RecurringRunInfo {
  displayName?: string;
  id?: string;
}

interface DisplayRun {
  experiment?: ExperimentInfo;
  recurringRun?: RecurringRunInfo;
  run: V2Run;
  pipelineVersion?: PipelineVersionInfo;
  error?: string;
}

// Both masks cannot be provided together.
type MaskProps = Exclude<
  { experimentIdMask?: string; namespaceMask?: string },
  { experimentIdMask: string; namespaceMask: string }
>;

export type RunListProps = MaskProps &
  NavigationProps & {
    scopeControl?: React.ReactNode;
    disablePaging?: boolean;
    disableSelection?: boolean;
    disableSorting?: boolean;
    hideExperimentColumn?: boolean;
    noFilterBox?: boolean;
    onError: (message: string, error: Error) => void;
    onLoadSuccess?: () => void;
    onSelectionChange?: (selectedRunIds: string[]) => void;
    runIdListMask?: string[];
    selectedIds?: string[];
    storageState?: V2RunStorageState;
  };

interface RunListState {
  runs: DisplayRun[];
  loadError: boolean;
  statusFilter: string;
}

function _pipelineVersionKey(pipelineId: string, pipelineVersionId: string): string {
  return `${pipelineId}/${pipelineVersionId}`;
}

class RunList extends React.PureComponent<RunListProps, RunListState> {
  private _isMounted = true;
  private _loadGeneration = 0;
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      runs: [],
      loadError: false,
      statusFilter: 'all',
    };
  }

  public render(): React.JSX.Element {
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 1.5,
        label: 'Run',
        sortKey: RunSortKeys.NAME,
      },
      { customRenderer: this._statusCustomRenderer, flex: 0.5, width: 120, label: 'Status' },
      {
        customRenderer: this._pipelineVersionCustomRenderer,
        label: 'Pipeline version',
        flex: 1,
        width: 160,
      },
      { label: 'Duration', flex: 0.5, width: 100 },
      {
        customRenderer: this._recurringRunCustomRenderer,
        label: 'Recurring run',
        flex: 0.5,
        width: 160,
      },
      {
        label: 'Started',
        width: 190,
        flex: 1,
        sortKey: RunSortKeys.CREATED_AT,
        customRenderer: ({ value }) => (
          <span className='kfp-runs-started' title={value}>
            {value}
          </span>
        ),
      },
    ];

    if (!this.props.hideExperimentColumn) {
      columns.splice(3, 0, {
        customRenderer: this._experimentCustomRenderer,
        flex: 1,
        label: 'Experiment',
        width: 160,
      });
    }

    const rows: Row[] = this.state.runs.map((r) => {
      const row = {
        error: r.error,
        id: r.run.run_id!,
        otherFields: [
          r.run!.display_name,
          r.run.state,
          r.pipelineVersion,
          getRunDurationV2(r.run),
          r.recurringRun,
          formatDateString(r.run.created_at),
        ] as any,
      };
      if (!this.props.hideExperimentColumn) {
        row.otherFields.splice(3, 0, r.experiment);
      }
      return row;
    });

    return (
      <div className='kfp-runs-table-view'>
        <DateListFilter
          label='Started'
          field='created_at'
          disabled={!!this.props.runIdListMask || this.props.noFilterBox}
        >
          {(date) => (
            <CustomTable
              filterActions={this.props.scopeControl}
              filterPredicates={[
                ...date.predicates,
                ...(this.state.statusFilter === 'all'
                  ? []
                  : [
                      {
                        key: 'state',
                        operation: V2PredicateOperation.EQUALS,
                        string_value: this.state.statusFilter,
                      },
                    ]),
              ]}
              renderTable={(table) => (
                <RunsTable
                  table={{
                    ...table,
                    emptyMessage:
                      date.active || this.state.statusFilter !== 'all'
                        ? 'No results match the selected filters.'
                        : table.emptyMessage,
                  }}
                  headerActions={{
                    created_at: date.control,
                    Status: !this.props.runIdListMask && !this.props.noFilterBox && (
                      <ColumnFilter
                        label='Status'
                        value={this.state.statusFilter}
                        options={[
                          { value: 'all', label: 'Any status' },
                          ...[
                            'PENDING',
                            'RUNNING',
                            'SUCCEEDED',
                            'FAILED',
                            'CANCELING',
                            'CANCELED',
                            'PAUSED',
                            'SKIPPED',
                          ].map((value) => ({
                            value,
                            label: value.charAt(0) + value.slice(1).toLowerCase(),
                          })),
                        ]}
                        onChange={(statusFilter) => this.setState({ statusFilter })}
                      />
                    ),
                  }}
                  activeFilters={
                    (date.active || this.state.statusFilter !== 'all') && (
                      <>
                        {date.chip}
                        {this.state.statusFilter !== 'all' && (
                          <FilterChip
                            label={`Status: ${this.state.statusFilter.toLowerCase()}`}
                            onClear={() => this.setState({ statusFilter: 'all' })}
                          />
                        )}
                      </>
                    )
                  }
                  reservedRowCount={this.props.runIdListMask?.length}
                  onOpenRun={(id) =>
                    this.props.navigate(RoutePageFactory.runDetails(id), {
                      state: { fromExperimentId: this.props.experimentIdMask },
                    })
                  }
                />
              )}
              columns={columns}
              rows={rows}
              selectedIds={this.props.selectedIds}
              initialSortColumn={RunSortKeys.CREATED_AT}
              ref={this._tableRef}
              filterLabel='Search runs'
              updateSelection={this.props.onSelectionChange}
              reload={this._loadRuns.bind(this)}
              disablePaging={this.props.disablePaging}
              disableSorting={this.props.disableSorting}
              disableSelection={this.props.disableSelection}
              noFilterBox={this.props.noFilterBox}
              errorMessage={
                this.state.loadError
                  ? 'Runs could not be loaded. Use Refresh to try again.'
                  : undefined
              }
              emptyMessage={
                `No` +
                `${
                  this.props.storageState === V2RunStorageState.ARCHIVED
                    ? ' archived'
                    : ' available'
                }` +
                ` runs found` +
                `${
                  this.props.experimentIdMask
                    ? ' for this experiment'
                    : this.props.namespaceMask
                      ? ' for this namespace'
                      : ''
                }.`
              }
            />
          )}
        </DateListFilter>
      </div>
    );
  }

  public componentDidMount(): void {
    this._isMounted = true;
  }

  public componentWillUnmount(): void {
    this._isMounted = false;
    this._loadGeneration++;
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      await this._tableRef.current.reload();
    }
  }

  public _nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    return (
      <div className='kfp-runs-name'>
        <Link
          data-testid='run-name-link'
          data-run-id={props.id}
          data-run-name={props.value || ''}
          title={props.value ? `${props.value} · ${props.id}` : props.id}
          onClick={(event) => event.stopPropagation()}
          to={RoutePageFactory.runDetails(props.id)}
          state={{ fromExperimentId: this.props.experimentIdMask }}
        >
          {props.value || props.id}
        </Link>
      </div>
    );
  };

  public _pipelineVersionCustomRenderer: React.FC<CustomRendererProps<PipelineVersionInfo>> = (
    props: CustomRendererProps<PipelineVersionInfo>,
  ) => {
    if (props.value?.unavailable) {
      return <span title='The pipeline version is no longer available.'>Unavailable</span>;
    }
    // If the getPipeline call failed or a run has no pipeline, we display a placeholder.
    if (!props.value || (!props.value.usePlaceholder && !props.value.pipelineId)) {
      return <div>-</div>;
    }
    const urlParser = new URLParser(this.props);
    const search = props.value.recurringRunId
      ? urlParser.build({ [QUERY_PARAMS.fromRecurringRunId]: props.value.recurringRunId })
      : urlParser.build({ [QUERY_PARAMS.fromRunId]: props.id });
    const url = props.value.usePlaceholder
      ? RoutePageFactory.pipelineDetails('') + search
      : props.value.versionId
        ? RoutePageFactory.pipelineVersionDetails(
            props.value.pipelineId || '',
            props.value.versionId,
          )
        : RoutePageFactory.pipelineDetails(props.value.pipelineId || '');
    if (props.value.usePlaceholder) {
      return (
        <Link className='kfp-runs-resource-link' onClick={(e) => e.stopPropagation()} to={url}>
          Inline definition
        </Link>
      );
    } else {
      return (
        <Link
          className='kfp-runs-resource-link'
          title={props.value.displayName || ''}
          onClick={(event) => event.stopPropagation()}
          to={url}
        >
          {props.value.displayName}
        </Link>
      );
    }
  };

  public _recurringRunCustomRenderer: React.FC<CustomRendererProps<RecurringRunInfo>> = (
    props: CustomRendererProps<RecurringRunInfo>,
  ) => {
    // If the response of listRuns doesn't contain job details, we display a placeholder.
    if (!props.value || !props.value.id) {
      return <div>-</div>;
    }
    const url = RoutePageFactory.recurringRunDetails(props.value.id);
    return (
      <Link className='kfp-runs-resource-link' onClick={(e) => e.stopPropagation()} to={url}>
        {props.value.displayName || '[View config]'}
      </Link>
    );
  };

  public _experimentCustomRenderer: React.FC<CustomRendererProps<ExperimentInfo>> = (
    props: CustomRendererProps<ExperimentInfo>,
  ) => {
    // If the getExperiment call failed or a run has no experiment, we display a placeholder.
    if (!props.value || !props.value.id) {
      return <div>-</div>;
    }
    return (
      <Link
        className='kfp-runs-resource-link'
        onClick={(e) => e.stopPropagation()}
        to={RoutePageFactory.experimentDetails(props.value.id)}
      >
        {props.value.displayName}
      </Link>
    );
  };

  public _statusCustomRenderer: React.FC<CustomRendererProps<V2RuntimeState>> = (
    props: CustomRendererProps<V2RuntimeState>,
  ) => {
    return <RunStatus state={props.value} />;
  };

  protected async _loadRuns(request: ListRequest): Promise<string> {
    const generation = ++this._loadGeneration;
    // Experiment metadata is independent of the run response. Start both reads together.
    const experiments = this._loadExperiments();
    let displayRuns: DisplayRun[];
    let nextPageToken = '';

    if (Array.isArray(this.props.runIdListMask)) {
      displayRuns = this.props.runIdListMask.map((id) => ({ run: { run_id: id } }));
      const filter = JSON.parse(
        decodeURIComponent(request.filter || '{"predicates": []}'),
      ) as V2Filter;
      // listRuns doesn't currently support batching by IDs, so in this case we retrieve and filter
      // each run individually.
      await this._getAndSetRuns(displayRuns);
      const predicates = filter.predicates?.filter(
        (p) => p.key === 'name' && p.operation === V2PredicateOperation.IS_SUBSTRING,
      );
      const substrings = predicates?.map((p) => p.string_value?.toLowerCase() || '') || [];
      displayRuns = displayRuns.filter((runDetail) => {
        for (const sub of substrings) {
          if (!runDetail?.run?.display_name?.toLowerCase().includes(sub)) {
            return false;
          }
        }
        return true;
      });
    } else {
      // Load all runs
      if (this.props.storageState) {
        try {
          // Augment the request filter with the storage state predicate
          const filter = JSON.parse(
            decodeURIComponent(request.filter || '{"predicates": []}'),
          ) as V2Filter;
          filter.predicates = (filter.predicates || []).concat([
            {
              key: 'storage_state',
              // Use EQUALS ARCHIVED or NOT EQUALS ARCHIVED to account for cases where the field
              // is missing, in which case it should be counted as available.
              operation:
                this.props.storageState === V2RunStorageState.ARCHIVED
                  ? V2PredicateOperation.EQUALS
                  : V2PredicateOperation.NOT_EQUALS,
              string_value: V2RunStorageState.ARCHIVED.toString(),
            },
          ]);
          request.filter = encodeURIComponent(JSON.stringify(filter));
        } catch (err) {
          logger.error('Could not parse request filter: ', request.filter);
        }
      }

      try {
        const response = await Apis.runServiceApiV2.listRuns(
          this.props.namespaceMask,
          this.props.experimentIdMask,
          request.pageToken,
          request.pageSize,
          request.sortBy,
          request.filter,
          /* skip_count */ true, // this page never displays the total run count
        );

        displayRuns = (response.runs || []).map((r) => ({ run: r }));
        nextPageToken = response.next_page_token || '';
      } catch (err) {
        const error = new Error(await errorToMessage(err));
        if (this._isMounted && generation === this._loadGeneration) {
          this.setStateSafe({ loadError: true });
          this.props.onError('Error: failed to fetch runs.', error);
        }
        // No point in continuing if we couldn't retrieve any runs.
        return '';
      }
    }

    if (!this._isMounted || generation !== this._loadGeneration) return '';
    await this._setColumns(displayRuns, experiments);

    if (this._isMounted && generation === this._loadGeneration) {
      this.setStateSafe({ runs: displayRuns, loadError: false });
      this.props.onLoadSuccess?.();
    }
    return nextPageToken;
  }

  private setStateSafe(newState: Partial<RunListState>, cb?: () => void): void {
    if (this._isMounted) {
      this.setState(newState as any, cb);
    }
  }

  private async _loadExperiments(): Promise<{
    response?: V2ListExperimentsResponse;
    error?: string;
  }> {
    if (this.props.hideExperimentColumn) return {};
    try {
      const response = !this.props.namespaceMask
        ? await Apis.experimentServiceApiV2.listExperiments()
        : await Apis.experimentServiceApiV2.listExperiments(
            undefined,
            undefined,
            undefined,
            undefined,
            this.props.namespaceMask,
          );
      return { response };
    } catch (error) {
      // Resolve failures here even if the concurrent run request fails or becomes stale.
      return { error: 'Failed to get associated experiment: ' + (await errorToMessage(error)) };
    }
  }

  private async _setColumns(
    displayRuns: DisplayRun[],
    experiments: ReturnType<RunList['_loadExperiments']>,
  ): Promise<DisplayRun[]> {
    const [{ response: experimentsResponse, error: experimentsGetError }, versions] =
      await Promise.all([experiments, this._getReferencedPipelineVersions(displayRuns)]);
    const { pipelineVersionsByKey, errorsByKey } = versions;

    return Promise.all(
      displayRuns.map(async (displayRun) => {
        this._setRecurringRun(displayRun);

        this._setPipelineVersionName(displayRun, pipelineVersionsByKey, errorsByKey);

        if (!this.props.hideExperimentColumn) {
          const experimentId = displayRun.run.experiment_id;

          if (experimentId) {
            const experiment = experimentsResponse?.experiments?.find(
              (e) => e.experiment_id === displayRun.run.experiment_id,
            );
            // If matching experiment id not found (typically because it has been deleted), set display name to "-".
            const displayName = experiment?.display_name || '-';
            if (experimentsGetError) {
              displayRun.error = experimentsGetError;
            } else {
              displayRun.experiment = {
                displayName: displayName,
                id: experimentId,
              };
            }
          }
        }
        return displayRun;
      }),
    );
  }

  private _setRecurringRun(displayRun: DisplayRun): void {
    const recurringRunId = displayRun.run.recurring_run_id;
    // TBD(jlyaoyuli): how to get recurringRun name
    if (recurringRunId) {
      displayRun.recurringRun = { id: recurringRunId };
    }
  }

  /**
   * For each run ID, fetch its corresponding run, and set it in DisplayRuns
   */
  private _getAndSetRuns(displayRuns: DisplayRun[]): Promise<DisplayRun[]> {
    return Promise.all(
      displayRuns.map(async (displayRun) => {
        let getRunResponse: V2Run;
        try {
          getRunResponse = await Apis.runServiceApiV2.getRun(displayRun.run!.run_id!);
          displayRun.run = getRunResponse;
        } catch (err) {
          displayRun.error = await errorToMessage(err);
        }
        return displayRun;
      }),
    );
  }

  /**
   * Fetches each unique (pipeline_id, pipeline_version_id) pair referenced by the given runs
   * exactly once, instead of fetching a pipeline version per run. Most runs on a page share
   * only a handful of distinct pipeline versions, so this turns what used to be one API call
   * per run into one call per unique version.
   */
  private async _getReferencedPipelineVersions(displayRuns: DisplayRun[]): Promise<{
    pipelineVersionsByKey: Map<string, V2PipelineVersion>;
    errorsByKey: Map<string, string>;
  }> {
    const uniqueVersionRefs = new Map<string, { pipelineId: string; pipelineVersionId: string }>();
    displayRuns.forEach((displayRun) => {
      const pipelineId = displayRun.run.pipeline_version_reference?.pipeline_id;
      const pipelineVersionId = displayRun.run.pipeline_version_reference?.pipeline_version_id;
      if (pipelineId && pipelineVersionId) {
        uniqueVersionRefs.set(_pipelineVersionKey(pipelineId, pipelineVersionId), {
          pipelineId,
          pipelineVersionId,
        });
      }
    });

    const pipelineVersionsByKey = new Map<string, V2PipelineVersion>();
    const errorsByKey = new Map<string, string>();
    await Promise.all(
      Array.from(uniqueVersionRefs.entries()).map(
        async ([key, { pipelineId, pipelineVersionId }]) => {
          try {
            const pipelineVersion = await Apis.pipelineServiceApiV2.getPipelineVersion(
              pipelineId,
              pipelineVersionId,
            );
            pipelineVersionsByKey.set(key, pipelineVersion);
          } catch (err) {
            // Deleting a version does not delete the runs that reference it.
            if (err instanceof ResponseError && err.response.status === 404) {
              return;
            }
            errorsByKey.set(key, await errorToMessage(err));
            logger.error(
              `Failed to get pipeline version ${pipelineVersionId} for pipeline ${pipelineId}`,
              err,
            );
          }
        },
      ),
    );
    return { pipelineVersionsByKey, errorsByKey };
  }

  /**
   * For the given DisplayRun, look up its associated pipeline version from the already-fetched
   * pipelineVersionsByKey batch and attach its name to the DisplayRun. If the ApiRun has no
   * Pipeline ID, then the corresponding DisplayRun will show '-'.
   */
  private _setPipelineVersionName(
    displayRun: DisplayRun,
    pipelineVersionsByKey: Map<string, V2PipelineVersion>,
    errorsByKey: Map<string, string>,
  ): void {
    const pipelineId = displayRun.run.pipeline_version_reference?.pipeline_id;
    const pipelineVersionId = displayRun.run.pipeline_version_reference?.pipeline_version_id;
    if (pipelineId && pipelineVersionId) {
      const key = _pipelineVersionKey(pipelineId, pipelineVersionId);
      const pipelineVersion = pipelineVersionsByKey.get(key);
      if (pipelineVersion) {
        displayRun.pipelineVersion = {
          displayName: pipelineVersion.display_name,
          pipelineId: pipelineVersion.pipeline_id,
          usePlaceholder: false,
          versionId: pipelineVersion.pipeline_version_id,
        };
      } else if (errorsByKey.has(key)) {
        const errorMessage = errorsByKey.get(key);
        displayRun.error =
          'Failed to get associated pipeline version' + (errorMessage ? ': ' + errorMessage : '');
      } else {
        displayRun.pipelineVersion = { usePlaceholder: false, unavailable: true };
      }
    } else if (displayRun.run.pipeline_spec) {
      displayRun.pipelineVersion = displayRun.recurringRun?.id
        ? { usePlaceholder: true, recurringRunId: displayRun.recurringRun.id }
        : { usePlaceholder: true };
    }
  }
}

export default RunList;
