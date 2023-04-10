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
import CustomTable, { Column, CustomRendererProps, Row } from 'src/components/CustomTable';
import * as React from 'react';
import { Link, RouteComponentProps } from 'react-router-dom';
import {
  V2beta1PipelineVersion,
  V2beta1ListPipelineVersionsResponse,
} from 'src/apisv2beta1/pipeline';
import { Description } from 'src/components/Description';
import { Apis, ListRequest, PipelineVersionSortKeys } from 'src/lib/Apis';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { RoutePage, RouteParams } from 'src/components/Router';
import { commonCss } from 'src/Css';

export interface PipelineVersionListProps extends RouteComponentProps {
  pipelineId?: string;
  disablePaging?: boolean;
  disableSelection?: boolean;
  disableSorting?: boolean;
  noFilterBox?: boolean;
  onError: (message: string, error: Error) => void;
  onSelectionChange?: (selectedIds: string[]) => void;
  selectedIds?: string[];
}

interface PipelineVersionListState {
  pipelineVersions: V2beta1PipelineVersion[];
}

const descriptionCustomRenderer: React.FC<CustomRendererProps<string>> = (
  props: CustomRendererProps<string>,
) => {
  return (
    <Tooltip title={props.value || ''} enterDelay={300} placement='bottom-start'>
      <span>
        <Description description={props.value || ''} forceInline={true} />
      </span>
    </Tooltip>
  );
};
class PipelineVersionList extends React.PureComponent<
  PipelineVersionListProps,
  PipelineVersionListState
> {
  private _tableRef = React.createRef<CustomTable>();

  constructor(props: any) {
    super(props);

    this.state = {
      pipelineVersions: [],
    };
  }

  public _nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
    props: CustomRendererProps<string>,
  ) => {
    return (
      <Tooltip title={props.value || ''} enterDelay={300} placement='bottom-start'>
        {this.props.pipelineId ? (
          <Link
            className={commonCss.link}
            onClick={e => e.stopPropagation()}
            to={RoutePage.PIPELINE_DETAILS.replace(
              ':' + RouteParams.pipelineId,
              this.props.pipelineId,
            ).replace(':' + RouteParams.pipelineVersionId, props.id)}
          >
            {props.value}
          </Link>
        ) : (
          <Link
            className={commonCss.link}
            onClick={e => e.stopPropagation()}
            to={RoutePage.PIPELINE_DETAILS.replace(':' + RouteParams.pipelineVersionId, props.id)}
          >
            {props.value}
          </Link>
        )}
      </Tooltip>
    );
  };

  public render(): JSX.Element {
    const columns: Column[] = [
      {
        customRenderer: this._nameCustomRenderer,
        flex: 1,
        label: 'Version name',
        sortKey: PipelineVersionSortKeys.NAME,
      },
      { label: 'Description', flex: 3, customRenderer: descriptionCustomRenderer },
      { label: 'Uploaded on', flex: 1, sortKey: PipelineVersionSortKeys.CREATED_AT },
    ];

    const rows: Row[] = this.state.pipelineVersions.map(v => {
      const row = {
        id: v.pipeline_version_id!,
        otherFields: [v.display_name, v.description, formatDateString(v.created_at)] as any,
      };
      return row;
    });

    return (
      <div>
        <CustomTable
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
    let response: V2beta1ListPipelineVersionsResponse | null = null;

    if (this.props.pipelineId) {
      try {
        response = await Apis.pipelineServiceApiV2.listPipelineVersions(
          this.props.pipelineId,
          request.pageToken,
          request.pageSize,
          request.sortBy,
          request.filter,
        );
      } catch (err) {
        const error = new Error(await errorToMessage(err));
        this.props.onError('Error: failed to fetch runs.', error);
        // No point in continuing if we couldn't retrieve any runs.
        return '';
      }

      this.setState({
        pipelineVersions: response.pipeline_versions || [],
      });
    }
    return response ? response.next_page_token || '' : '';
  }
}

export default PipelineVersionList;
