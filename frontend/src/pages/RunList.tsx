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

import { Apis, RunSortKeys, BaseListRequest, ListRunsRequest } from '../lib/Apis';
import * as React from 'react';
import CustomTable, { Column, Row } from '../components/CustomTable';
import { Link, RouteComponentProps } from 'react-router-dom';
import { NodePhase, statusToIcon } from './Status';
import { RoutePage, RouteParams } from '../components/Router';
import { Workflow } from '../../../frontend/third_party/argo-ui/argo_template';
import { commonCss } from '../Css';
import { getRunTime, getLastInStatusList } from '../lib/Utils';
import { ApiListRunsResponse, ApiRunDetail, ApiRun, RunMetricFormat } from '../../src/apis/run';
import { orderBy } from 'lodash';

interface DisplayRun {
  metadata: ApiRun;
  workflow?: Workflow;
  error?: string;
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
  jobIdMask?: string;
  runIdListMask?: string[];
  onError: (message: string, error: Error) => void;
  selectedIds?: string[];
  onSelectionChange?: (selectedRunIds: string[]) => void;
}

interface RunListState {
  metrics: MetricMetadata[];
  orderAscending: boolean;
  pageSize: number;
  pageToken: string;
  runs: DisplayRun[];
  sortBy: string;
}

class RunList extends React.Component<RunListProps, RunListState> {

  constructor(props: any) {
    super(props);

    this.state = {
      metrics: [],
      orderAscending: true,
      pageSize: 10,
      pageToken: '',
      runs: [],
      sortBy: RunSortKeys.CREATED_AT,
    };
  }

  public render() {
    const displayMetrics: MetricMetadata[] = this.state.metrics.slice(0, 2);
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer.bind(this),
        flex: 2,
        label: 'Run name',
        sortKey: RunSortKeys.NAME,
      },
      { customRenderer: this._statusCustomRenderer.bind(this), flex: .5, label: 'Status' },
      { label: 'Duration', flex: 1 },
      { label: 'Start time', flex: 2, sortKey: RunSortKeys.CREATED_AT },
    ].concat(displayMetrics.map((metric) => {
      return { flex: 1, label: metric.name };
    }));


    const rows: Row[] = this.state.runs.map(r => {
      const metricFields = displayMetrics.map(dm => {
        if (!r.metadata.metrics) {
          return '';
        }
        const foundMetric = r.metadata.metrics.find(m => m.name === dm.name);
        if (!foundMetric || foundMetric.number_value === undefined) {
          return '';
        }
        if (foundMetric.format === RunMetricFormat.PERCENTAGE) {
          return (foundMetric.number_value * 100).toFixed(3) + '%';
        }
        return foundMetric.number_value.toFixed(3);
      });
      return {
        error: r.error,
        id: r.metadata.id!,
        otherFields: [
          r.metadata!.name,
          getLastInStatusList(r.metadata.status || '') || '-',
          getRunTime(r.workflow),
          this._createdAtToString(r.metadata),
        ].concat(metricFields),
      };
    });

    return (
      <div>
        <CustomTable columns={columns} rows={rows} orderAscending={this.state.orderAscending}
          pageSize={this.state.pageSize} sortBy={this.state.sortBy}
          updateSelection={this.props.onSelectionChange} selectedIds={this.props.selectedIds}
          disablePaging={this.props.disablePaging} reload={this._loadRuns.bind(this)}
          disableSelection={this.props.disableSelection} disableSorting={this.props.disableSorting}
          emptyMessage={`No runs found${this.props.jobIdMask ? ' for this job' : ''}.`} />
      </div>
    );
  }

  public async refresh() {
    await this._loadRuns();
  }

  private async _loadRuns(loadRequest?: BaseListRequest): Promise<string> {
    if (Array.isArray(this.props.runIdListMask)) {
      return await this._loadSpecificRuns(this.props.runIdListMask);
    }
    return await this._loadAllRuns(loadRequest);
  }

  private _createdAtToString(run?: ApiRun) {
    if (run && run.created_at && run.created_at.toLocaleString) {
      return run.created_at.toLocaleString();
    } else {
      return '';
    }
  }

  private async _loadSpecificRuns(runIdListMask: string[]): Promise<string> {
    await Promise.all(runIdListMask.map(async id => await Apis.runServiceApi.getRunV2(id)))
      .then((result) => {
        const displayRuns: DisplayRun[] = result.map(r => ({
          metadata: r.run!,
          workflow: JSON.parse(r.workflow || '{}'),
        }));

        this.setState({
          metrics: this._extractMetricMetadata(displayRuns),
          runs: displayRuns,
        });
      })
      .catch((err) =>
        this.props.onError(
          'Error: failed to fetch runs.', err));
    return '';
  }

  private async _loadAllRuns(loadRequest?: BaseListRequest): Promise<string> {
    // Override the current state with incoming request
    const request: ListRunsRequest = Object.assign({
      orderAscending: this.state.orderAscending,
      pageSize: this.state.pageSize,
      pageToken: this.state.pageToken,
      sortBy: this.state.sortBy,
    }, loadRequest);

    request.jobId = this.props.jobIdMask;

    let response: ApiListRunsResponse;
    try {
      if (request.jobId) {
        response = await Apis.jobServiceApi.listJobRuns(
          request.jobId,
          request.pageToken,
          request.pageSize,
          request.sortBy ? request.sortBy + (request.orderAscending ? ' asc' : ' desc') : ''
        );
      } else {
        response = await Apis.runServiceApi.listRuns(
          request.pageToken,
          request.pageSize,
          request.sortBy ? request.sortBy + (request.orderAscending ? ' asc' : ' desc') : ''
        );
      }
    } catch (err) {
      this.props.onError(
        'Error: failed to fetch runs.', err);
      // No point in continuing if we couldn't retrieve any runs.
      return '';
    }

    const displayRuns: DisplayRun[] = (response.runs || []).map(r => ({ metadata: r }));

    // Fetch and set the workflow details
    await Promise.all(displayRuns.map(async displayRun => {
      let getRunResponse: ApiRunDetail;
      try {
        getRunResponse = await Apis.runServiceApi.getRunV2(displayRun.metadata!.id!);
        displayRun.workflow = JSON.parse(getRunResponse.workflow || '{}');
      } catch (err) {
        // This could be an API exception, or a JSON parse exception.
        displayRun.error = err.message;
      }
    }));
    this.setState({
      metrics: this._extractMetricMetadata(displayRuns),
      orderAscending: request.orderAscending!,
      pageSize: request.pageSize!,
      pageToken: request.pageToken!,
      runs: displayRuns,
      sortBy: request.sortBy!,
    });

    return response.next_page_token || '';
  }

  private _nameCustomRenderer(value: string, id: string) {
    return <Link className={commonCss.link} onClick={(e) => e.stopPropagation()}
      to={RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, id)}>{value}</Link>;
  }

  private _statusCustomRenderer(status: NodePhase) {
    return statusToIcon(status);
  }

  private _extractMetricMetadata(runs: DisplayRun[]) {
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
    return orderBy(metrics, 'count', 'desc');
  }
}

export default RunList;
