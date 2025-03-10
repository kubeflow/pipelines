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

import * as React from 'react';
import CustomTable, { Column, Row, CustomRendererProps } from 'src/components/CustomTable';
import Metric from 'src/components/Metric';
import { MetricMetadata, ExperimentInfo } from 'src/lib/RunUtils';
import { V2beta1Run, V2beta1RuntimeState, V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { V2beta1ListExperimentsResponse } from 'src/apisv2beta1/experiment';
import { Apis, RunSortKeys, ListRequest } from 'src/lib/Apis';
import { Link, RouteComponentProps } from 'react-router-dom';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import { RoutePage, RouteParams, QUERY_PARAMS } from 'src/components/Router';
import { URLParser } from 'src/lib/URLParser';
import { commonCss, color } from 'src/Css';
import { formatDateString, logger, errorToMessage, getRunDurationV2 } from 'src/lib/Utils';
import { statusToIcon } from './StatusV2';
import Tooltip from '@material-ui/core/Tooltip';

interface PipelineVersionInfo {
  displayName?: string;
  versionId?: string;
  runId?: string;
  recurringRunId?: string;
  pipelineId?: string;
  usePlaceholder: boolean;
}

interface RecurringRunInfo {
  displayName?: string;
  id?: string;
}

interface DisplayRun {
  experiment?: ExperimentInfo;
  recurringRun?: RecurringRunInfo;
  run: V2beta1Run;
  pipelineVersion?: PipelineVersionInfo;
  error?: string;
}

interface DisplayMetric {
  metadata?: MetricMetadata;
  // run metric field is currently not supported in v2 API
  // Context: https://github.com/kubeflow/pipelines/issues/8957
}

// Both masks cannot be provided together.
type MaskProps = Exclude<
  { experimentIdMask?: string; namespaceMask?: string },
  { experimentIdMask: string; namespaceMask: string }
>;

export type RunListProps = MaskProps &
  RouteComponentProps & {
    disablePaging?: boolean;
    disableSelection?: boolean;
    disableSorting?: boolean;
    hideExperimentColumn?: boolean;
    hideMetricMetadata?: boolean;
    noFilterBox?: boolean;
    onError: (message: string, error: Error) => void;
    onSelectionChange?: (selectedRunIds: string[]) => void;
    runIdListMask?: string[];
    selectedIds?: string[];
    storageState?: V2beta1RunStorageState;
  };

interface RunListState {
  metrics: MetricMetadata[];
  runs: DisplayRun[];
}

class RunList extends React.PureComponent<RunListProps, RunListState> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      metrics: [],
      runs: [],
    };
  }

  public render(): JSX.Element {
    // Only show the two most prevalent metrics
    const metricMetadata: MetricMetadata[] = this.state.metrics.slice(0, 2);
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 1.5,
        label: 'Run name',
        sortKey: RunSortKeys.NAME,
      },
      { customRenderer: this._statusCustomRenderer, flex: 0.5, label: 'Status' },
      { label: 'Duration', flex: 0.5 },
      { customRenderer: this._pipelineVersionCustomRenderer, label: 'Pipeline Version', flex: 1 },
      { customRenderer: this._recurringRunCustomRenderer, label: 'Recurring Run', flex: 0.5 },
      { label: 'Start time', flex: 1, sortKey: RunSortKeys.CREATED_AT },
    ];

    if (!this.props.hideExperimentColumn) {
      columns.splice(3, 0, {
        customRenderer: this._experimentCustomRenderer,
        flex: 1,
        label: 'Experiment',
      });
    }

    if (metricMetadata.length && !this.props.hideMetricMetadata) {
      // This is a column of empty cells with a left border to separate the metrics from the other
      // columns.
      columns.push({
        customRenderer: this._metricBufferCustomRenderer,
        flex: 0.1,
        label: '',
      });

      columns.push(
        ...metricMetadata.map(metadata => {
          return {
            customRenderer: this._metricCustomRenderer,
            flex: 0.5,
            label: metadata.name!,
          };
        }),
      );
    }

    const rows: Row[] = this.state.runs.map(r => {
      const displayMetrics = metricMetadata.map(metadata => {
        const displayMetric: DisplayMetric = { metadata };
        return displayMetric;
      });
      const row = {
        error: r.error,
        id: r.run.run_id!,
        otherFields: [
          r.run!.display_name,
          r.run.state || '-',
          getRunDurationV2(r.run),
          r.pipelineVersion,
          r.recurringRun,
          formatDateString(r.run.created_at),
        ] as any,
      };
      if (!this.props.hideExperimentColumn) {
        row.otherFields.splice(3, 0, r.experiment);
      }
      if (displayMetrics.length && !this.props.hideMetricMetadata) {
        row.otherFields.push(''); // Metric buffer column
        row.otherFields.push(...(displayMetrics as any));
      }
      return row;
    });

    return (
      <div>
        <CustomTable
          columns={columns}
          rows={rows}
          selectedIds={this.props.selectedIds}
          initialSortColumn={RunSortKeys.CREATED_AT}
          ref={this._tableRef}
          filterLabel='Filter runs'
          updateSelection={this.props.onSelectionChange}
          reload={this._loadRuns.bind(this)}
          disablePaging={this.props.disablePaging}
          disableSorting={this.props.disableSorting}
          disableSelection={this.props.disableSelection}
          noFilterBox={this.props.noFilterBox}
          emptyMessage={
            `No` +
            `${
              this.props.storageState === V2beta1RunStorageState.ARCHIVED
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
      </div>
    );
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
      <Tooltip title={props.value || ''} enterDelay={300} placement='top-start'>
        <Link
          className={commonCss.link}
          onClick={e => e.stopPropagation()}
          to={RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, props.id)}
        >
          {props.value}
        </Link>
      </Tooltip>
    );
  };

  public _pipelineVersionCustomRenderer: React.FC<CustomRendererProps<PipelineVersionInfo>> = (
    props: CustomRendererProps<PipelineVersionInfo>,
  ) => {
    // If the getPipeline call failed or a run has no pipeline, we display a placeholder.
    if (!props.value || (!props.value.usePlaceholder && !props.value.pipelineId)) {
      return <div>-</div>;
    }
    const urlParser = new URLParser(this.props);
    const search = props.value.recurringRunId
      ? urlParser.build({ [QUERY_PARAMS.fromRecurringRunId]: props.value.recurringRunId })
      : urlParser.build({ [QUERY_PARAMS.fromRunId]: props.id });
    const url = props.value.usePlaceholder
      ? RoutePage.PIPELINE_DETAILS_NO_VERSION.replace(':' + RouteParams.pipelineId + '?', '') +
        search
      : !!props.value.versionId
      ? RoutePage.PIPELINE_DETAILS.replace(
          ':' + RouteParams.pipelineId,
          props.value.pipelineId || '',
        ).replace(':' + RouteParams.pipelineVersionId, props.value.versionId || '')
      : RoutePage.PIPELINE_DETAILS_NO_VERSION.replace(
          ':' + RouteParams.pipelineId,
          props.value.pipelineId || '',
        );
    if (props.value.usePlaceholder) {
      return (
        <Link className={commonCss.link} onClick={e => e.stopPropagation()} to={url}>
          [View pipeline]
        </Link>
      );
    } else {
      // Display name could be too long, so we show the full content in tooltip on hover.
      return (
        <Tooltip title={props.value.displayName || ''} enterDelay={300} placement='top-start'>
          <Link className={commonCss.link} onClick={e => e.stopPropagation()} to={url}>
            {props.value.displayName}
          </Link>
        </Tooltip>
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
    const url = RoutePage.RECURRING_RUN_DETAILS.replace(
      ':' + RouteParams.recurringRunId,
      props.value.id || '',
    );
    return (
      <Link className={commonCss.link} onClick={e => e.stopPropagation()} to={url}>
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
        className={commonCss.link}
        onClick={e => e.stopPropagation()}
        to={RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, props.value.id)}
      >
        {props.value.displayName}
      </Link>
    );
  };

  public _statusCustomRenderer: React.FC<CustomRendererProps<V2beta1RuntimeState>> = (
    props: CustomRendererProps<V2beta1RuntimeState>,
  ) => {
    return statusToIcon(props.value);
  };

  public _metricBufferCustomRenderer: React.FC<CustomRendererProps<{}>> = (
    props: CustomRendererProps<{}>,
  ) => {
    return <div style={{ borderLeft: `1px solid ${color.divider}`, padding: '20px 0' }} />;
  };

  public _metricCustomRenderer: React.FC<CustomRendererProps<DisplayMetric>> = (
    props: CustomRendererProps<DisplayMetric>,
  ) => {
    const displayMetric = props.value;
    if (!displayMetric) {
      return <div />;
    }

    return <Metric metadata={displayMetric.metadata} />;
  };

  protected async _loadRuns(request: ListRequest): Promise<string> {
    let displayRuns: DisplayRun[] = [];
    let nextPageToken = '';

    if (Array.isArray(this.props.runIdListMask)) {
      displayRuns = this.props.runIdListMask.map(id => ({ run: { run_id: id } }));
      const filter = JSON.parse(
        decodeURIComponent(request.filter || '{"predicates": []}'),
      ) as V2beta1Filter;
      // listRuns doesn't currently support batching by IDs, so in this case we retrieve and filter
      // each run individually.
      await this._getAndSetRuns(displayRuns);
      const predicates = filter.predicates?.filter(
        p => p.key === 'name' && p.operation === V2beta1PredicateOperation.ISSUBSTRING,
      );
      const substrings = predicates?.map(p => p.string_value?.toLowerCase() || '') || [];
      displayRuns = displayRuns.filter(runDetail => {
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
          ) as V2beta1Filter;
          filter.predicates = (filter.predicates || []).concat([
            {
              key: 'storage_state',
              // Use EQUALS ARCHIVED or NOT EQUALS ARCHIVED to account for cases where the field
              // is missing, in which case it should be counted as available.
              operation:
                this.props.storageState === V2beta1RunStorageState.ARCHIVED
                  ? V2beta1PredicateOperation.EQUALS
                  : V2beta1PredicateOperation.NOTEQUALS,
              string_value: V2beta1RunStorageState.ARCHIVED.toString(),
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
        );

        displayRuns = (response.runs || []).map(r => ({ run: r }));
        nextPageToken = response.next_page_token || '';
      } catch (err) {
        const error = new Error(await errorToMessage(err));
        this.props.onError('Error: failed to fetch runs.', error);
        // No point in continuing if we couldn't retrieve any runs.
        return '';
      }
    }

    await this._setColumns(displayRuns);

    this.setState({
      // metrics: RunUtils.extractMetricMetadata(displayRuns.map(r => r.run)),
      runs: displayRuns,
    });
    return nextPageToken;
  }

  private async _setColumns(displayRuns: DisplayRun[]): Promise<DisplayRun[]> {
    let experimentsResponse: V2beta1ListExperimentsResponse;
    let experimentsGetError: string;
    try {
      if (!this.props.namespaceMask) {
        // Single-user mode.
        experimentsResponse = await Apis.experimentServiceApiV2.listExperiments();
      } else {
        // Multi-user mode.
        experimentsResponse = await Apis.experimentServiceApiV2.listExperiments(
          undefined,
          undefined,
          undefined,
          undefined,
          this.props.namespaceMask,
        );
      }
    } catch (error) {
      experimentsGetError = 'Failed to get associated experiment: ' + (await errorToMessage(error));
    }

    return Promise.all(
      displayRuns.map(async displayRun => {
        this._setRecurringRun(displayRun);

        await this._getAndSetPipelineVersionNames(displayRun);

        if (!this.props.hideExperimentColumn) {
          const experimentId = displayRun.run.experiment_id;

          if (experimentId) {
            const experiment = experimentsResponse?.experiments?.find(
              e => e.experiment_id === displayRun.run.experiment_id,
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
      displayRuns.map(async displayRun => {
        let getRunResponse: V2beta1Run;
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
   * For the given DisplayRun, get its ApiRun and retrieve that ApiRun's Pipeline ID if it has one,
   * then use that Pipeline ID to fetch its associated Pipeline and attach that Pipeline's name to
   * the DisplayRun. If the ApiRun has no Pipeline ID, then the corresponding DisplayRun will show
   * '-'.
   */
  private async _getAndSetPipelineVersionNames(displayRun: DisplayRun): Promise<void> {
    const pipelineId = displayRun.run.pipeline_version_reference?.pipeline_id;
    const pipelineVersionId = displayRun.run.pipeline_version_reference?.pipeline_version_id;
    if (pipelineId && pipelineVersionId) {
      try {
        const pipelineVersion = await Apis.pipelineServiceApiV2.getPipelineVersion(
          pipelineId,
          pipelineVersionId,
        );
        displayRun.pipelineVersion = {
          displayName: pipelineVersion.display_name,
          pipelineId: pipelineVersion.pipeline_id,
          usePlaceholder: false,
          versionId: pipelineVersion.pipeline_version_id,
        };
      } catch (err) {
        displayRun.error =
          'Failed to get associated pipeline version: ' + (await errorToMessage(err));
        return;
      }
    } else if (displayRun.run.pipeline_spec) {
      // pipeline_spec in v2 can store either workflow_manifest or pipeline_manifest
      displayRun.pipelineVersion = displayRun.recurringRun?.id
        ? { usePlaceholder: true, recurringRunId: displayRun.recurringRun.id }
        : { usePlaceholder: true };
    }
  }
}

export default RunList;
