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
import { MouseEvent as ReactMouseEvent, useEffect, useState } from 'react';
import { Edge, FlowElement, Node } from 'react-flow-renderer';
import { useQuery } from 'react-query';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';
import { V2beta1Run, V2beta1RuntimeState, V2beta1RunStorageState } from 'src/apisv2beta1/run';
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
import { KeyValue } from 'src/lib/StaticGraphParser';
import { hasFinishedV2, statusProtoMap } from 'src/lib/StatusUtils';
import { formatDateString, getRunDurationV2 } from 'src/lib/Utils';
import {
  convertSubDagToRuntimeFlowElements,
  getNodeMlmdInfo,
  updateFlowElementsState,
} from 'src/lib/v2/DynamicFlow';
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
import { statusToIcon } from './StatusV2';
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
  run: V2beta1Run;
}

export type RunDetailsV2Props = RunDetailsV2Info & RunDetailsProps;

export function RunDetailsV2(props: RunDetailsV2Props) {
  const runId = props.match.params[RouteParams.runId];
  const run = props.run;
  const pipelineJobStr = props.pipeline_job;
  const pipelineSpec = WorkflowUtils.convertYamlToV2PipelineSpec(pipelineJobStr);
  const elements = convertFlowElements(pipelineSpec);

  const [flowElements, setFlowElements] = useState(elements);
  const [layers, setLayers] = useState(['root']);
  const [selectedTab, setSelectedTab] = useState(0);
  const [selectedNode, setSelectedNode] = useState<FlowElement<FlowElementDataBase> | null>(null);
  const [selectedNodeMlmdInfo, setSelectedNodeMlmdInfo] = useState<NodeMlmdInfo | null>(null);
  const [, forceUpdate] = useState();
  const [runFinished, setRunFinished] = useState(false);

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

  const layerChange = (layers: string[]) => {
    setSelectedNode(null);
    setLayers(layers);
    setFlowElements(
      convertSubDagToRuntimeFlowElements(pipelineSpec, layers, data ? data.executions : []),
    ); // render elements in the sub-layer.
  };

  let dynamicFlowElements = flowElements;
  if (isSuccess && data) {
    dynamicFlowElements = updateFlowElementsState(
      layers,
      flowElements,
      data.executions,
      data.events,
      data.artifacts,
    );
  }

  const onElementSelection = (event: ReactMouseEvent, element: Node | Edge) => {
    setSelectedNode(element);
    if (data) {
      setSelectedNodeMlmdInfo(
        getNodeMlmdInfo(element, data.executions, data.events, data.artifacts),
      );
    }
  };

  // Retrieves experiment detail.
  const experimentId = run.experiment_id || null;
  const { data: experiment } = useQuery<V2beta1Experiment, Error>(
    ['RunDetailsV2_experiment', { runId: runId, experimentId: experimentId }],
    () => getExperiment(experimentId),
    {},
  );
  const namespace = experiment?.namespace;

  // Update page title and experiment information.
  useEffect(() => {
    updateToolBar(run, experiment, props.updateToolbar);
  }, [run, experiment, props.updateToolbar]);

  // Update buttons for managing runs.
  const [buttons] = useState(new Buttons(props, () => forceUpdate));
  const [runIdFromParams] = useState(props.match.params[RouteParams.runId]);
  useEffect(() => {
    if (hasFinishedV2(run.state)) {
      setRunFinished(true);
    }
    updateToolBarActions(
      buttons,
      runIdFromParams,
      run,
      runFinished,
      props.updateToolbar,
      () => forceUpdate,
      () => setRunFinished(false),
    );
  }, [buttons, runIdFromParams, run, runFinished, props.updateToolbar]);

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
              elements={dynamicFlowElements}
              onElementClick={onElementSelection}
              setFlowElements={elems => setFlowElements(elems)}
            ></DagCanvas>

            {/* Side panel for Execution, Artifact, Sub-DAG. */}
            <div className='z-20'>
              <SidePanel
                isOpen={!!selectedNode}
                title={getNodeName(selectedNode)}
                onClose={() => setSelectedNode(null)}
                defaultWidth={'50%'}
              >
                <RuntimeNodeDetailsV2
                  layers={layers}
                  onLayerChange={layerChange}
                  pipelineJobString={pipelineJobStr}
                  runId={runId}
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
            <DetailsTable title='Run details' fields={getDetailsFields(run)} />

            {!!run.runtime_config?.parameters && (
              <DetailsTable
                title='Run parameters'
                fields={Object.entries(run.runtime_config?.parameters).map(param => [
                  param[0],
                  param[1],
                ])}
              />
            )}
          </div>
        )}

        {/* Pipeline Spec tab */}
        {selectedTab === 2 && (
          <div className={commonCss.codeEditor} data-testid={'spec-ir'}>
            <PipelineSpecTabContent templateString={pipelineJobStr || ''} />
          </div>
        )}
      </div>
    </>
  );
}

async function getExperiment(experimentId: string | null): Promise<V2beta1Experiment> {
  if (experimentId) {
    return Apis.experimentServiceApiV2.getExperiment(experimentId);
  }
  return Promise.resolve({});
}

function updateToolBar(
  run: V2beta1Run | undefined,
  experiment: V2beta1Experiment | undefined,
  updateToolBarCallback: (toolbarProps: Partial<ToolbarProps>) => void,
) {
  const runMetadata = run;
  if (runMetadata) {
    const pageTitle = (
      <div className={commonCss.flex}>
        {statusToIcon(runMetadata.state, runMetadata.created_at)}
        <span style={{ marginLeft: 10 }}>{runMetadata.display_name || 'Run name unknown'}</span>
      </div>
    );

    updateToolBarCallback({ pageTitle, pageTitleTooltip: runMetadata.display_name });
  }

  const breadcrumbs: Array<{ displayName: string; href: string }> = [];
  if (experiment && experiment.experiment_id && experiment.display_name) {
    breadcrumbs.push(
      { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
      {
        displayName: experiment.display_name,
        href: RoutePage.EXPERIMENT_DETAILS.replace(
          ':' + RouteParams.experimentId,
          experiment.experiment_id,
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
  run: V2beta1Run | undefined,
  runFinished: boolean,
  updateToolbar: (toolbarProps: Partial<ToolbarProps>) => void,
  refresh: () => void,
  retry: () => void,
) {
  const runMetadata = run;
  const getRunIdList = () =>
    runMetadata && runMetadata.run_id
      ? [runMetadata.run_id]
      : runIdFromParams
      ? [runIdFromParams]
      : [];

  buttons
    .retryRun(getRunIdList, true, () => retry())
    .cloneRun(getRunIdList, true)
    .terminateRun(getRunIdList, true, () => refresh());
  !runMetadata || runMetadata.storage_state === V2beta1RunStorageState.ARCHIVED
    ? buttons.restore('run', getRunIdList, true, () => refresh())
    : buttons.archive('run', getRunIdList, true, () => refresh());

  const actions = buttons.getToolbarActionMap();
  actions[ButtonKeys.TERMINATE_RUN].disabled =
    (runMetadata && runMetadata.state === V2beta1RuntimeState.CANCELING) || runFinished;
  actions[ButtonKeys.RETRY].disabled =
    !runMetadata || runMetadata.state !== V2beta1RuntimeState.FAILED;

  updateToolbar({ actions });
}

function getDetailsFields(run?: V2beta1Run): Array<KeyValue<string>> {
  return [
    ['Run ID', run?.run_id || '-'],
    ['Workflow name', run?.display_name || '-'],
    ['Status', run?.state ? statusProtoMap.get(run?.state) : '-'],
    ['Description', run?.description || ''],
    ['Created at', run?.created_at ? formatDateString(run.created_at) : '-'],
    ['Started at', formatDateString(run?.scheduled_at)],
    ['Finished at', hasFinishedV2(run?.state) ? formatDateString(run?.finished_at) : '-'],
    ['Duration', hasFinishedV2(run?.state) ? getRunDurationV2(run) : '-'],
  ];
}
