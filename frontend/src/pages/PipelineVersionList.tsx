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

import Tooltip from '@mui/material/Tooltip';
import CustomTable, { Column, CustomRendererProps, Row } from 'src/components/CustomTable';
import * as React from 'react';
import { Link } from 'react-router-dom';
import {
  V2beta1PipelineVersion,
  V2beta1ListPipelineVersionsResponse,
} from 'src/apisv2beta1/pipeline';
import { Description } from 'src/components/Description';
import { Apis, ListRequest, PipelineVersionSortKeys } from 'src/lib/Apis';
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { RoutePage, RouteParams } from 'src/components/Router';
import { commonCss } from 'src/Css';
import { ForwardedLink } from 'src/atoms/ForwardedLink';

export type PipelineVersionListProps = {
  pipelineId?: string;
  disablePaging?: boolean;
  disableSelection?: boolean;
  disableSorting?: boolean;
  noFilterBox?: boolean;
  onError: (message: string, error: Error) => void;
  onSelectionChange?: (selectedIds: string[]) => void;
  selectedIds?: string[];
};

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

const PipelineVersionList: React.FC<PipelineVersionListProps> = props => {
  const [pipelineVersions, setPipelineVersions] = React.useState<V2beta1PipelineVersion[]>([]);
  const tableRef = React.useRef<CustomTable>(null);

  const nameCustomRenderer: React.FC<CustomRendererProps<{
    display_name?: string;
    name: string;
  }>> = rendererProps => {
    return (
      <Tooltip
        title={'Name: ' + (rendererProps.value?.name || '')}
        enterDelay={300}
        placement='bottom-start'
      >
        {props.pipelineId ? (
          <ForwardedLink
            className={commonCss.link}
            onClick={e => e.stopPropagation()}
            to={RoutePage.PIPELINE_DETAILS.replace(
              ':' + RouteParams.pipelineId,
              props.pipelineId,
            ).replace(':' + RouteParams.pipelineVersionId, rendererProps.id)}
          >
            {rendererProps.value?.display_name || rendererProps.value?.name}
          </ForwardedLink>
        ) : (
          <Link
            className={commonCss.link}
            onClick={e => e.stopPropagation()}
            to={RoutePage.PIPELINE_DETAILS.replace(
              ':' + RouteParams.pipelineVersionId,
              rendererProps.id,
            )}
          >
            {rendererProps.value?.display_name || rendererProps.value?.name}
          </Link>
        )}
      </Tooltip>
    );
  };

  const loadPipelineVersions = React.useCallback(
    async (request: ListRequest): Promise<string> => {
      let response: V2beta1ListPipelineVersionsResponse | null = null;

      if (props.pipelineId) {
        try {
          response = await Apis.pipelineServiceApiV2.listPipelineVersions(
            props.pipelineId,
            request.pageToken,
            request.pageSize,
            request.sortBy,
            request.filter,
          );
        } catch (err) {
          const error = new Error(await errorToMessage(err));
          props.onError('Error: failed to fetch runs.', error);
          // No point in continuing if we couldn't retrieve any runs.
          return '';
        }

        setPipelineVersions(response.pipeline_versions || []);
      }
      return response ? response.next_page_token || '' : '';
    },
    [props],
  );

  const columns: Column[] = [
    {
      customRenderer: nameCustomRenderer,
      flex: 1,
      label: 'Version name',
      sortKey: PipelineVersionSortKeys.DISPLAY_NAME,
    },
    { label: 'Description', flex: 3, customRenderer: descriptionCustomRenderer },
    { label: 'Uploaded on', flex: 1, sortKey: PipelineVersionSortKeys.CREATED_AT },
  ];

  const rows: Row[] = pipelineVersions.map(v => {
    const row = {
      id: v.pipeline_version_id!,
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
        columns={columns}
        rows={rows}
        selectedIds={props.selectedIds}
        initialSortColumn={PipelineVersionSortKeys.CREATED_AT}
        ref={tableRef}
        updateSelection={props.onSelectionChange}
        reload={loadPipelineVersions}
        disablePaging={props.disablePaging}
        disableSorting={props.disableSorting}
        disableSelection={props.disableSelection}
        noFilterBox={props.noFilterBox}
        emptyMessage='No pipeline versions found.'
      />
    </div>
  );
};

export default PipelineVersionList;
