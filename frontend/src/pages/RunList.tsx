/*
 * Copyright 2018 Google LLC
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
import CustomTable, { Column, Row, CustomRendererProps } from '../components/CustomTable';
import Metric from '../components/Metric';
import RunUtils, { MetricMetadata, ExperimentInfo } from '../../src/lib/RunUtils';
import { ApiRun, ApiRunMetric, RunStorageState, ApiRunDetail } from '../../src/apis/run';
import { Apis, RunSortKeys, ListRequest } from '../lib/Apis';
import { Link, RouteComponentProps } from 'react-router-dom';
import { NodePhase } from '../lib/StatusUtils';
import { PredicateOp, ApiFilter } from '../apis/filter';
import { RoutePage, RouteParams, QUERY_PARAMS } from '../components/Router';
import { URLParser } from '../lib/URLParser';
import { commonCss, color } from '../Css';
import { formatDateString, logger, errorToMessage, getRunDuration } from '../lib/Utils';
import { statusToIcon } from './Status';
import Tooltip from '@material-ui/core/Tooltip';
import { TFunction } from 'i18next';

interface PipelineVersionInfo {
  displayName?: string;
  versionId?: string;
  runId?: string;
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
  run: ApiRun;
  pipelineVersion?: PipelineVersionInfo;
  error?: string;
}

interface DisplayMetric {
  metadata?: MetricMetadata;
  metric?: ApiRunMetric;
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
    storageState?: RunStorageState;
    t: TFunction;
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
        label: this.props.t('experiments:runName'),
        sortKey: RunSortKeys.NAME,
      },
      {
        customRenderer: this._statusCustomRenderer,
        flex: 0.5,
        label: this.props.t('common:status'),
      },
      { label: this.props.t('common:duration'), flex: 0.5 },
      {
        customRenderer: this._pipelineVersionCustomRenderer,
        label: this.props.t('common:pipelineVersion'),
        flex: 1,
      },
      {
        customRenderer: this._recurringRunCustomRenderer,
        label: this.props.t('common:recurringRun'),
        flex: 0.5,
      },
      { label: this.props.t('common:startTime'), flex: 1, sortKey: RunSortKeys.CREATED_AT },
    ];

    if (!this.props.hideExperimentColumn) {
      columns.splice(3, 0, {
        customRenderer: this._experimentCustomRenderer,
        flex: 1,
        label: this.props.t('common:experiment'),
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
    const { t } = this.props;
    const rows: Row[] = this.state.runs.map(r => {
      const displayMetrics = metricMetadata.map(metadata => {
        const displayMetric: DisplayMetric = { metadata };
        if (r.run.metrics) {
          const foundMetric = r.run.metrics.find(m => m.name === metadata.name);
          if (foundMetric && foundMetric.number_value !== undefined) {
            displayMetric.metric = foundMetric;
          }
        }
        return displayMetric;
      });
      const row = {
        error: r.error,
        id: r.run.id!,
        otherFields: [
          r.run!.name,
          r.run.status || '-',
          getRunDuration(r.run),
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
          t={t}
          columns={columns}
          rows={rows}
          selectedIds={this.props.selectedIds}
          initialSortColumn={RunSortKeys.CREATED_AT}
          ref={this._tableRef}
          filterLabel={t('experiments:filterRuns')}
          updateSelection={this.props.onSelectionChange}
          reload={this._loadRuns.bind(this)}
          disablePaging={this.props.disablePaging}
          disableSorting={this.props.disableSorting}
          disableSelection={this.props.disableSelection}
          noFilterBox={this.props.noFilterBox}
          emptyMessage={
            `${t('common:no')}` +
            `${
              this.props.storageState === RunStorageState.ARCHIVED
                ? t('common:archived')
                : t('common:available')
            }` +
            ` ${t('common:runFound')}` +
            `${
              this.props.experimentIdMask
                ? t('common:forThisExperiment')
                : this.props.namespaceMask
                ? t('common:forThisNamespace')
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
    const search = new URLParser(this.props).build({ [QUERY_PARAMS.fromRunId]: props.id });
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
    // If the getJob call failed or a run has no job, we display a placeholder.
    if (!props.value || !props.value.id) {
      return <div>-</div>;
    }
    const url = RoutePage.RECURRING_RUN.replace(':' + RouteParams.runId, props.value.id || '');
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

  public _statusCustomRenderer: React.FC<CustomRendererProps<NodePhase>> = (
    props: CustomRendererProps<NodePhase>,
  ) => {
    const { t } = this.props;
    return statusToIcon(t, props.value);
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

    return <Metric metric={displayMetric.metric} metadata={displayMetric.metadata} />;
  };

  protected async _loadRuns(request: ListRequest): Promise<string> {
    let displayRuns: DisplayRun[] = [];
    let nextPageToken = '';

    if (Array.isArray(this.props.runIdListMask)) {
      displayRuns = this.props.runIdListMask.map(id => ({ run: { id } }));
      // listRuns doesn't currently support batching by IDs, so in this case we retrieve each run
      // individually.
      await this._getAndSetRuns(displayRuns);
    } else {
      // Load all runs
      if (this.props.storageState) {
        try {
          // Augment the request filter with the storage state predicate
          const filter = JSON.parse(
            decodeURIComponent(request.filter || '{"predicates": []}'),
          ) as ApiFilter;
          filter.predicates = (filter.predicates || []).concat([
            {
              key: 'storage_state',
              // Use EQUALS ARCHIVED or NOT EQUALS ARCHIVED to account for cases where the field
              // is missing, in which case it should be counted as available.
              op:
                this.props.storageState === RunStorageState.ARCHIVED
                  ? PredicateOp.EQUALS
                  : PredicateOp.NOTEQUALS,
              string_value: RunStorageState.ARCHIVED.toString(),
            },
          ]);
          request.filter = encodeURIComponent(JSON.stringify(filter));
        } catch (err) {
          logger.error('Could not parse request filter: ', request.filter);
        }
      }

      try {
        let resourceReference: {
          keyType?: 'EXPERIMENT' | 'NAMESPACE';
          keyId?: string;
        } = {};
        if (this.props.experimentIdMask) {
          resourceReference = {
            keyType: 'EXPERIMENT',
            keyId: this.props.experimentIdMask,
          };
        } else if (this.props.namespaceMask) {
          resourceReference = {
            keyType: 'NAMESPACE',
            keyId: this.props.namespaceMask,
          };
        }
        const response = await Apis.runServiceApi.listRuns(
          request.pageToken,
          request.pageSize,
          request.sortBy,
          resourceReference.keyType,
          resourceReference.keyId,
          request.filter,
        );

        displayRuns = (response.runs || []).map(r => ({ run: r }));
        nextPageToken = response.next_page_token || '';
      } catch (err) {
        const error = new Error(await errorToMessage(err));
        this.props.onError(this.props.t('pipelines:errorFetchRuns'), error);
        // No point in continuing if we couldn't retrieve any runs.
        return '';
      }
    }

    await this._setColumns(displayRuns);

    this.setState({
      metrics: RunUtils.extractMetricMetadata(displayRuns.map(r => r.run)),
      runs: displayRuns,
    });
    return nextPageToken;
  }

  private async _setColumns(displayRuns: DisplayRun[]): Promise<DisplayRun[]> {
    return Promise.all(
      displayRuns.map(async displayRun => {
        this._setRecurringRun(displayRun);

        await this._getAndSetPipelineVersionNames(displayRun);

        if (!this.props.hideExperimentColumn) {
          await this._getAndSetExperimentNames(displayRun);
        }
        return displayRun;
      }),
    );
  }

  private _setRecurringRun(displayRun: DisplayRun): void {
    const recurringRunId = RunUtils.getRecurringRunId(displayRun.run);
    if (recurringRunId) {
      // TODO: It would be better to use name here, but that will require another n API calls at
      // this time.
      displayRun.recurringRun = { id: recurringRunId, displayName: recurringRunId };
    }
  }

  /**
   * For each run ID, fetch its corresponding run, and set it in DisplayRuns
   */
  private _getAndSetRuns(displayRuns: DisplayRun[]): Promise<DisplayRun[]> {
    return Promise.all(
      displayRuns.map(async displayRun => {
        let getRunResponse: ApiRunDetail;
        try {
          getRunResponse = await Apis.runServiceApi.getRun(displayRun.run!.id!);
          displayRun.run = getRunResponse.run!;
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
    const pipelineVersionId = RunUtils.getPipelineVersionId(displayRun.run);
    const pipelineId = RunUtils.getPipelineId(displayRun.run);
    if (pipelineVersionId) {
      try {
        const pipelineVersion = await Apis.pipelineServiceApi.getPipelineVersion(pipelineVersionId);
        const pipelineVersionName = pipelineVersion.name || '';
        displayRun.pipelineVersion = {
          displayName: pipelineVersionName,
          pipelineId: RunUtils.getPipelineIdFromApiPipelineVersion(pipelineVersion),
          usePlaceholder: false,
          versionId: pipelineVersionId,
        };
      } catch (err) {
        displayRun.error =
          'Failed to get associated pipeline version: ' + (await errorToMessage(err));
        return;
      }
    } else if (pipelineId) {
      // For backward compatibility. Runs created before version is introduced
      // refer to pipeline id instead of pipeline version id.
      let pipelineName = RunUtils.getPipelineName(displayRun.run);
      if (!pipelineName) {
        try {
          const pipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);
          pipelineName = pipeline.name || '';
        } catch (err) {
          displayRun.error = 'Failed to get associated pipeline: ' + (await errorToMessage(err));
          return;
        }
      }
      displayRun.pipelineVersion = {
        displayName: pipelineName,
        pipelineId,
        usePlaceholder: false,
        versionId: undefined,
      };
    } else if (!!RunUtils.getWorkflowManifest(displayRun.run)) {
      displayRun.pipelineVersion = { usePlaceholder: true };
    }
  }

  /**
   * For the given DisplayRun, get its ApiRun and retrieve that ApiRun's Experiment ID if it has
   * one, then use that Experiment ID to fetch its associated Experiment and attach that
   * Experiment's name to the DisplayRun. If the ApiRun has no Experiment ID, then the corresponding
   * DisplayRun will show '-'.
   */
  private async _getAndSetExperimentNames(displayRun: DisplayRun): Promise<void> {
    const experimentId = RunUtils.getFirstExperimentReferenceId(displayRun.run);
    if (experimentId) {
      let experimentName = RunUtils.getFirstExperimentReferenceName(displayRun.run);
      if (!experimentName) {
        try {
          const experiment = await Apis.experimentServiceApi.getExperiment(experimentId);
          experimentName = experiment.name || '';
        } catch (err) {
          displayRun.error = 'Failed to get associated experiment: ' + (await errorToMessage(err));
          return;
        }
      }
      displayRun.experiment = {
        displayName: experimentName,
        id: experimentId,
      };
    }
  }
}
export default RunList;
