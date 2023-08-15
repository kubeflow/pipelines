/*
 * Copyright 2023 The Kubeflow Authors
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
import React, { useEffect, useState } from 'react';
import { useQuery } from 'react-query';
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
import { Apis, ListRequest, PipelineSortKeys } from 'src/lib/Apis';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import { formatDateString } from 'src/lib/Utils';
import { Page, PageProps } from './Page';
import PipelineVersionList from './PipelineVersionList';

interface pipelineListProps {
  namespace?: string;
}

type pipelineListFCProps = PageProps & pipelineListProps;

interface DisplayPipeline extends V2beta1Pipeline {
  expandState?: ExpandState;
}

export function PipelineListFC(props: pipelineListFCProps) {
  const selectionChanged = (pipelineId: string | undefined, selectedIds: string[]) => {};
  const { namespace, updateToolbar } = props;
  const [refresh, setRefresh] = useState(true);
  const Refresh = () => setRefresh(refreshed => !refreshed);
  const [toolbarState, setToolbarState] = useState<ToolbarProps>(
    getInitialToolbarState([], {}, selectionChanged, props, Refresh),
  );
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [selectedVersionIds, setSelectedVersionIds] = useState({});
  const [displayPipelines, setDisplayPipelines] = useState<DisplayPipeline[]>([]);
  const [rows, setRows] = useState<Row[]>([]);
  const [nextPageToken, setNextPageToken] = useState<string>('');
  const [request, setRequest] = useState<ListRequest>({
    pageToken: '',
    pageSize: 10,
    sortBy: 'created_at desc',
    filter: '',
  });

  const columns: Column[] = [
    {
      customRenderer: nameCustomRenderer,
      flex: 1,
      label: 'Pipeline name',
      sortKey: PipelineSortKeys.NAME,
    },
    { label: 'Description', flex: 3, customRenderer: descriptionCustomRenderer },
    { label: 'Uploaded on', sortKey: PipelineSortKeys.CREATED_AT, flex: 1 },
  ];

  const {
    isFetched: pipelineIsFetched,
    data: pipelineList,
    refetch: refetchPipelineList,
  } = useQuery<V2beta1Pipeline[], Error>(
    ['pipelineList'],
    async () => {
      let pipelineListResponse: V2beta1ListPipelinesResponse;
      try {
        pipelineListResponse = await Apis.pipelineServiceApiV2.listPipelines(
          namespace,
          request.pageToken,
          request.pageSize,
          request.sortBy,
          request.filter,
        );
        setNextPageToken(pipelineListResponse.next_page_token || '');
        return pipelineListResponse.pipelines ?? [];
      } catch (err) {
        throw new Error('Error: failed to retrieve list of pipelines.');
      }
    },
    { enabled: !!request },
  );

  useEffect(() => {
    if (pipelineList) {
      let updatedDisplayPipelines: DisplayPipeline[] = pipelineList;
      updatedDisplayPipelines.forEach(exp => (exp.expandState = ExpandState.COLLAPSED));
      setDisplayPipelines(updatedDisplayPipelines);
    }
  }, [pipelineList]);

  useEffect(() => {
    setToolbarState(
      getInitialToolbarState(selectedIds, selectedVersionIds, selectionChanged, props, Refresh),
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedIds, selectedVersionIds]);

  useEffect(() => {
    updateToolbar(toolbarState);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [toolbarState]);

  useEffect(() => {
    const rows = displayPipelines.map(p => {
      return {
        expandState: p.expandState,
        id: p.pipeline_id!,
        otherFields: [p.display_name!, p.description!, formatDateString(p.created_at!)],
      };
    });
    setRows(rows);
  }, [displayPipelines]);

  useEffect(() => {
    refetchPipelineList();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [request]);

  const reload = async (listRequest: ListRequest) => {
    setRequest(listRequest);
    return nextPageToken;
  };

  return (
    <div className={classes(commonCss.page, padding(20, 'lr'))}>
      Display pipeline list here.
      {pipelineIsFetched && (
        <CustomTable
          // ref={this._tableRef}
          columns={columns}
          rows={rows}
          initialSortColumn={PipelineSortKeys.CREATED_AT}
          // updateSelection={this._selectionChanged.bind(this, undefined)}
          selectedIds={selectedIds}
          reload={reload}
          // toggleExpansion={this._toggleRowExpand.bind(this)}
          // getExpandComponent={this._getExpandedPipelineComponent.bind(this)}
          filterLabel='Filter pipelines'
          emptyMessage='No pipelines found. Click "Upload pipeline" to start.'
        />
      )}
    </div>
  );
}

function getInitialToolbarState(
  selectedIds: string[],
  selectedVersionIds: { [pipelineId: string]: string[] },
  selectionChanged: (pipelineId: string | undefined, selectedIds: string[]) => void,
  props: PageProps,
  refresh: () => void,
) {
  const buttons = new Buttons(props, refresh);
  return {
    actions: buttons
      .newPipelineVersion('Upload pipeline')
      .refresh(refresh)
      .deletePipelinesAndPipelineVersions(
        () => selectedIds,
        () => selectedVersionIds,
        (pipelineId, ids) => selectionChanged(pipelineId, ids),
        false /* useCurrentResource */,
      )
      .getToolbarActionMap(),
    breadcrumbs: [],
    pageTitle: 'Pipelines',
  };
}

const descriptionCustomRenderer: React.FC<CustomRendererProps<string>> = (
  props: CustomRendererProps<string>,
) => {
  return <Description description={props.value || ''} forceInline={true} />;
};

const nameCustomRenderer: React.FC<CustomRendererProps<string>> = (
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
