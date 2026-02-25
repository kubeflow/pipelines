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
import immerProduce from 'immer';
import * as React from 'react';
import { Link } from 'react-router-dom';
import { classes } from 'typestyle';
import { V2beta1Pipeline, V2beta1ListPipelinesResponse } from 'src/apisv2beta1/pipeline';
import CustomTable, {
  Column,
  CustomRendererProps,
  ExpandState,
  Row,
} from 'src/components/CustomTable';
import { Description } from 'src/components/Description';
import { RoutePage, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { commonCss, padding } from 'src/Css';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { Apis, ListRequest, PipelineSortKeys } from 'src/lib/Apis';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import { Page } from './Page';
import PipelineVersionList from './PipelineVersionList';

interface DisplayPipeline extends V2beta1Pipeline {
  expandState?: ExpandState;
}

interface PipelineListState {
  displayPipelines: DisplayPipeline[];
  selectedIds: string[];

  // selectedVersionIds is a map from string to string array.
  // For each pipeline, there is a list of selected version ids.
  selectedVersionIds: { [pipelineId: string]: string[] };
}

const descriptionCustomRenderer: React.FC<CustomRendererProps<string>> = (
  props: CustomRendererProps<string>,
) => {
  return <Description description={props.value || ''} forceInline={true} />;
};

class PipelineList extends Page<{ namespace?: string }, PipelineListState> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      displayPipelines: [],
      selectedIds: [],
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
        sortKey: PipelineSortKeys.DISPLAY_NAME,
      },
      { label: 'Description', flex: 3, customRenderer: descriptionCustomRenderer },
      { label: 'Uploaded on', sortKey: PipelineSortKeys.CREATED_AT, flex: 1 },
    ];

    const rows: Row[] = this.state.displayPipelines.map(p => {
      return {
        expandState: p.expandState,
        id: p.pipeline_id!,
        otherFields: [
          { display_name: p.display_name, name: p.name },
          p.description!,
          formatDateString(p.created_at!),
        ] as any,
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
      </div>
    );
  }

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      await this._tableRef.current.reload();
    }
  }

  private _toggleRowExpand(rowIndex: number): void {
    const displayPipelines = immerProduce(this.state.displayPipelines, draft => {
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
        pipelineId={pipeline.pipeline_id}
        onError={() => null}
        {...this.props}
        selectedIds={this.state.selectedVersionIds[pipeline.pipeline_id!] || []}
        noFilterBox={true}
        onSelectionChange={this._selectionChanged.bind(this, pipeline.pipeline_id)}
        disableSorting={false}
        disablePaging={false}
      />
    );
  }

  private async _reload(request: ListRequest): Promise<string> {
    let response: V2beta1ListPipelinesResponse | null = null;
    let displayPipelines: DisplayPipeline[];
    try {
      response = await Apis.pipelineServiceApiV2.listPipelines(
        this.props.namespace,
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
      );
      displayPipelines = response.pipelines || [];
      displayPipelines.forEach(exp => (exp.expandState = ExpandState.COLLAPSED));
      this.clearBanner();
    } catch (err) {
      const error = err instanceof Error ? err : new Error(await errorToMessage(err));
      await this.showPageError('Error: failed to retrieve list of pipelines.', error);
    }

    this.setStateSafe({ displayPipelines: (response && response.pipelines) || [] });

    return response ? response.next_page_token || '' : '';
  }

  private _nameCustomRenderer: React.FC<
    CustomRendererProps<{ display_name?: string; name: string }>
  > = (props: CustomRendererProps<{ display_name?: string; name: string }>) => {
    return (
      <Tooltip title={'Name: ' + (props.value?.name || '')} enterDelay={300} placement='top-start'>
        <Link
          onClick={e => e.stopPropagation()}
          className={commonCss.link}
          to={RoutePage.PIPELINE_DETAILS_NO_VERSION.replace(':' + RouteParams.pipelineId, props.id)}
        >
          {props.value?.display_name || props.value?.name}
        </Link>
      </Tooltip>
    );
  };

  // selection changes passed in via "selectedIds" can be
  // (1) changes of selected pipeline ids, and will be stored in "this.state.selectedIds" or
  // (2) changes of selected pipeline version ids, and will be stored in "selectedVersionIds" with key "pipelineId"
  private _selectionChanged(pipelineId: string | undefined, selectedIds: string[]): void {
    if (pipelineId) {
      // Update selected pipeline version ids.
      this.setStateSafe({
        selectedVersionIds: {
          ...this.state.selectedVersionIds,
          ...{ [pipelineId!]: selectedIds || [] },
        },
      });
      const actions = this.props.toolbarProps.actions;
      actions[ButtonKeys.DELETE_RUN].disabled =
        (this.state.selectedIds?.length || 0) < 1 && (selectedIds?.length || 0) < 1;
      this.props.updateToolbar({ actions });
    } else {
      // Update selected pipeline ids.
      this.setStateSafe({ selectedIds: selectedIds || [] });
      const selectedVersionIdsCt = this._deepCountDictionary(this.state.selectedVersionIds || {});
      const actions = this.props.toolbarProps.actions;
      actions[ButtonKeys.DELETE_RUN].disabled =
        (selectedIds?.length || 0) < 1 && selectedVersionIdsCt < 1;
      this.props.updateToolbar({ actions });
    }
  }

  private _deepCountDictionary(dict: { [pipelineId: string]: string[] }): number {
    return Object.keys(dict).reduce((count, pipelineId) => count + dict[pipelineId].length, 0);
  }
}

export default PipelineList;
