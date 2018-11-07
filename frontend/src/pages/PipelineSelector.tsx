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
import { Apis, ListRequest, PipelineSortKeys } from '../lib/Apis';
import { RouteComponentProps } from 'react-router-dom';
import { logger, formatDateString, errorToMessage } from '../lib/Utils';
import { ApiPipeline } from '../apis/pipeline';
import { DialogProps } from '../components/Router';

interface PipelineSelectorProps extends RouteComponentProps {
  pipelineSelectionChanged: (selectedPipelineId: string) => void;
  updateDialog: (dialogProps: DialogProps) => void;
}

interface PipelineSelectorState {
  pipelines: ApiPipeline[];
  selectedIds: string[];
  toolbarActions: ToolbarActionConfig[];
}

class PipelineSelector extends React.Component<PipelineSelectorProps, PipelineSelectorState> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      pipelines: [],
      selectedIds: [],
      toolbarActions: [],
    };
  }

  public render() {
    const { pipelines, selectedIds, toolbarActions } = this.state;

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

    return (
      <React.Fragment>
        <Toolbar actions={toolbarActions} breadcrumbs={[{ displayName: 'Choose a pipeline', href: '' }]} />
        <CustomTable columns={columns} rows={rows} selectedIds={selectedIds} useRadioButtons={true}
          updateSelection={ids => { this._pipelineSelectionChanged(ids); this.setState({ selectedIds: ids }); }}
          initialSortColumn={PipelineSortKeys.CREATED_AT} ref={this._tableRef}
          reload={this._loadPipelines.bind(this)} emptyMessage={'No pipelines found. Upload a pipeline and then try again.'} />
      </React.Fragment>
    );
  }

  public async refresh() {
    if (this._tableRef.current) {
      this._tableRef.current.reload();
    }
  }

  private _pipelineSelectionChanged(selectedIds: string[]): void {
    if (!Array.isArray(selectedIds) || selectedIds.length !== 1) {
      logger.error(`${selectedIds.length} pipelines were selected somehow`, selectedIds);
      return;
    }
    this.props.pipelineSelectionChanged(selectedIds[0]);
  }

  private async _loadPipelines(request: ListRequest): Promise<string> {
    let pipelines: ApiPipeline[] = [];
    let nextPageToken = '';
    try {
      const response = await Apis.pipelineServiceApi.listPipelines(
        request.pageToken, request.pageSize, request.sortBy);
      pipelines = response.pipelines || [];
      nextPageToken = response.next_page_token || '';
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      this.props.updateDialog({
        buttons: [{ text: 'Dismiss' }],
        content: 'List pipelines request failed with:\n' + errorMessage,
        title: 'Error retrieving pipelines',
      });
      logger.error('Could not get list of pipelines', errorMessage);
    }

    this.setState({ pipelines });
    return nextPageToken;
  }
}

export default PipelineSelector;
