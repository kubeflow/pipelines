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
import Buttons, { ButtonKeys } from '../lib/Buttons';
import CustomTable, { Column, Row, CustomRendererProps } from '../components/CustomTable';
import UploadPipelineDialog, { ImportMethod } from '../components/UploadPipelineDialog';
import { ApiPipeline, ApiListPipelinesResponse } from '../apis/pipeline';
import { Apis, PipelineSortKeys, ListRequest } from '../lib/Apis';
import { Link } from 'react-router-dom';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { formatDateString, errorToMessage } from '../lib/Utils';
import { Description } from '../components/Description';

interface PipelineListState {
  pipelines: ApiPipeline[];
  selectedIds: string[];
  uploadDialogOpen: boolean;
}

const descriptionCustomRenderer: React.FC<CustomRendererProps<string>> = (
  props: CustomRendererProps<string>,
) => {
  return <Description description={props.value || ''} forceInline={true} />;
};

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
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons
        .upload(() => this.setStateSafe({ uploadDialogOpen: true }))
        .refresh(this.refresh.bind(this))
        .delete(
          () => this.state.selectedIds,
          'pipeline',
          ids => this._selectionChanged(ids),
          false /* useCurrentResource */,
        )
        .getToolbarActionMap(),
      breadcrumbs: [],
      pageTitle: 'Pipelines',
    };
  }

  public render(): JSX.Element {
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 1,
        label: 'Pipeline name',
        sortKey: PipelineSortKeys.NAME,
      },
      { label: 'Description', flex: 3, customRenderer: descriptionCustomRenderer },
      { label: 'Uploaded on', sortKey: PipelineSortKeys.CREATED_AT, flex: 1 },
    ];

    const rows: Row[] = this.state.pipelines.map(p => {
      return {
        id: p.id!,
        otherFields: [p.name!, p.description!, formatDateString(p.created_at!)],
      };
    });

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <CustomTable
          ref={this._tableRef}
          columns={columns}
          rows={rows}
          initialSortColumn={PipelineSortKeys.CREATED_AT}
          updateSelection={this._selectionChanged.bind(this)}
          selectedIds={this.state.selectedIds}
          reload={this._reload.bind(this)}
          filterLabel='Filter pipelines'
          emptyMessage='No pipelines found. Click "Upload pipeline" to start.'
        />

        <UploadPipelineDialog
          open={this.state.uploadDialogOpen}
          onClose={this._uploadDialogClosed.bind(this)}
        />
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
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
      );
      this.clearBanner();
    } catch (err) {
      await this.showPageError('Error: failed to retrieve list of pipelines.', err);
    }

    this.setStateSafe({ pipelines: (response && response.pipelines) || [] });

    return response ? response.next_page_token || '' : '';
  }

  private _nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    return (
      <Link
        onClick={e => e.stopPropagation()}
        className={commonCss.link}
        to={RoutePage.PIPELINE_DETAILS.replace(':' + RouteParams.pipelineId, props.id)}
      >
        {props.value}
      </Link>
    );
  };

  private _selectionChanged(selectedIds: string[]): void {
    const actions = this.props.toolbarProps.actions;
    actions[ButtonKeys.DELETE_RUN].disabled = selectedIds.length < 1;
    this.props.updateToolbar({ actions });
    this.setStateSafe({ selectedIds });
  }

  private async _uploadDialogClosed(
    confirmed: boolean,
    name: string,
    file: File | null,
    url: string,
    method: ImportMethod,
    description?: string,
  ): Promise<boolean> {
    if (
      !confirmed ||
      (method === ImportMethod.LOCAL && !file) ||
      (method === ImportMethod.URL && !url)
    ) {
      this.setStateSafe({ uploadDialogOpen: false });
      return false;
    }

    try {
      method === ImportMethod.LOCAL
        ? await Apis.uploadPipeline(name, file!)
        : await Apis.pipelineServiceApi.createPipeline({ name, url: { pipeline_url: url } });
      this.setStateSafe({ uploadDialogOpen: false });
      this.refresh();
      return true;
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      this.showErrorDialog('Failed to upload pipeline', errorMessage);
      return false;
    }
  }
}

export default PipelineList;
