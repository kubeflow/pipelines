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
import AddIcon from '@material-ui/icons/Add';
import CustomTable, { Column, Row } from '../components/CustomTable';
import UploadPipelineDialog from '../components/UploadPipelineDialog';
import produce from 'immer';
import { ApiPipeline, ApiListPipelinesResponse } from '../apis/pipeline';
import { Apis, PipelineSortKeys, ListRequest } from '../lib/Apis';
import { Link } from 'react-router-dom';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { formatDateString, errorToMessage } from '../lib/Utils';

interface PipelineListState {
  pipelines: ApiPipeline[];
  selectedIds: string[];
  uploadDialogOpen: boolean;
}

class PipelineList extends Page<{}, PipelineListState> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      pipelines: [],
      selectedIds: [],
      uploadDialogOpen: false,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [{
        action: () => this.setStateSafe({ uploadDialogOpen: true }),
        icon: AddIcon,
        id: 'uploadBtn',
        outlined: true,
        title: 'Upload pipeline',
        tooltip: 'Upload pipeline',
      }, {
        action: () => this.refresh(),
        id: 'refreshBtn',
        title: 'Refresh',
        tooltip: 'Refresh',
      }, {
        action: () => this.props.updateDialog({
          buttons: [
            { onClick: async () => await this._deleteDialogClosed(true), text: 'Delete' },
            { onClick: async () => await this._deleteDialogClosed(false), text: 'Cancel' },
          ],
          onClose: async () => await this._deleteDialogClosed(false),
          title: `Delete ${this.state.selectedIds.length} pipeline${this.state.selectedIds.length === 1 ? '' : 's'}?`,
        }),
        disabled: true,
        disabledTitle: 'Select at least one pipeline to delete',
        id: 'deleteBtn',
        title: 'Delete',
        tooltip: 'Delete',
      }],
      breadcrumbs: [{ displayName: 'Pipelines', href: RoutePage.PIPELINES }],
    };
  }

  public render(): JSX.Element {
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer.bind(this),
        flex: 1,
        label: 'Pipeline name',
        sortKey: PipelineSortKeys.NAME,
      },
      { label: 'Description', flex: 3 },
      { label: 'Uploaded on', sortKey: PipelineSortKeys.CREATED_AT, flex: 1 },
    ];

    const rows: Row[] = this.state.pipelines.map((p) => {
      return {
        id: p.id!,
        otherFields: [p.name!, p.description!, formatDateString(p.created_at!)],
      };
    });

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <CustomTable ref={this._tableRef} columns={columns} rows={rows} initialSortColumn={PipelineSortKeys.CREATED_AT}
          updateSelection={this._selectionChanged.bind(this)} selectedIds={this.state.selectedIds}
          reload={this._reload.bind(this)}
          emptyMessage='No pipelines found. Click "Upload pipeline" to start.' />

        <UploadPipelineDialog open={this.state.uploadDialogOpen}
          onClose={this._uploadDialogClosed.bind(this)} />
      </div>
    );
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      await this._tableRef.current.reload();
    }
  }

  private async _reload(request: ListRequest): Promise<string> {
    let response: ApiListPipelinesResponse | null = null;
    try {
      response = await Apis.pipelineServiceApi.listPipelines(
        request.pageToken, request.pageSize, request.sortBy);
      this.clearBanner();
    } catch (err) {
      await this.showPageError('Error: failed to retrieve list of pipelines.', err);
    }

    this.setStateSafe({ pipelines: response ? response.pipelines || [] : [] });

    return response ? response.next_page_token || '' : '';
  }

  private _nameCustomRenderer(value: string, id: string): React.ReactElement<Link> {
    return (
      <Link onClick={(e) => e.stopPropagation()}
        className={commonCss.link}
        to={RoutePage.PIPELINE_DETAILS.replace(':' + RouteParams.pipelineId, id)}>{value}
      </Link>
    );
  }

  private _selectionChanged(selectedIds: string[]): void {
    const toolbarActions = produce(this.props.toolbarProps.actions, draft => {
      // Delete pipeline
      draft[2].disabled = selectedIds.length < 1;
    });
    this.props.updateToolbar({ breadcrumbs: this.props.toolbarProps.breadcrumbs, actions: toolbarActions });
    this.setStateSafe({ selectedIds });
  }

  private async _deleteDialogClosed(deleteConfirmed: boolean): Promise<void> {
    if (deleteConfirmed) {
      const unsuccessfulDeleteIds: string[] = [];
      const errorMessages: string[] = [];
      // TODO: Show spinner during wait.
      await Promise.all(this.state.selectedIds.map(async (id) => {
        try {
          await Apis.pipelineServiceApi.deletePipeline(id);
        } catch (err) {
          unsuccessfulDeleteIds.push(id);
          const pipeline = this.state.pipelines.find((p) => p.id === id);
          const errorMessage = await errorToMessage(err);
          errorMessages.push(
            `Deleting pipeline${pipeline ? ': ' + pipeline.name : ''} failed with error: "${errorMessage}"`);
        }
      }));

      const successfulDeletes = this.state.selectedIds.length - unsuccessfulDeleteIds.length;
      if (successfulDeletes > 0) {
        this.props.updateSnackbar({
          message: `Successfully deleted ${successfulDeletes} pipeline${successfulDeletes === 1 ? '' : 's'}!`,
          open: true,
        });
        this.refresh();
      }

      if (unsuccessfulDeleteIds.length > 0) {
        this.showErrorDialog(
          `Failed to delete ${unsuccessfulDeleteIds.length} pipeline${unsuccessfulDeleteIds.length === 1 ? '' : 's'}`,
          errorMessages.join('\n\n'));
      }

      this._selectionChanged(unsuccessfulDeleteIds);
    }
  }

  private async _uploadDialogClosed(name: string, file: File | null, description?: string): Promise<boolean> {
    if (!!file) {
      try {
        await Apis.uploadPipeline(name, file);
        this.setStateSafe({ uploadDialogOpen: false });
        this.refresh();
        return true;
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        this.showErrorDialog('Failed to upload pipeline', errorMessage);
        return false;
      }
    } else {
      this.setStateSafe({ uploadDialogOpen: false });
      return false;
    }
  }
}

export default PipelineList;
