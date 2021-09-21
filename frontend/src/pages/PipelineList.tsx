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

import Tooltip from '@material-ui/core/Tooltip';
import produce from 'immer';
import * as React from 'react';
import { Link } from 'react-router-dom';
import { classes } from 'typestyle';
import { ApiListPipelinesResponse, ApiPipeline } from '../apis/pipeline';
import CustomTable, {
  Column,
  CustomRendererProps,
  ExpandState,
  Row,
} from '../components/CustomTable';
import { Description } from '../components/Description';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import UploadPipelineDialog, { ImportMethod } from '../components/UploadPipelineDialog';
import { commonCss, padding } from '../Css';
import { Apis, ListRequest, PipelineSortKeys } from '../lib/Apis';
import Buttons, { ButtonKeys } from '../lib/Buttons';
import { errorToMessage, formatDateString } from '../lib/Utils';
import { Page } from './Page';
import PipelineVersionList from './PipelineVersionList';

interface DisplayPipeline extends ApiPipeline {
  expandState?: ExpandState;
}

interface PipelineListState {
  displayPipelines: DisplayPipeline[];
  selectedIds: string[];
  uploadDialogOpen: boolean;

  // selectedVersionIds is a map from string to string array.
  // For each pipeline, there is a list of selected version ids.
  selectedVersionIds: { [pipelineId: string]: string[] };
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
      displayPipelines: [],
      selectedIds: [],
      uploadDialogOpen: false,

      selectedVersionIds: {},
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons
        .newPipelineVersion('Upload pipeline')
        .refresh(this.refresh.bind(this))
        .deletePipelinesAndPipelineVersions(
          () => this.state.selectedIds,
          () => this.state.selectedVersionIds,
          (pipelineId, ids) => this._selectionChanged(pipelineId, ids),
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

    const rows: Row[] = this.state.displayPipelines.map(p => {
      return {
        expandState: p.expandState,
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
          updateSelection={this._selectionChanged.bind(this, undefined)}
          selectedIds={this.state.selectedIds}
          reload={this._reload.bind(this)}
          toggleExpansion={this._toggleRowExpand.bind(this)}
          getExpandComponent={this._getExpandedPipelineComponent.bind(this)}
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

  private _toggleRowExpand(rowIndex: number): void {
    const displayPipelines = produce(this.state.displayPipelines, draft => {
      draft[rowIndex].expandState =
        draft[rowIndex].expandState === ExpandState.COLLAPSED
          ? ExpandState.EXPANDED
          : ExpandState.COLLAPSED;
    });

    this.setState({ displayPipelines });
  }

  private _getExpandedPipelineComponent(rowIndex: number): JSX.Element {
    const pipeline = this.state.displayPipelines[rowIndex];
    return (
      <PipelineVersionList
        pipelineId={pipeline.id}
        onError={() => null}
        {...this.props}
        selectedIds={this.state.selectedVersionIds[pipeline.id!] || []}
        noFilterBox={true}
        onSelectionChange={this._selectionChanged.bind(this, pipeline.id)}
        disableSorting={false}
        disablePaging={false}
      />
    );
  }

  private async _reload(request: ListRequest): Promise<string> {
    let response: ApiListPipelinesResponse | null = null;
    let displayPipelines: DisplayPipeline[];
    try {
      response = await Apis.pipelineServiceApi.listPipelines(
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
      );
      displayPipelines = response.pipelines || [];
      displayPipelines.forEach(exp => (exp.expandState = ExpandState.COLLAPSED));
      this.clearBanner();
    } catch (err) {
      await this.showPageError('Error: failed to retrieve list of pipelines.', err);
    }

    this.setStateSafe({ displayPipelines: (response && response.pipelines) || [] });

    return response ? response.next_page_token || '' : '';
  }

  private _nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    return (
      <Tooltip title={props.value || ''} enterDelay={300} placement='top-start'>
        <Link
          onClick={e => e.stopPropagation()}
          className={commonCss.link}
          to={RoutePage.PIPELINE_DETAILS_NO_VERSION.replace(':' + RouteParams.pipelineId, props.id)}
        >
          {props.value || ''}
        </Link>
      </Tooltip>
    );
  };

  // selection changes passed in via "selectedIds" can be
  // (1) changes of selected pipeline ids, and will be stored in "this.state.selectedIds" or
  // (2) changes of selected pipeline version ids, and will be stored in "selectedVersionIds" with key "pipelineId"
  private _selectionChanged(pipelineId: string | undefined, selectedIds: string[]): void {
    if (!!pipelineId) {
      // Update selected pipeline version ids.
      this.setStateSafe({
        selectedVersionIds: { ...this.state.selectedVersionIds, ...{ [pipelineId!]: selectedIds } },
      });
      const actions = this.props.toolbarProps.actions;
      actions[ButtonKeys.DELETE_RUN].disabled =
        this.state.selectedIds.length < 1 && selectedIds.length < 1;
      this.props.updateToolbar({ actions });
    } else {
      // Update selected pipeline ids.
      this.setStateSafe({ selectedIds });
      const selectedVersionIdsCt = this._deepCountDictionary(this.state.selectedVersionIds);
      const actions = this.props.toolbarProps.actions;
      actions[ButtonKeys.DELETE_RUN].disabled = selectedIds.length < 1 && selectedVersionIdsCt < 1;
      this.props.updateToolbar({ actions });
    }
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
        ? await Apis.uploadPipeline(name, description || '', file!)
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

  private _deepCountDictionary(dict: { [pipelineId: string]: string[] }): number {
    return Object.keys(dict).reduce((count, pipelineId) => count + dict[pipelineId].length, 0);
  }
}

export default PipelineList;
