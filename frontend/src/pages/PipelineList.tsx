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

import * as React from 'react';
import { Link } from 'react-router';
import { V2Pipeline, V2ListPipelinesResponse } from 'src/apisv2/pipeline';
import CustomTable, { Column, CustomRendererProps, Row } from 'src/components/CustomTable';
import { Description } from 'src/components/Description';
import { RoutePageFactory } from 'src/components/Router';
import { ToolbarProps } from 'src/lib/PageChromeTypes';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { Apis, ListRequest, PipelineSortKeys } from 'src/lib/Apis';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import { Page } from './Page';
import { LatestPipelineVersion } from 'src/components/pipelines/LatestPipelineVersion';
import { PipelineCollection } from 'src/components/pipelines/PipelineCollection';
import { Button } from 'src/components/ui/button';
import {
  UploadedDateFilter,
  uploadedDatePredicates,
  type UploadedPeriod,
} from 'src/components/pipelines/UploadedDateFilter';

type DisplayPipeline = V2Pipeline;

interface PipelineListState {
  uploadedPeriod: UploadedPeriod;
  uploadedSince?: Date;
  displayPipelines: DisplayPipeline[];
  selectedIds: string[];
  loadError: boolean;
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
      uploadedPeriod: 'all',
      displayPipelines: [],
      loadError: false,
      selectedIds: [],
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
          () => ({}),
          (_pipelineId, ids) => this._selectionChanged(ids),
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
      { label: 'Description', width: '35%', flex: 3, customRenderer: descriptionCustomRenderer },
      {
        label: 'Latest version',
        width: 180,
        customRenderer: (props) => <LatestPipelineVersion {...props} />,
      },
      { label: 'Uploaded on', width: 190, sortKey: PipelineSortKeys.CREATED_AT, flex: 1 },
    ];

    const rows: Row[] = this.state.displayPipelines.map((p) => {
      return {
        id: p.pipeline_id!,
        error: p.error?.message,
        otherFields: [
          { display_name: p.display_name, name: p.name },
          p.description!,
          p.pipeline_id!,
          formatDateString(p.created_at!),
        ] as any,
      };
    });

    return (
      <div className='kfp-pipelines-page'>
        <CustomTable
          ref={this._tableRef}
          filterPredicates={uploadedDatePredicates(this.state.uploadedSince)}
          renderTable={(table) => (
            <PipelineCollection
              table={table}
              dateFilter={
                <UploadedDateFilter
                  value={this.state.uploadedPeriod}
                  onChange={this._setUploadedPeriod}
                />
              }
              activeFilters={
                this.state.uploadedPeriod !== 'all' && (
                  <div className='kfp-pipeline-active-filters'>
                    <Button
                      variant='secondary'
                      size='sm'
                      aria-label='Clear uploaded date filter'
                      onClick={() => this._setUploadedPeriod('all')}
                    >
                      Uploaded: last {this.state.uploadedPeriod} days ×
                    </Button>
                  </div>
                )
              }
            />
          )}
          errorMessage={this.state.loadError ? 'Unable to load pipelines. Try Refresh.' : undefined}
          columns={columns}
          rows={rows}
          initialSortColumn={PipelineSortKeys.CREATED_AT}
          updateSelection={this._selectionChanged.bind(this)}
          selectedIds={this.state.selectedIds}
          reload={this._reload.bind(this)}
          filterLabel='Search pipelines'
          emptyMessage={
            this.state.uploadedPeriod === 'all'
              ? 'No pipelines found. Click "Upload pipeline" to start.'
              : 'No pipelines match the uploaded date. Clear the date filter to see older pipelines.'
          }
        />
      </div>
    );
  }

  private _setUploadedPeriod = (uploadedPeriod: UploadedPeriod): void => {
    this.setState({
      uploadedPeriod,
      uploadedSince:
        uploadedPeriod === 'all'
          ? undefined
          : new Date(Date.now() - Number(uploadedPeriod) * 86400000),
    });
  };

  public async refresh(): Promise<void> {
    if (this._tableRef.current) {
      await this._tableRef.current.reload();
    }
  }

  private async _reload(request: ListRequest): Promise<string> {
    const generation = ++this._loadGeneration;
    try {
      const response: V2ListPipelinesResponse = await Apis.pipelineServiceApiV2.listPipelines(
        this.props.namespace,
        request.pageToken,
        request.pageSize,
        request.sortBy,
        request.filter,
      );
      if (!this._isMounted || generation !== this._loadGeneration) return '';
      this.setStateSafe({
        displayPipelines: response.pipelines || [],
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
      to={RoutePageFactory.pipelineDetails(props.id)}
    >
      {props.value?.display_name || props.value?.name}
    </Link>
  );

  private _selectionChanged(selectedIds: string[]): void {
    if (!this._isMounted) return;
    this.setState({ selectedIds: selectedIds || [] }, () => {
      const actions = this.props.toolbarProps.actions;
      actions[ButtonKeys.DELETE_RUN].disabled = this.state.selectedIds.length === 0;
      this.props.updateToolbar({ actions });
    });
  }
}

export default PipelineList;
