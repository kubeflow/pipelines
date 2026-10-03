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

import { produce as immerProduce } from 'immer';
import * as React from 'react';
import { Link } from 'react-router';
import { V2beta1Pipeline, V2beta1ListPipelinesResponse } from 'src/apisv2beta1/pipeline';
import CustomTable, {
  Column,
  CustomRendererProps,
  ExpandState,
  Row,
} from 'src/components/CustomTable';
import { Description } from 'src/components/Description';
import { RoutePage, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/lib/PageChromeTypes';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { Apis, ListRequest, PipelineSortKeys } from 'src/lib/Apis';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import { Page } from './Page';
import PipelineVersionList from './PipelineVersionList';
import { PipelineCards } from 'src/components/modernization/PipelineCards';

interface DisplayPipeline extends V2beta1Pipeline {
  expandState?: ExpandState;
}

interface PipelineListState {
  displayPipelines: DisplayPipeline[];
  selectedIds: string[];
  loadError: boolean;

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
  private _loadGeneration = 0;

  constructor(props: any) {
    super(props);

    this.state = {
      displayPipelines: [],
      loadError: false,
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

  public render(): React.JSX.Element {
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

    const rows: Row[] = this.state.displayPipelines.map((p) => {
      return {
        expandState: p.expandState,
        id: p.pipeline_id!,
        error: p.error?.message,
        otherFields: [
          { display_name: p.display_name, name: p.name },
          p.description!,
          formatDateString(p.created_at!),
        ] as any,
      };
    });

    return (
      <div className='kfp-pipelines-page'>
        <CustomTable
          ref={this._tableRef}
          renderTable={(table) => <PipelineCards table={table} />}
          errorMessage={this.state.loadError ? 'Unable to load pipelines. Try Refresh.' : undefined}
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
    const displayPipelines = immerProduce(this.state.displayPipelines, (draft) => {
      draft[rowIndex].expandState =
        draft[rowIndex].expandState === ExpandState.COLLAPSED
          ? ExpandState.EXPANDED
          : ExpandState.COLLAPSED;
    });

    this.setState({ displayPipelines });
  }

  private _getExpandedPipelineComponent(rowIndex: number): React.JSX.Element {
    const pipeline = this.state.displayPipelines[rowIndex];
    return (
      <PipelineVersionList
        pipelineId={pipeline.pipeline_id}
        onError={(message, error) => this.showPageError(message, error)}
        onLoadSuccess={() => this.clearBanner()}
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
    const generation = ++this._loadGeneration;
    try {
      const response: V2beta1ListPipelinesResponse = await Apis.pipelineServiceApiV2.listPipelines(
        this.props.namespace,
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
      );
      if (!this._isMounted || generation !== this._loadGeneration) return '';
      this.setStateSafe({
        displayPipelines: (response.pipelines || []).map((pipeline) => ({
          ...pipeline,
          expandState: ExpandState.COLLAPSED,
        })),
        loadError: false,
      });
      this.clearBanner();
      return response.next_page_token || '';
    } catch (err) {
      const error = err instanceof Error ? err : new Error(await errorToMessage(err));
      if (!this._isMounted || generation !== this._loadGeneration) return '';
      this.setStateSafe({ loadError: true });
      await this.showPageError('Error: failed to retrieve list of pipelines.', error);
      return '';
    }
  }

  private _nameCustomRenderer: React.FC<
    CustomRendererProps<{ display_name?: string; name: string }>
  > = (props) => (
    <Link
      onClick={(event) => event.stopPropagation()}
      title={'Name: ' + (props.value?.name || '')}
      to={RoutePage.PIPELINE_DETAILS_NO_VERSION.replace(
        ':' + RouteParams.pipelineId,
        encodeURIComponent(props.id),
      )}
    >
      {props.value?.display_name || props.value?.name}
    </Link>
  );

  // selection changes passed in via "selectedIds" can be
  // (1) changes of selected pipeline ids, and will be stored in "this.state.selectedIds" or
  // (2) changes of selected pipeline version ids, and will be stored in "selectedVersionIds" with key "pipelineId"
  private _selectionChanged(pipelineId: string | undefined, selectedIds: string[]): void {
    if (!this._isMounted) return;
    this.setState(
      (current) => ({
        selectedIds: pipelineId ? current.selectedIds : selectedIds || [],
        selectedVersionIds: pipelineId
          ? { ...current.selectedVersionIds, [pipelineId]: selectedIds || [] }
          : current.selectedVersionIds,
      }),
      () => {
        const actions = this.props.toolbarProps.actions;
        actions[ButtonKeys.DELETE_RUN].disabled =
          this.state.selectedIds.length === 0 &&
          this._deepCountDictionary(this.state.selectedVersionIds) === 0;
        this.props.updateToolbar({ actions });
      },
    );
  }

  private _deepCountDictionary(dict: { [pipelineId: string]: string[] }): number {
    return Object.keys(dict).reduce((count, pipelineId) => count + dict[pipelineId].length, 0);
  }
}

export default PipelineList;
