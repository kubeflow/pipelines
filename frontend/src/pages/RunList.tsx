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
import RunUtils, { MetricMetadata } from '../../src/lib/RunUtils';
import { ApiRun, ApiResourceType, RunMetricFormat, ApiRunMetric, RunStorageState, ApiRunDetail } from '../../src/apis/run';
import { Apis, RunSortKeys, ListRequest } from '../lib/Apis';
import { Link, RouteComponentProps } from 'react-router-dom';
import { NodePhase, statusToIcon } from './Status';
import { PredicateOp, ApiFilter } from '../apis/filter';
import { RoutePage, RouteParams, QUERY_PARAMS } from '../components/Router';
import { URLParser } from '../lib/URLParser';
import { commonCss, color } from '../Css';
import { formatDateString, logger, errorToMessage, getRunDuration } from '../lib/Utils';
import { stylesheet } from 'typestyle';

const css = stylesheet({
  metricContainer: {
    background: '#f6f7f9',
    marginLeft: 6,
    marginRight: 10,
  },
  metricFill: {
    background: '#cbf0f8',
    boxSizing: 'border-box',
    color: '#202124',
    fontFamily: 'Roboto',
    fontSize: 13,
    textIndent: 6,
  },
});

interface ExperimentInfo {
  displayName?: string;
  id: string;
}

interface PipelineInfo {
  displayName?: string;
  id?: string;
  runId?: string;
  showLink: boolean;
}

interface DisplayRun {
  experiment?: ExperimentInfo;
  run: ApiRun;
  pipeline?: PipelineInfo;
  error?: string;
}

interface DisplayMetric {
  metadata?: MetricMetadata;
  metric?: ApiRunMetric;
}

export interface RunListProps extends RouteComponentProps {
  disablePaging?: boolean;
  disableSelection?: boolean;
  disableSorting?: boolean;
  experimentIdMask?: string;
  hideExperimentColumn?: boolean;
  noFilterBox?: boolean;
  onError: (message: string, error: Error) => void;
  onSelectionChange?: (selectedRunIds: string[]) => void;
  runIdListMask?: string[];
  selectedIds?: string[];
  storageState?: RunStorageState;
}

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
        flex: 2,
        label: 'Run name',
        sortKey: RunSortKeys.NAME,
      },
      { customRenderer: this._statusCustomRenderer, flex: 0.5, label: 'Status' },
      { label: 'Duration', flex: 0.5 },
      { customRenderer: this._pipelineCustomRenderer, label: 'Pipeline', flex: 1 },
      { label: 'Start time', flex: 1, sortKey: RunSortKeys.CREATED_AT },
    ];

    if (!this.props.hideExperimentColumn) {
      columns.splice(3, 0, { customRenderer: this._experimentCustomRenderer, label: 'Experiment', flex: 1 });
    }

    if (metricMetadata.length) {
      // This is a column of empty cells with a left border to separate the metrics from the other
      // columns.
      columns.push({
        customRenderer: this._metricBufferCustomRenderer,
        flex: 0.1,
        label: '',
      });

      columns.push(...metricMetadata.map((metadata) => {
        return {
          customRenderer: this._metricCustomRenderer,
          flex: 1,
          label: metadata.name!
        };
      }));
    }

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
          r.pipeline,
          formatDateString(r.run.created_at),
        ] as any,
      };
      if (!this.props.hideExperimentColumn) {
        row.otherFields.splice(3, 0, r.experiment);
      }
      if (displayMetrics.length) {
        row.otherFields.push(''); // Metric buffer column
        row.otherFields.push(...displayMetrics as any);
      }
      return row;
    });

    return (<div>
      <CustomTable columns={columns} rows={rows} selectedIds={this.props.selectedIds}
        initialSortColumn={RunSortKeys.CREATED_AT} ref={this._tableRef} filterLabel='Filter runs'
        updateSelection={this.props.onSelectionChange} reload={this._loadRuns.bind(this)}
        disablePaging={this.props.disablePaging} disableSorting={this.props.disableSorting}
        disableSelection={this.props.disableSelection} noFilterBox={this.props.noFilterBox}
        emptyMessage={
          `No` +
          `${this.props.storageState === RunStorageState.ARCHIVED ? ' archived' : ' available'}` +
          ` runs found` +
          `${this.props.experimentIdMask ? ' for this experiment' : ''}.`
        }
      />
    </div>);
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      await this._tableRef.current.reload();
    }
  }

  public _nameCustomRenderer: React.FC<CustomRendererProps<string>> = (props: CustomRendererProps<string>) => {
    return <Link className={commonCss.link} onClick={(e) => e.stopPropagation()}
      to={RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, props.id)}>{props.value}</Link>;
  }

  public _pipelineCustomRenderer: React.FC<CustomRendererProps<PipelineInfo>> = (props: CustomRendererProps<PipelineInfo>) => {
    // If the getPipeline call failed or a run has no pipeline, we display a placeholder.
    if (!props.value || (!props.value.showLink && !props.value.id)) {
      return <div>-</div>;
    }
    const search = new URLParser(this.props).build({ [QUERY_PARAMS.fromRunId]: props.id });
    const url = props.value.showLink ?
      RoutePage.PIPELINE_DETAILS.replace(':' + RouteParams.pipelineId + '?', '') + search :
      RoutePage.PIPELINE_DETAILS.replace(':' + RouteParams.pipelineId, props.value.id || '');
    return (
      <Link className={commonCss.link} onClick={(e) => e.stopPropagation()}
        to={url}>
        {props.value.showLink ? '[View pipeline]' : props.value.displayName}
      </Link>
    );
  }

  public _experimentCustomRenderer: React.FC<CustomRendererProps<ExperimentInfo>> = (props: CustomRendererProps<ExperimentInfo>) => {
    // If the getExperiment call failed or a run has no experiment, we display a placeholder.
    if (!props.value || !props.value.id) {
      return <div>-</div>;
    }
    return (
      <Link className={commonCss.link} onClick={(e) => e.stopPropagation()}
        to={RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, props.value.id)}>
        {props.value.displayName}
      </Link>
    );
  }

  public _statusCustomRenderer: React.FC<CustomRendererProps<NodePhase>> = (props: CustomRendererProps<NodePhase>) => {
    return statusToIcon(props.value);
  }

  public _metricBufferCustomRenderer: React.FC<CustomRendererProps<{}>> = (props: CustomRendererProps<{}>) => {
    return <div style={{ borderLeft: `1px solid ${color.divider}`, padding: '20px 0' }} />;
  }

  public _metricCustomRenderer: React.FC<CustomRendererProps<DisplayMetric>> = (props: CustomRendererProps<DisplayMetric>) => {
    const displayMetric = props.value;
    if (!displayMetric || !displayMetric.metric ||
      displayMetric.metric.number_value === undefined) {
      return <div />;
    }

    const leftSpace = 6;
    let displayString = '';
    let width = '';

    if (displayMetric.metric.format === RunMetricFormat.PERCENTAGE) {
      displayString = (displayMetric.metric.number_value * 100).toFixed(3) + '%';
      width = `calc(${displayString})`;
    } else {

      // Non-percentage metrics must contain metadata
      if (!displayMetric.metadata) {
        return <div />;
      }

      displayString = displayMetric.metric.number_value.toFixed(3);

      if (displayMetric.metadata.maxValue === 0 && displayMetric.metadata.minValue === 0) {
        return <div style={{ paddingLeft: leftSpace }}>{displayString}</div>;
      }

      if (displayMetric.metric.number_value - displayMetric.metadata.minValue < 0) {
        logger.error(`Run ${props.id}'s metric ${displayMetric.metadata.name}'s value:`
          + ` (${displayMetric.metric.number_value}) was lower than the supposed minimum of`
          + ` (${displayMetric.metadata.minValue})`);
        return <div style={{ paddingLeft: leftSpace }}>{displayString}</div>;
      }

      if (displayMetric.metadata.maxValue - displayMetric.metric.number_value < 0) {
        logger.error(`Run ${props.id}'s metric ${displayMetric.metadata.name}'s value:`
          + ` (${displayMetric.metric.number_value}) was greater than the supposed maximum of`
          + ` (${displayMetric.metadata.maxValue})`);
        return <div style={{ paddingLeft: leftSpace }}>{displayString}</div>;
      }

      const barWidth =
        (displayMetric.metric.number_value - displayMetric.metadata.minValue)
        / (displayMetric.metadata.maxValue - displayMetric.metadata.minValue)
        * 100;

      width = `calc(${barWidth}%)`;
    }
    return (
      <div className={css.metricContainer}>
        <div className={css.metricFill} style={{ width }}>
          {displayString}
        </div>
      </div>
    );
  }

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
          const filter = JSON.parse(decodeURIComponent(request.filter || '{"predicates": []}')) as ApiFilter;
          filter.predicates = (filter.predicates || []).concat([{
            key: 'storage_state',
            // Use EQUALS ARCHIVED or NOT EQUALS ARCHIVED to account for cases where the field
            // is missing, in which case it should be counted as available.
            op: this.props.storageState === RunStorageState.ARCHIVED ? PredicateOp.EQUALS : PredicateOp.NOTEQUALS,
            string_value: RunStorageState.ARCHIVED.toString(),
          }]);
          request.filter = encodeURIComponent(JSON.stringify(filter));
        } catch (err) {
          logger.error('Could not parse request filter: ', request.filter);
        }
      }

      try {
        const response = await Apis.runServiceApi.listRuns(
          request.pageToken,
          request.pageSize,
          request.sortBy,
          this.props.experimentIdMask ? ApiResourceType.EXPERIMENT.toString() : undefined,
          this.props.experimentIdMask,
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

    await this._getAndSetPipelineNames(displayRuns);
    if (!this.props.hideExperimentColumn) {
      await this._getAndSetExperimentNames(displayRuns);
    }

    this.setState({
      metrics: RunUtils.extractMetricMetadata(displayRuns.map(r => r.run)),
      runs: displayRuns,
    });
    return nextPageToken;
  }

  /**
   * For each run ID, fetch its corresponding run, and set it in DisplayRuns
   */
  private _getAndSetRuns(displayRuns: DisplayRun[]): Promise<DisplayRun[]> {
    return Promise.all(displayRuns.map(async displayRun => {
      let getRunResponse: ApiRunDetail;
      try {
        getRunResponse = await Apis.runServiceApi.getRun(displayRun.run!.id!);
        displayRun.run = getRunResponse.run!;
      } catch (err) {
        displayRun.error = await errorToMessage(err);
      }
      return displayRun;
    }));
  }

  /**
   * For each DisplayRun, get its ApiRun and retrieve that ApiRun's Pipeline ID if it has one, then
   * use that Pipeline ID to fetch its associated Pipeline and attach that Pipeline's name to the
   * DisplayRun. If the ApiRun has no Pipeline ID, then the corresponding DisplayRun will show '-'.
   */
  private _getAndSetPipelineNames(displayRuns: DisplayRun[]): Promise<DisplayRun[]> {
    return Promise.all(
      displayRuns.map(async (displayRun) => {
        const pipelineId = RunUtils.getPipelineId(displayRun.run);
        if (pipelineId) {
          try {
            const pipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);
            displayRun.pipeline = { displayName: pipeline.name || '', id: pipelineId, showLink: false };
          } catch (err) {
            // This could be an API exception, or a JSON parse exception.
            displayRun.error = 'Failed to get associated pipeline: ' + await errorToMessage(err);
          }
        } else if (!!RunUtils.getPipelineSpec(displayRun.run)) {
          displayRun.pipeline = { showLink: true };
        }
        return displayRun;
      })
    );
  }

  /**
   * For each DisplayRun, get its ApiRun and retrieve that ApiRun's Experiment ID if it has one,
   * then use that Experiment ID to fetch its associated Experiment and attach that Experiment's
   * name to the DisplayRun. If the ApiRun has no Experiment ID, then the corresponding DisplayRun
   * will show '-'.
   */
  private _getAndSetExperimentNames(displayRuns: DisplayRun[]): Promise<DisplayRun[]> {
    return Promise.all(
      displayRuns.map(async (displayRun) => {
        const experimentId = RunUtils.getFirstExperimentReferenceId(displayRun.run);
        if (experimentId) {
          try {
            // TODO: Experiment could be an optional field in state since whenever the RunList is
            // created from the ExperimentDetails page, we already have the experiment (and will)
            // be fetching the same one over and over here.
            const experiment = await Apis.experimentServiceApi.getExperiment(experimentId);
            displayRun.experiment = { displayName: experiment.name || '', id: experimentId };
          } catch (err) {
            // This could be an API exception, or a JSON parse exception.
            displayRun.error = 'Failed to get associated experiment: ' + await errorToMessage(err);
          }
        }
        return displayRun;
      })
    );
  }
}

export default RunList;
