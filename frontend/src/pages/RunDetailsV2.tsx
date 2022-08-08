// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as React from 'react';
import { useEffect, useState } from 'react';
import { Elements, FlowElement } from 'react-flow-renderer';
import { useQuery } from 'react-query';
import { ApiExperiment } from 'src/apis/experiment';
import { ApiRun, ApiRunDetail, ApiRunStorageState } from 'src/apis/run';
import MD2Tabs from 'src/atoms/MD2Tabs';
import DetailsTable from 'src/components/DetailsTable';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import { PipelineSpecTabContent } from 'src/components/PipelineSpecTabContent';
import { RoutePage, RouteParams } from 'src/components/Router';
import SidePanel from 'src/components/SidePanel';
import { RuntimeNodeDetailsV2 } from 'src/components/tabs/RuntimeNodeDetailsV2';
import { ToolbarProps } from 'src/components/Toolbar';
import { commonCss, padding } from 'src/Css';
import { Apis } from 'src/lib/Apis';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import RunUtils from 'src/lib/RunUtils';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { hasFinished, NodePhase } from 'src/lib/StatusUtils';
import { formatDateString, getRunDurationFromApiRun } from 'src/lib/Utils';
import { getNodeMlmdInfo, updateFlowElementsState } from 'src/lib/v2/DynamicFlow';
import { convertFlowElements } from 'src/lib/v2/StaticFlow';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import {
  getArtifactsFromContext,
  getEventsByExecutions,
  getExecutionsFromContext,
  getKfpV2RunContext,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { Artifact, Event, Execution } from 'src/third_party/mlmd';
import { classes } from 'typestyle';
import { RunDetailsProps } from './RunDetails';
import { statusToIcon } from './Status';
import DagCanvas from './v2/DagCanvas';

const QUERY_STALE_TIME = 10000; // 10000 milliseconds == 10 seconds.
const QUERY_REFETCH_INTERNAL = 10000; // 10000 milliseconds == 10 seconds.
const TAB_NAMES = ['Graph', 'Detail', 'Pipeline Spec'];

interface MlmdPackage {
  executions: Execution[];
  artifacts: Artifact[];
  events: Event[];
}

export interface NodeMlmdInfo {
  execution?: Execution;
  linkedArtifact?: LinkedArtifact;
}

interface RunDetailsV2Info {
  pipeline_job: string;
  runDetail: ApiRunDetail;
}

export type RunDetailsV2Props = RunDetailsV2Info & RunDetailsProps;

export function RunDetailsV2(props: RunDetailsV2Props) {
  const runId = props.match.params[RouteParams.runId];
  const runDetail = props.runDetail;
  const pipelineJobStr = props.pipeline_job;
  const pipelineSpec = WorkflowUtils.convertYamlToV2PipelineSpec(pipelineJobStr);
  const elements = convertFlowElements(pipelineSpec);
  const templateString = runDetail.run?.pipeline_spec?.pipeline_manifest;

  const [flowElements, setFlowElements] = useState(elements);
  const [layers, setLayers] = useState(['root']);
  const [selectedTab, setSelectedTab] = useState(0);
  const [selectedNode, setSelectedNode] = useState<FlowElement<FlowElementDataBase> | null>(null);
  const [selectedNodeMlmdInfo, setSelectedNodeMlmdInfo] = useState<NodeMlmdInfo | null>(null);
  const [, forceUpdate] = useState();
  const [runFinished, setRunFinished] = useState(false);

  // TODO(zijianjoy): Update elements and states when layers change.
  const layerChange = (layers: string[]) => {
    setSelectedNode(null);
    setLayers(layers);
  };

  const getNodeName = function(element: FlowElement<FlowElementDataBase> | null): string {
    if (element && element.data && element.data.label) {
      return element.data.label;
    }

    return 'unknown';
  };

  // Retrieves MLMD states from the MLMD store.
  const { isSuccess, data } = useQuery<MlmdPackage, Error>(
    ['mlmd_package', { id: runId }],
    async () => {
      const context = await getKfpV2RunContext(runId);
      const executions = await getExecutionsFromContext(context);
      const artifacts = await getArtifactsFromContext(context);
      const events = await getEventsByExecutions(executions);

      return { executions, artifacts, events };
    },
    {
      staleTime: QUERY_STALE_TIME,
      refetchInterval: QUERY_REFETCH_INTERNAL,
      onError: error =>
        props.updateBanner({
          message: 'Cannot get MLMD objects from Metadata store.',
          additionalInfo: error.message,
          mode: 'error',
        }),
      onSuccess: () => props.updateBanner({}),
    },
  );

  if (isSuccess && data) {
    updateFlowElementsState(flowElements, data.executions, data.events, data.artifacts);
  }

  const onSelectionChange = (elements: Elements<FlowElementDataBase> | null) => {
    if (!elements || elements?.length === 0) {
      setSelectedNode(null);
      return;
    }
    if (elements && elements.length === 1) {
      setSelectedNode(elements[0]);
      if (data) {
        setSelectedNodeMlmdInfo(
          getNodeMlmdInfo(elements[0], data.executions, data.events, data.artifacts),
        );
      }
    }
  };

  // Retrieves experiment detail.
  const experimentId = RunUtils.getFirstExperimentReferenceId(runDetail.run);
  const { data: apiExperiment } = useQuery<ApiExperiment, Error>(
    ['RunDetailsV2_experiment', { runId: runId, experimentId: experimentId }],
    () => getExperiment(experimentId),
    {},
  );
  const namespace = RunUtils.getNamespaceReferenceName(apiExperiment);

  // Update page title and experiment information.
  useEffect(() => {
    updateToolBar(runDetail, apiExperiment, props.updateToolbar);
  }, [runDetail, apiExperiment, props.updateToolbar]);

  // Update buttons for managing runs.
  const [buttons] = useState(new Buttons(props, () => forceUpdate));
  const [runIdFromParams] = useState(props.match.params[RouteParams.runId]);
  useEffect(() => {
    if (hasFinished(runDetail.run?.status as NodePhase)) {
      setRunFinished(true);
    }
    updateToolBarActions(
      buttons,
      runIdFromParams,
      runDetail,
      runFinished,
      props.updateToolbar,
      () => forceUpdate,
      () => setRunFinished(false),
    );
  }, [buttons, runIdFromParams, runDetail, runFinished, props.updateToolbar]);

  return (
    <>
      <div className={classes(commonCss.page, padding(20, 't'))}>
        <MD2Tabs selectedTab={selectedTab} tabs={TAB_NAMES} onSwitch={setSelectedTab} />
        {/* DAG tab */}
        {selectedTab === 0 && (
          <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
            <DagCanvas
              layers={layers}
              onLayersUpdate={layerChange}
              elements={flowElements}
              onSelectionChange={onSelectionChange}
              setFlowElements={elems => setFlowElements(elems)}
            ></DagCanvas>

            {/* Side panel for Execution, Artifact, Sub-DAG. */}
            <div className='z-20'>
              <SidePanel
                isOpen={!!selectedNode}
                title={getNodeName(selectedNode)}
                onClose={() => onSelectionChange(null)}
                defaultWidth={'50%'}
              >
                <RuntimeNodeDetailsV2
                  element={selectedNode}
                  elementMlmdInfo={selectedNodeMlmdInfo}
                  namespace={namespace}
                ></RuntimeNodeDetailsV2>
              </SidePanel>
            </div>
          </div>
        )}

        {/* Run details tab */}
        {selectedTab === 1 && (
          <div className={padding()}>
            <DetailsTable title='Run details' fields={getDetailsFields(runDetail.run)} />
          </div>

          // TODO(zijianjoy): Wait backend to supply run parameters, so UI can show them.
        )}

        {/* Pipeline Spec tab */}
        {selectedTab === 2 && (
          <div className={commonCss.codeEditor} data-testid={'spec-ir'}>
            <PipelineSpecTabContent templateString={templateString || ''} />
          </div>
        )}
      </div>
    </>
  );
}

async function getExperiment(experimentId: string | null): Promise<ApiExperiment> {
  if (experimentId) {
    return Apis.experimentServiceApi.getExperiment(experimentId);
  }
  return Promise.resolve({});
}

function updateToolBar(
  apiRunDetail: ApiRunDetail | undefined,
  apiExperiment: ApiExperiment | undefined,
  updateToolBarCallback: (toolbarProps: Partial<ToolbarProps>) => void,
) {
  const runMetadata = apiRunDetail?.run;
  if (runMetadata) {
    const pageTitle = (
      <div className={commonCss.flex}>
        {statusToIcon(runMetadata.status as NodePhase, runMetadata.created_at)}
        <span style={{ marginLeft: 10 }}>{runMetadata.name || 'Run name unknown'}</span>
      </div>
    );

    updateToolBarCallback({ pageTitle, pageTitleTooltip: runMetadata.name });
  }

  const breadcrumbs: Array<{ displayName: string; href: string }> = [];
  if (apiExperiment && apiExperiment.id && apiExperiment.name) {
    breadcrumbs.push(
      { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
      {
        displayName: apiExperiment.name,
        href: RoutePage.EXPERIMENT_DETAILS.replace(
          ':' + RouteParams.experimentId,
          apiExperiment.id,
        ),
      },
    );
  } else {
    breadcrumbs.push({ displayName: 'All runs', href: RoutePage.RUNS });
  }
  updateToolBarCallback({ breadcrumbs });
}

function updateToolBarActions(
  buttons: Buttons,
  runIdFromParams: string,
  apiRunDetail: ApiRunDetail | undefined,
  runFinished: boolean,
  updateToolbar: (toolbarProps: Partial<ToolbarProps>) => void,
  refresh: () => void,
  retry: () => void,
) {
  const runMetadata = apiRunDetail?.run;
  const getRunIdList = () =>
    runMetadata && runMetadata.id ? [runMetadata.id] : runIdFromParams ? [runIdFromParams] : [];

  buttons
    .retryRun(getRunIdList, true, () => retry())
    .cloneRun(getRunIdList, true)
    .terminateRun(getRunIdList, true, () => refresh());
  !runMetadata || runMetadata.storage_state === ApiRunStorageState.ARCHIVED
    ? buttons.restore('run', getRunIdList, true, () => refresh())
    : buttons.archive('run', getRunIdList, true, () => refresh());

  const actions = buttons.getToolbarActionMap();
  actions[ButtonKeys.TERMINATE_RUN].disabled =
    (runMetadata && (runMetadata.status as NodePhase) === NodePhase.TERMINATING) || runFinished;
  actions[ButtonKeys.RETRY].disabled =
    !runMetadata ||
    ((runMetadata.status as NodePhase) !== NodePhase.FAILED &&
      (runMetadata.status as NodePhase) !== NodePhase.ERROR);

  updateToolbar({ actions });
}

function getDetailsFields(apiRun?: ApiRun): Array<KeyValue<string>> {
  // check if the run has finished or not. The default value for apiRun.finished_at is
  // Date(0), when it is not specified.
  let finishedAt = new Date(0);
  if (apiRun?.finished_at) {
    finishedAt = apiRun?.finished_at;
  }

  return [
    ['Run ID', apiRun?.id || '-'],
    ['Workflow name', apiRun?.name || '-'],
    ['Status', apiRun?.status],
    ['Description', apiRun?.description || ''],
    ['Created at', apiRun?.created_at ? formatDateString(apiRun.created_at) : '-'],
    ['Started at', formatDateString(apiRun?.scheduled_at)],
    ['Finished at', finishedAt > new Date(0) ? formatDateString(apiRun?.finished_at) : '-'],
    ['Duration', finishedAt > new Date(0) ? getRunDurationFromApiRun(apiRun) : '-'],
  ];
}
