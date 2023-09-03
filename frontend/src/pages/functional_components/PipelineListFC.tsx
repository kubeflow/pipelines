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
import { errorToMessage, formatDateString } from 'src/lib/Utils';
import { Page, PageProps } from 'src/pages/Page';
import PipelineVersionList from 'src/pages/PipelineVersionList';

interface pipelineListProps {
  namespace?: string;
}

type pipelineListFCProps = PageProps & pipelineListProps;

type versionIdsMap = {
  [pipelineId: string]: string[];
};

interface DisplayPipeline extends V2beta1Pipeline {
  expandState?: ExpandState;
}

export function PipelineListFC(props: pipelineListFCProps) {
  const { namespace, updateBanner, updateToolbar } = props;
  const [selectedPipelineIds, setSelectedPipelineIds] = useState<string[]>([]);
  const [selectedVersionIds, setSelectedVersionIds] = useState<versionIdsMap>({});
  const [refresh, setRefresh] = useState(true);
  const Refresh = () => setRefresh(refreshed => !refreshed);
  const selectionChanged = (parentId: string | undefined, selectedIds: string[]) => {
    if (!!parentId) {
      // select version
      // console.log({[parentId]: selectedIds})
      // let updatedSelectedVersionId = Object.assign(selectedVersionIds, {[parentId]: selectedIds})

      // console.log(updatedSelectedVersionId)
      // setSelectedVersionIds(updatedSelectedVersionId);

      // let updatedSelectedVersionId = new Map<string, string[]>();
      setSelectedVersionIds({ [parentId]: selectedIds });
      // Update selected pipeline version ids.
      // this.setStateSafe({
      //   selectedVersionIds: { ...this.state.selectedVersionIds, ...{ [pipelineId!]: selectedIds } },
      // });
      // console.log('set is finished')

      // const actions = this.props.toolbarProps.actions;
      // actions[ButtonKeys.DELETE_RUN].disabled =
      //   this.state.selectedIds.length < 1 && selectedIds.length < 1;
      // this.props.updateToolbar({ actions });
    } else {
      // select pipeline
      setSelectedPipelineIds(selectedIds);

      // const selectedVersionIdsCt = this._deepCountDictionary(this.state.selectedVersionIds);
      // const actions = this.props.toolbarProps.actions;
      // actions[ButtonKeys.DELETE_RUN].disabled = selectedIds.length < 1 && selectedVersionIdsCt < 1;
      // this.props.updateToolbar({ actions });
    }
  };

  // console.log(selectedVersionIds)

  const [toolbarState, setToolbarState] = useState<ToolbarProps>(
    getInitialToolbarState([], {}, selectionChanged, props, Refresh),
  );
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
      getInitialToolbarState(
        selectedPipelineIds,
        selectedVersionIds,
        selectionChanged,
        props,
        Refresh,
      ),
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedPipelineIds, selectedVersionIds]);

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

  // useEffect(() => {
  //   refetchPipelineList();
  //   // eslint-disable-next-line react-hooks/exhaustive-deps
  // }, [request]);

  const reload = async (updateRequest: ListRequest) => {
    let response: V2beta1ListPipelinesResponse | null = null;
    let displayPipelines: DisplayPipeline[];
    try {
      response = await Apis.pipelineServiceApiV2.listPipelines(
        namespace,
        updateRequest.pageToken,
        updateRequest.pageSize,
        updateRequest.sortBy,
        updateRequest.filter,
      );
      displayPipelines = response.pipelines || [];
      displayPipelines.forEach(exp => (exp.expandState = ExpandState.COLLAPSED));
      setDisplayPipelines(displayPipelines);
    } catch (err) {
      await showPageError('Error: failed to retrieve list of pipelines.', err);
    }
    return response ? response.next_page_token || '' : '';
    // setRequest(listRequest)
    // await refetchPipelineList();
    // // wait until nextPageToken is updated, then return
    // return nextPageToken;
  };

  const showPageError = async (message: string, error: Error | undefined) => {
    const errorMessage = await errorToMessage(error);
    updateBanner({
      additionalInfo: errorMessage ? errorMessage : undefined,
      message: message + (errorMessage ? ' Click Details for more information.' : ''),
    });
  };

  const getExpandedPipelineComponent = (rowIndex: number) => {
    const pipeline = displayPipelines[rowIndex];
    const updateVersionSelection = (selectedIds: string[]) => {
      selectionChanged(pipeline.pipeline_id, selectedIds);
    };
    return (
      <PipelineVersionList
        pipelineId={pipeline.pipeline_id}
        onError={() => null}
        {...props}
        selectedIds={selectedVersionIds[pipeline.pipeline_id!] || []}
        noFilterBox={true}
        onSelectionChange={updateVersionSelection}
        disableSorting={false}
        disablePaging={false}
      />
    );
  };

  const toggleRowExpand = (rowIndex: number) => {
    const updatedDisplayPipelines = produce(displayPipelines, draft => {
      draft[rowIndex].expandState =
        draft[rowIndex].expandState === ExpandState.COLLAPSED
          ? ExpandState.EXPANDED
          : ExpandState.COLLAPSED;
    });

    setDisplayPipelines(updatedDisplayPipelines);
  };

  const updatePipelineSelection = (selectedIds: string[]) => {
    selectionChanged(undefined, selectedIds);
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
          updateSelection={updatePipelineSelection}
          selectedIds={selectedPipelineIds}
          reload={reload}
          toggleExpansion={toggleRowExpand}
          getExpandComponent={getExpandedPipelineComponent}
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
