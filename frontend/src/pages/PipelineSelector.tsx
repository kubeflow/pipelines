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
import Toolbar, { ToolbarActionConfig } from '../components/Toolbar';
import { Apis, BaseListRequest, ListPipelinesRequest, PipelineSortKeys } from '../lib/Apis';
import { RouteComponentProps } from 'react-router-dom';
import { logger, formatDateString } from '../lib/Utils';
import { ApiPipeline } from '../apis/pipeline';

interface PipelineSelectorProps extends RouteComponentProps {
  pipelineSectionChanged: (selectedPipelineId: string[]) => void;
}

interface PipelineSelectorState {
  busyIds: Set<string>;
  orderAscending: boolean;
  pageSize: number;
  pageToken: string;
  pipelines: ApiPipeline[];
  selectedIds: string[];
  sortBy: string;
  toolbarActions: ToolbarActionConfig[];
  viewIndex: number;
}

class PipelineSelector extends React.Component<PipelineSelectorProps, PipelineSelectorState> {

  constructor(props: any) {
    super(props);

    this.state = {
      busyIds: new Set(),
      orderAscending: false,
      pageSize: 10,
      pageToken: '',
      pipelines: [],
      selectedIds: [],
      sortBy: PipelineSortKeys.CREATED_AT,
      toolbarActions: [],
      viewIndex: 1,
    };
  }

  public render() {
    const { pipelines, orderAscending, pageSize, selectedIds, sortBy, toolbarActions } = this.state;

    const columns: Column[] = [
      { label: 'Pipeline name', flex: 1, sortKey: PipelineSortKeys.NAME },
      { label: 'Description', flex: 1.5 },
      { label: 'Uploaded on', flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
    ];

    const rows: Row[] = pipelines.map((p) => {
      return {
        error: p.error,
        id: p.id!,
        otherFields: [
          p.name,
          p.description,
          formatDateString(p.created_at),
        ],
      };
    });

    return (<React.Fragment>
      <Toolbar actions={toolbarActions} breadcrumbs={[{ displayName: 'Choose a pipeline', href: '' }]} />
      <CustomTable columns={columns} rows={rows} orderAscending={orderAscending}
        pageSize={pageSize} selectedIds={selectedIds} useRadioButtons={true}
        updateSelection={ids => { this.props.pipelineSectionChanged(ids); this.setState({ selectedIds: ids });}} sortBy={sortBy}
        reload={this._loadPipelines.bind(this)} emptyMessage={'No pipelines found. Upload a pipeline and then try again.'} />
    </React.Fragment>);
  }

  public async load() {
    await this._loadPipelines();
  }

  private async _loadPipelines(loadRequest?: BaseListRequest): Promise<string> {
    // Override the current state with incoming request
    const request: ListPipelinesRequest = Object.assign({
      orderAscending: this.state.orderAscending,
      pageSize: this.state.pageSize,
      pageToken: this.state.pageToken,
      sortBy: this.state.sortBy,
    }, loadRequest);

    let pipelines: ApiPipeline[] = [];
    let nextPageToken = '';
    try {
      const response = await Apis.pipelineServiceApi.listPipelines(
        request.pageToken,
        request.pageSize,
        request.sortBy ? request.sortBy + (request.orderAscending ? ' asc' : ' desc') : '',
      );
      pipelines = response.pipelines || [];
      nextPageToken = response.next_page_token || '';
    } catch (err) {
      // TODO: better error experience here
      logger.error('Could not get list of pipelines');
    }

    this.setState({
      orderAscending: request.orderAscending!,
      pageSize: request.pageSize!,
      pageToken: request.pageToken!,
      pipelines,
      sortBy: request.sortBy!,
    });

    return nextPageToken;
  }
}

export default PipelineSelector;
