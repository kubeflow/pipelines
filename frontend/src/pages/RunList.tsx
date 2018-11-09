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
import CustomTable, { Column, Row } from '../components/CustomTable';
import RunUtils from '../../src/lib/RunUtils';
import { Apis, RunSortKeys, ListRequest } from '../lib/Apis';
import { ApiListRunsResponse, ApiRunDetail, ApiRun, ApiResourceType, RunMetricFormat, ApiRunMetric } from '../../src/apis/run';
import { Link, RouteComponentProps } from 'react-router-dom';
import { NodePhase, statusToIcon } from './Status';
import { RoutePage, RouteParams } from '../components/Router';
import { Workflow } from '../../../frontend/third_party/argo-ui/argo_template';
import { commonCss, color } from '../Css';
import { getRunTime, formatDateString, logger, errorToMessage } from '../lib/Utils';
import { orderBy } from 'lodash';
import { stylesheet } from 'typestyle';

const css = stylesheet({
  metricContainer: {
    background: '#f6f7f9',
    marginRight: 10,
  },
  metricFill: {
    background: '#cbf0f8',
    boxSizing: 'border-box',
    color: '#202124',
    fontFamily: 'Roboto',
    fontSize: 13,
  },
});

interface ExperimentInfo {
  displayName: string;
  id: string;
}

interface PipelineInfo {
  displayName: string;
  id: string;
}

interface DisplayRun {
  experiment?: ExperimentInfo;
  metadata: ApiRun;
  pipeline?: PipelineInfo;
  workflow?: Workflow;
  error?: string;
}

interface DisplayMetric {
  metadata: MetricMetadata;
  metric?: ApiRunMetric;
}

interface MetricMetadata {
  count: number;
  maxValue: number;
  minValue: number;
  name: string;
}

export interface RunListProps extends RouteComponentProps {
  disablePaging?: boolean;
  disableSelection?: boolean;
  disableSorting?: boolean;
  experimentIdMask?: string;
  runIdListMask?: string[];
  onError: (message: string, error: Error) => void;
  selectedIds?: string[];
  onSelectionChange?: (selectedRunIds: string[]) => void;
}

interface RunListState {
  metrics: MetricMetadata[];
  runs: DisplayRun[];
  sortBy: string;
}

class RunList extends React.Component<RunListProps, RunListState> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      metrics: [],
      runs: [],
      sortBy: RunSortKeys.CREATED_AT,
    };
  }

  public render(): JSX.Element {
    // Only show the two most prevalent metrics
    const metricMetadata: MetricMetadata[] = this.state.metrics.slice(0, 2);
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer.bind(this),
        flex: 2,
        label: 'Run name',
        sortKey: RunSortKeys.NAME,
      },
      { customRenderer: this._statusCustomRenderer.bind(this), flex: 0.5, label: 'Status' },
      { label: 'Duration', flex: 0.5 },
      { customRenderer: this._experimentCustomRenderer.bind(this), label: 'Experiment', flex: 1 },
      { customRenderer: this._pipelineCustomRenderer.bind(this), label: 'Pipeline', flex: 1 },
      { label: 'Start time', flex: 1, sortKey: RunSortKeys.CREATED_AT },
    ];

    if (metricMetadata.length) {
      // This is a column of empty cells with a left border to separate the metrics from the other
      // columns.
      columns.push({
        customRenderer: this._metricBufferCustomRenderer.bind(this),
        flex: 0.1,
        label: '',
      });

      columns.push(...metricMetadata.map((metadata) => {
        return {
          customRenderer: this._metricCustomRenderer.bind(this),
          flex: 1,
          label: metadata.name!
        };
      }));
    }

    const rows: Row[] = this.state.runs.map(r => {
      const displayMetrics = metricMetadata.map(metadata => {
        const displayMetric: DisplayMetric = { metadata };
        if (r.metadata.metrics) {
          const foundMetric = r.metadata.metrics.find(m => m.name === metadata.name);
          if (foundMetric && foundMetric.number_value !== undefined) {
            displayMetric.metric = foundMetric;
          }
        }
        return displayMetric;
      });
      const row = {
        error: r.error,
        id: r.metadata.id!,
        otherFields: [
          r.metadata!.name,
          r.metadata.status || '-',
          getRunTime(r.workflow),
          r.experiment,
          r.pipeline,
          formatDateString(r.metadata.created_at),
        ]
      };
      if (displayMetrics.length) {
        row.otherFields.push(''); // Metric buffer column
        row.otherFields.push(...displayMetrics as any);
      }
      return row;
    });

    return (<div>
      <CustomTable columns={columns} rows={rows} selectedIds={this.props.selectedIds}
        initialSortColumn={this.state.sortBy} ref={this._tableRef}
        updateSelection={this.props.onSelectionChange} reload={this._loadRuns.bind(this)}
        disablePaging={this.props.disablePaging} disableSorting={this.props.disableSorting}
        disableSelection={this.props.disableSelection}
        emptyMessage={`No runs found${this.props.experimentIdMask ? ' for this experiment' : ''}.`}
      />
    </div>);
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      await this._tableRef.current.reload();
    }
  }

  private _metricBufferCustomRenderer(): JSX.Element {
    return <div style={{ borderLeft: `1px solid ${color.divider}`, padding: '20px 0' }} />;
  }

  private _metricCustomRenderer(displayMetric: DisplayMetric): JSX.Element {
    if (!displayMetric.metric || !displayMetric.metric.number_value) {
      return <div />;
    }

    const leftSpace = 6;
    let displayString = '';
    let width = '';

    if (displayMetric.metric.format === RunMetricFormat.PERCENTAGE) {
      displayString = (displayMetric.metric.number_value * 100).toFixed(3) + '%';
      width = `calc(${displayString})`;
    } else {
      displayString = displayMetric.metric.number_value.toFixed(3);

      if (displayMetric.metadata.maxValue === 0 && displayMetric.metadata.minValue === 0) {
        return <div style={{ paddingLeft: leftSpace }}>{displayString}</div>;
      }

      if (displayMetric.metric.number_value - displayMetric.metadata.minValue < 0) {
        logger.error(`Run ${arguments[1]}'s metric ${displayMetric.metadata.name}'s value:`
          + ` ${displayMetric.metric.number_value}) was lower than the supposed minimum of`
          + ` ${displayMetric.metadata.minValue})`);
        return <div style={{ paddingLeft: leftSpace }}>{displayString}</div>;
      }

      if (displayMetric.metadata.maxValue - displayMetric.metric.number_value < 0) {
        logger.error(`Run ${arguments[1]}'s metric ${displayMetric.metadata.name}'s value:`
          + ` ${displayMetric.metric.number_value}) was greater than the supposed maximum of`
          + ` ${displayMetric.metadata.maxValue})`);
        return <div style={{ paddingLeft: leftSpace }}>{displayString}</div>;
      }

      const barWidth =
        (displayMetric.metric.number_value - displayMetric.metadata.minValue)
        / (displayMetric.metadata.maxValue - displayMetric.metadata.minValue)
        * 100;

      width = `calc(${barWidth}%)`;
    }
    return (
      <div className={css.metricContainer} style={{ marginLeft: leftSpace }}>
        <div className={css.metricFill} style={{ textIndent: leftSpace, width }}>
          {displayString}
        </div>
      </div>
    );
  }

  private async _loadRuns(loadRequest: ListRequest): Promise<string> {
    if (Array.isArray(this.props.runIdListMask)) {
      return await this._loadSpecificRuns(this.props.runIdListMask);
    }
    return await this._loadAllRuns(loadRequest);
  }

  private async _loadSpecificRuns(runIdListMask: string[]): Promise<string> {
    await Promise.all(runIdListMask.map(async id => await Apis.runServiceApi.getRun(id)))
      .then(async (result) => {
        const displayRuns: DisplayRun[] = result.map(r => ({
          metadata: r.run!,
          workflow: JSON.parse(r.pipeline_runtime!.workflow_manifest || '{}'),
        }));

        await this._getAndSetPipelineNames(displayRuns);
        await this._getAndSetExperimentNames(displayRuns);

        this.setState({
          metrics: this._extractMetricMetadata(displayRuns),
          runs: displayRuns,
        });
      })
      .catch((err) => this.props.onError('Error: failed to fetch runs.', err));
    return '';
  }

  private async _loadAllRuns(request: ListRequest): Promise<string> {
    let response: ApiListRunsResponse;
    try {
      response = await Apis.runServiceApi.listRuns(
        request.pageToken,
        request.pageSize,
        request.sortBy,
        this.props.experimentIdMask ? ApiResourceType.EXPERIMENT.toString() : undefined,
        this.props.experimentIdMask,
      );
    } catch (err) {
      this.props.onError('Error: failed to fetch runs.', err);
      // No point in continuing if we couldn't retrieve any runs.
      return '';
    }

    const displayRuns: DisplayRun[] = (response.runs || []).map(r => ({ metadata: r }));

    // Fetch and set the workflow details
    await Promise.all(displayRuns.map(async displayRun => {
      let getRunResponse: ApiRunDetail;
      try {
        getRunResponse = await Apis.runServiceApi.getRun(displayRun.metadata!.id!);
        displayRun.workflow =
          JSON.parse(getRunResponse.pipeline_runtime!.workflow_manifest || '{}');
      } catch (err) {
        // This could be an API exception, or a JSON parse exception.
        displayRun.error = await errorToMessage(err);
      }
    }));

    await this._getAndSetPipelineNames(displayRuns);
    await this._getAndSetExperimentNames(displayRuns);

    this.setState({
      metrics: this._extractMetricMetadata(displayRuns),
      runs: displayRuns,
      sortBy: request.sortBy!,
    });

    return response.next_page_token || '';
  }

  /**
   * For each DisplayRun, get its ApiRun and retrieve that ApiRun's Pipeline ID if it has one, then
   * use that Pipeline ID to fetch its associated Pipeline and attach that Pipeline's name to the
   * DisplayRun. If the ApiRun has no Pipeline ID, then the corresponding DisplayRun will show '-'.
   */
  private async _getAndSetPipelineNames(displayRuns: DisplayRun[]): Promise<DisplayRun[]> {
    return await Promise.all(
      displayRuns.map(async (displayRun) => {
        const pipelineId = RunUtils.getPipelineId(displayRun.metadata);
        if (pipelineId) {
          try {
            const pipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);
            displayRun.pipeline = { displayName: pipeline.name || '', id: pipelineId };
          } catch (err) {
            // This could be an API exception, or a JSON parse exception.
            displayRun.error = await errorToMessage(err);
          }
        }
        return displayRun;
      })
    );
  }

  private _pipelineCustomRenderer(pipelineInfo?: PipelineInfo): JSX.Element {
    // If the getPipeline call failed or a run has no pipeline, we display a placeholder.
    if (!pipelineInfo || !pipelineInfo.id) {
      return <div>-</div>;
    }
    return (
      <Link className={commonCss.link} onClick={(e) => e.stopPropagation()}
        to={RoutePage.PIPELINE_DETAILS.replace(':' + RouteParams.pipelineId, pipelineInfo.id)}>
        {pipelineInfo.displayName}
      </Link>
    );
  }

  /**
   * For each DisplayRun, get its ApiRun and retrieve that ApiRun's Experiment ID if it has one,
   * then use that Experiment ID to fetch its associated Experiment and attach that Experiment's
   * name to the DisplayRun. If the ApiRun has no Experiment ID, then the corresponding DisplayRun
   * will show '-'.
   */
  private async _getAndSetExperimentNames(displayRuns: DisplayRun[]): Promise<DisplayRun[]> {
    return await Promise.all(
      displayRuns.map(async (displayRun) => {
        const experimentId = RunUtils.getFirstExperimentReferenceId(displayRun.metadata);
        if (experimentId) {
          try {
            // TODO: Experiment could be an optional field in state since whenever the RunList is
            // created from the ExperimentDetails page, we already have the experiment (and will)
            // be fetching the same one over and over here.
            const experiment = await Apis.experimentServiceApi.getExperiment(experimentId);
            displayRun.experiment = { displayName: experiment.name || '', id: experimentId };
          } catch (err) {
            // This could be an API exception, or a JSON parse exception.
            displayRun.error = await errorToMessage(err);
          }
        }
        return displayRun;
      })
    );
  }

  private _experimentCustomRenderer(experimentInfo?: ExperimentInfo): JSX.Element {
    // If the getExperiment call failed or a run has no experiment, we display a placeholder.
    if (!experimentInfo || !experimentInfo.id) {
      return <div>-</div>;
    }
    return (
      <Link className={commonCss.link} onClick={(e) => e.stopPropagation()}
        to={RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experimentInfo.id)}>
        {experimentInfo.displayName}
      </Link>
    );
  }

  private _nameCustomRenderer(value: string, id: string): JSX.Element {
    return <Link className={commonCss.link} onClick={(e) => e.stopPropagation()}
      to={RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, id)}>{value}</Link>;
  }

  private _statusCustomRenderer(status: NodePhase): JSX.Element {
    return statusToIcon(status);
  }

  private _extractMetricMetadata(runs: DisplayRun[]): MetricMetadata[] {
    const metrics = Array.from(
      runs.reduce((metricMetadatas, run) => {
        if (!run.metadata || !run.metadata.metrics) {
          return metricMetadatas;
        }
        run.metadata.metrics.forEach((metric) => {
          if (!metric.name || metric.number_value === undefined || isNaN(metric.number_value)) {
            return;
          }

          let metricMetadata = metricMetadatas.get(metric.name);
          if (!metricMetadata) {
            metricMetadata = {
              count: 0,
              maxValue: Number.MIN_VALUE,
              minValue: Number.MAX_VALUE,
              name: metric.name,
            };
            metricMetadatas.set(metricMetadata.name, metricMetadata);
          }
          metricMetadata.count++;
          metricMetadata.minValue = Math.min(metricMetadata.minValue, metric.number_value);
          metricMetadata.maxValue = Math.max(metricMetadata.maxValue, metric.number_value);
        });
        return metricMetadatas;
      }, new Map<string, MetricMetadata>()).values()
    );
    return orderBy(metrics, ['count', 'name'], ['desc', 'asc']);
  }
}

export default RunList;
