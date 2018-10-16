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

import * as Apis from '../lib/Apis';
import * as React from 'react';
import CustomTable, { Column, Row } from '../components/CustomTable';
import { Link } from 'react-router-dom';
import { NodePhase, statusToIcon } from './Status';
import { RouteComponentProps } from 'react-router';
import { RoutePage, RouteParams } from '../components/Router';
import { Workflow } from '../../../frontend/third_party/argo-ui/argo_template';
import { apiListRunsResponse, apiRun, apiRunDetail } from '../../../frontend/src/api/run';
import { commonCss } from '../Css';
import { getRunTime, getLastInStatusList } from '../lib/Utils';

interface DisplayRun {
  metadata: apiRun;
  workflow?: Workflow;
  error?: string;
}

interface RunListProp extends RouteComponentProps {
  disablePaging?: boolean;
  disableSelection?: boolean;
  jobIdMask?: string;
  runIdListMask?: string[];
  handleError: (message: string, error: Error) => void;
  selectedIds?: string[];
  updateSelection?: (selectedRunIds: string[]) => void;
}

interface RunListState {
  orderAscending: boolean;
  pageSize: number;
  pageToken: string;
  runs: DisplayRun[];
  sortBy: string;
}

class RunList extends React.Component<RunListProp, RunListState> {

  constructor(props: any) {
    super(props);

    this.state = {
      orderAscending: true,
      pageSize: 10,
      pageToken: '',
      runs: [],
      sortBy: Apis.RunSortKeys.CREATED_AT,
    };
  }

  public render() {
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer.bind(this),
        flex: 2,
        label: 'Run name',
        sortKey: Apis.RunSortKeys.NAME,
      },
      { customRenderer: this._statusCustomRenderer.bind(this), flex: .5, label: 'Status' },
      { label: 'Duration', flex: 1 },
      { label: 'Start time', flex: 2, sortKey: Apis.RunSortKeys.CREATED_AT },
    ];

    const rows: Row[] = this.state.runs.map(r => {
      return {
        error: r.error,
        id: r.metadata.id!,
        otherFields: [
          r.metadata!.name,
          getLastInStatusList(r.metadata.status || '') || '-',
          getRunTime(r.workflow),
          r.metadata!.created_at!.toLocaleString(),
        ],
      };
    });

    return (
      <div>
        <CustomTable columns={columns} rows={rows} orderAscending={this.state.orderAscending}
          pageSize={this.state.pageSize} sortBy={this.state.sortBy}
          updateSelection={this.props.updateSelection} selectedIds={this.props.selectedIds}
          disablePaging={this.props.disablePaging} reload={this._loadRuns.bind(this)}
          disableSelection={this.props.disableSelection}
          emptyMessage={`No runs found${this.props.jobIdMask ? ' for this job' : ''}.`} />
      </div>
    );
  }

  public async refresh() {
    await this._loadRuns();
  }

  private async _loadRuns(loadRequest?: Apis.BaseListRequest): Promise<string> {
    if (this.props.runIdListMask && this.props.runIdListMask.length) {
      return await this._loadSpecificRuns(this.props.runIdListMask);
    }
    return await this._loadAllRuns(loadRequest);
  }

  private async _loadSpecificRuns(runIdListMask: string[]): Promise<string> {
    await Promise.all(runIdListMask.map(async id => await Apis.getRun(id)))
      .then((result) => {
        const displayRuns: DisplayRun[] = result.map(r => ({
          metadata: r.run!,
          workflow: JSON.parse(r.workflow || '{}'),
        }));

        this.setState({
          runs: displayRuns,
        });
      })
      .catch((err) =>
        this.props.handleError(
          'Error: failed to fetch runs. Click Details for more information.', err));
    return '';
  }

  private async _loadAllRuns(loadRequest?: Apis.BaseListRequest): Promise<string> {
    // Override the current state with incoming request
    const request: Apis.ListRunsRequest = Object.assign({
      orderAscending: this.state.orderAscending,
      pageSize: this.state.pageSize,
      pageToken: this.state.pageToken,
      sortBy: this.state.sortBy,
    }, loadRequest);

    request.jobId = this.props.jobIdMask;

    let response: apiListRunsResponse;
    try {
      response = await Apis.listRuns(request);
    } catch (err) {
      this.props.handleError(
        'Error: failed to fetch runs. Click Details for more information.', err);
      // No point in continuing if we couldn't retrieve any runs.
      return '';
    }

    const displayRuns: DisplayRun[] = (response.runs || []).map(r => ({ metadata: r }));

    // Fetch and set the workflow details
    await Promise.all(displayRuns.map(async displayRun => {
      let getRunResponse: apiRunDetail;
      try {
        getRunResponse = await Apis.getRun(displayRun.metadata!.id!);
        displayRun.workflow = JSON.parse(getRunResponse.workflow || '{}');
      } catch (err) {
        // This could be an API exception, or a JSON parse exception.
        displayRun.error = err.message;
      }
    }));
    this.setState({
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
}

export default RunList;
