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

import { NavigationProps } from 'src/lib/Navigation';
import CustomTable, { Column, CustomRendererProps, Row } from 'src/components/CustomTable';
import * as React from 'react';
import { Link } from 'react-router';
import {
  V2beta1PipelineVersion,
  V2beta1ListPipelineVersionsResponse,
} from 'src/apisv2beta1/pipeline';
import { Description } from 'src/components/Description';
import { Apis, ListRequest, PipelineVersionSortKeys } from 'src/lib/Apis';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { RoutePage, RouteParams } from 'src/components/Router';
import { ResourceTable } from 'src/components/tables/ResourceTable';
import 'src/components/pipelines/Pipelines.css';

export interface PipelineVersionListProps extends NavigationProps {
  pipelineId?: string;
  disablePaging?: boolean;
  disableSelection?: boolean;
  disableSorting?: boolean;
  noFilterBox?: boolean;
  onError: (message: string, error: Error) => void;
  onLoadSuccess?: () => void;
  onSelectionChange?: (selectedIds: string[]) => void;
  selectedIds?: string[];
}

interface PipelineVersionListState {
  pipelineVersions: V2beta1PipelineVersion[];
  loadError: boolean;
}

const descriptionCustomRenderer: React.FC<CustomRendererProps<string>> = (
  props: CustomRendererProps<string>,
) => {
  return (
    <span title={props.value || ''}>
      <Description description={props.value || ''} forceInline={true} />
    </span>
  );
};
class PipelineVersionList extends React.PureComponent<
  PipelineVersionListProps,
  PipelineVersionListState
> {
  private _tableRef = React.createRef<CustomTable>();
  private _loadGeneration = 0;
  private _isMounted = false;

  constructor(props: any) {
    super(props);

    this.state = {
      pipelineVersions: [],
      loadError: false,
    };
  }

  public componentDidMount(): void {
    this._isMounted = true;
  }
  public componentWillUnmount(): void {
    this._isMounted = false;
  }

  public _nameCustomRenderer: React.FC<
    CustomRendererProps<{ display_name?: string; name: string }>
  > = (props) => (
    <Link
      className='kfp-pipeline-version-link'
      title={'Name: ' + (props.value?.name || '')}
      onClick={(event) => event.stopPropagation()}
      to={RoutePage.PIPELINE_DETAILS.replace(
        ':' + RouteParams.pipelineId,
        encodeURIComponent(this.props.pipelineId || ''),
      ).replace(':' + RouteParams.pipelineVersionId, encodeURIComponent(props.id))}
    >
      {props.value?.display_name || props.value?.name}
    </Link>
  );

  public render(): React.JSX.Element {
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 1,
        label: 'Version name',
        sortKey: PipelineVersionSortKeys.DISPLAY_NAME,
      },
      { label: 'Description', flex: 3, customRenderer: descriptionCustomRenderer },
      { label: 'Uploaded on', flex: 1, sortKey: PipelineVersionSortKeys.CREATED_AT },
    ];

    const rows: Row[] = this.state.pipelineVersions.map((v) => {
      const row = {
        id: v.pipeline_version_id!,
        error: v.error?.message,
        otherFields: [
          { display_name: v.display_name, name: v.name },
          v.description,
          formatDateString(v.created_at),
        ] as any,
      };
      return row;
    });

    return (
      <div>
        <CustomTable
          renderTable={(table) => (
            <ResourceTable
              table={table}
              label='Pipeline versions'
              singular='version'
              plural='versions'
              getRowLabel={(row) => {
                const value = row.otherFields[0] as { display_name?: string; name?: string };
                return value.display_name || value.name || row.id;
              }}
            />
          )}
          errorMessage={
            this.state.loadError ? 'Unable to load pipeline versions. Try Refresh.' : undefined
          }
          columns={columns}
          rows={rows}
          selectedIds={this.props.selectedIds}
          initialSortColumn={PipelineVersionSortKeys.CREATED_AT}
          ref={this._tableRef}
          updateSelection={this.props.onSelectionChange}
          reload={this._loadPipelineVersions.bind(this)}
          disablePaging={this.props.disablePaging}
          disableSorting={this.props.disableSorting}
          disableSelection={this.props.disableSelection}
          noFilterBox={this.props.noFilterBox}
          emptyMessage='No pipeline versions found.'
        />
      </div>
    );
  }

  protected async _loadPipelineVersions(request: ListRequest): Promise<string> {
    const generation = ++this._loadGeneration;
    if (!this.props.pipelineId) return '';
    try {
      const response: V2beta1ListPipelineVersionsResponse =
        await Apis.pipelineServiceApiV2.listPipelineVersions(
          this.props.pipelineId,
          request.pageToken,
          request.pageSize,
          request.sortBy,
          request.filter,
        );
      if (!this._isMounted || generation !== this._loadGeneration) return '';
      this.setState({ pipelineVersions: response.pipeline_versions || [], loadError: false });
      this.props.onLoadSuccess?.();
      return response.next_page_token || '';
    } catch (err) {
      const error = new Error(await errorToMessage(err));
      if (!this._isMounted || generation !== this._loadGeneration) return '';
      this.setState({ loadError: true });
      this.props.onError('Error: failed to fetch pipeline versions.', error);
      return '';
    }
  }
}

export default PipelineVersionList;
