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

import {
  MouseEvent as ReactMouseEvent,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useQuery } from '@tanstack/react-query';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import { queryKeys } from 'src/hooks/queryKeys';
import { useKeyedState } from 'src/hooks/useKeyedState';
import {
  V2beta1PipelineTask,
  V2beta1Run,
  V2beta1RuntimeState,
  V2beta1RunStorageState,
} from 'src/apisv2beta1/run';
import MD2Tabs from 'src/atoms/MD2Tabs';
import Banner from 'src/components/Banner';
import DetailsTable from 'src/components/DetailsTable';
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
  buildRuntimeFlowContext,
  convertSubDagToRuntimeFlowElements,
  getNodeRuntimeInfo,
  reconcileRuntimeFlowElements,
} from 'src/lib/v2/DynamicFlow';
import { convertFlowElements, getNodeName, PipelineFlowElement } from 'src/lib/v2/StaticFlow';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import { listAllRunTasks } from 'src/lib/v2/RunTaskUtils';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { classes } from 'typestyle';
import { RouteComponentProps } from 'react-router-dom';
import { RunDetailsProps } from './RunDetails';
import { statusToIcon } from './StatusV2';
import DagCanvas from './v2/DagCanvas';

const QUERY_STALE_TIME = 10000; // 10000 milliseconds == 10 seconds.
const QUERY_REFETCH_INTERVAL = 10000; // 10000 milliseconds == 10 seconds.
const TAB_NAMES = ['Graph', 'Detail', 'Pipeline Spec'];

interface RunDetailsV2Info {
  onRetryStarted?: () => void;
  pipeline_job: string;
  parsedPipelineSpec?: PipelineSpec;
  run: V2beta1Run;
  runRefreshError?: Error | null;
}

export interface RunDetailsV2Params {
  [RouteParams.runId]: string;
}

export type RunDetailsV2Props = RunDetailsV2Info &
  RunDetailsProps &
  RouteComponentProps<RunDetailsV2Params>;

export function RunDetailsV2(props: RunDetailsV2Props) {
  const { onRetryStarted, updateToolbar } = props;
  const { updateBanner } = props;
  const runId = props.match.params[RouteParams.runId];
  const run = props.run;
  const selectedNamespace = useContext(NamespaceContext);
  const pipelineJobStr = props.pipeline_job;
  const pipelineSpec = useMemo(
    () => props.parsedPipelineSpec || WorkflowUtils.convertYamlToV2PipelineSpec(pipelineJobStr),
    [pipelineJobStr, props.parsedPipelineSpec],
  );
  const initialElements = useMemo(() => convertFlowElements(pipelineSpec), [pipelineSpec]);

  const [flowElements, setFlowElements] = useState(initialElements);
  const [layers, setLayers] = useState(['root']);
  const [selectedTab, setSelectedTab] = useState(0);
  const [selectedNode, setSelectedNode] = useState<PipelineFlowElement | null>(null);
  const [, forceUpdate] = useState();
  const runStateKey = `${run.run_id || runId}:${run.state || ''}`;
  const [retriedCurrentRunState, setRetriedCurrentRunState] = useKeyedState(runStateKey, false);
  const runFinished = hasFinishedV2(run.state) && !retriedCurrentRunState;
  const runIsTerminal = hasFinishedV2(run.state);
  const previousRunStatus = useRef({ runId, isTerminal: runIsTerminal });

  const {
    isSuccess,
    isError,
    error,
    data: tasks,
    refetch: refetchTasks,
  } = useQuery<V2beta1PipelineTask[], Error>({
    queryKey: queryKeys.runTasks(runId),
    queryFn: () => listAllRunTasks(runId),
    staleTime: QUERY_STALE_TIME,
    refetchInterval: runFinished ? false : QUERY_REFETCH_INTERVAL,
    // Terminal run data can arrive while the cached task snapshot is still fresh. Always verify
    // task state on a terminal mount instead of preserving a potentially running graph forever.
    refetchOnMount: runIsTerminal ? 'always' : true,
  });

  // The terminal run update stops polling immediately, so fetch once more to capture the final
  // task states. Initial terminal mounts and same-state rerenders use the normal query lifecycle.
  useEffect(() => {
    const previousStatus = previousRunStatus.current;
    previousRunStatus.current = { runId, isTerminal: runIsTerminal };

    if (previousStatus.runId === runId && !previousStatus.isTerminal && runIsTerminal) {
      void refetchTasks();
    }
  }, [refetchTasks, runId, runIsTerminal]);

  // Retrieves experiment detail.
  const experimentId = run.experiment_id || null;
  const {
    data: experiment,
    isError: experimentIsError,
    error: experimentError,
  } = useQuery<V2beta1Experiment, Error>({
    queryKey: queryKeys.runDetailsV2Experiment(runId, experimentId),
    queryFn: () => getExperiment(experimentId),
  });
  const namespace = experiment?.namespace || selectedNamespace;

  // Query errors take precedence over experiment errors; clear only after both recover.
  useEffect(() => {
    if (isError && error) {
      updateBanner({
        message: 'Cannot get tasks for this run. Refresh the page to try again.',
        additionalInfo: error.message,
        mode: 'error',
      });
    } else if (experimentIsError && experimentError) {
      updateBanner({
        message: 'Error: failed to retrieve experiment details.',
        additionalInfo: experimentError.message,
        mode: 'warning',
      });
    } else if (isSuccess) {
      updateBanner({});
    }
  }, [isError, isSuccess, error, experimentIsError, experimentError, updateBanner]);

  const layerChange = useCallback(
    (layers: string[]) => {
      setSelectedNode(null);
      setLayers(layers);
      setFlowElements(convertSubDagToRuntimeFlowElements(pipelineSpec, layers, tasks || []));
    },
    [pipelineSpec, tasks],
  );

  const runtimeFlowContext = useMemo(
    () => buildRuntimeFlowContext(layers, tasks || []),
    [layers, tasks],
  );

  const dynamicFlowElements = useMemo(() => {
    if (!tasks) {
      return flowElements;
    }

    return reconcileRuntimeFlowElements(layers, flowElements, tasks, runtimeFlowContext);
  }, [flowElements, layers, runtimeFlowContext, tasks]);

  const selectedNodeRuntimeInfo = useMemo(
    () => getNodeRuntimeInfo(selectedNode, tasks || [], layers, runtimeFlowContext),
    [layers, runtimeFlowContext, selectedNode, tasks],
  );

  const onElementSelection = (_event: ReactMouseEvent, element: PipelineFlowElement) => {
    setSelectedNode(element);
  };

  // Update page title and experiment information.
  useEffect(() => {
    updateToolBar(run, experiment, updateToolbar);
  }, [run, experiment, updateToolbar]);

  // Update buttons for managing runs.
  const [buttons] = useState(new Buttons(props, () => forceUpdate));
  const [runIdFromParams] = useState(props.match.params[RouteParams.runId]);
  useEffect(() => {
    updateToolBarActions(
      buttons,
      runIdFromParams,
      run,
      runFinished,
      updateToolbar,
      () => forceUpdate,
      (_selectedIds, success) => {
        if (success) {
          setRetriedCurrentRunState(true);
          onRetryStarted?.();
        }
      },
    );
  }, [
    buttons,
    runIdFromParams,
    run,
    runFinished,
    updateToolbar,
    onRetryStarted,
    setRetriedCurrentRunState,
  ]);

  return (
    <>
      {props.runRefreshError && (
        <Banner
          message='Unable to refresh this run. The last known run state is still shown. Refresh the page to try again.'
          additionalInfo={props.runRefreshError.message}
          mode='warning'
        />
      )}
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
              setFlowElements={(elems) => setFlowElements(elems)}
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
                  elementRuntimeInfo={selectedNodeRuntimeInfo}
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
                fields={Object.entries(run.runtime_config?.parameters).map((param) => [
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
  retry: (selectedIds: string[], success: boolean) => void,
) {
  const runMetadata = run;
  const getRunIdList = () =>
    runMetadata && runMetadata.run_id
      ? [runMetadata.run_id]
      : runIdFromParams
        ? [runIdFromParams]
        : [];

  buttons
    .retryRun(getRunIdList, true, retry)
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

function getActualStartTime(run?: V2beta1Run): Date | undefined {
  if (run?.state_history) {
    for (let i = run.state_history.length - 1; i >= 0; i--) {
      const entry = run.state_history[i];
      if (entry.state === V2beta1RuntimeState.RUNNING && entry.update_time !== undefined) {
        return entry.update_time;
      }
    }
  }
  return run?.scheduled_at;
}

function getDetailsFields(run?: V2beta1Run): Array<KeyValue<string>> {
  const actualStart = getActualStartTime(run);
  const scheduledAt = run?.scheduled_at;
  const startDiffers =
    actualStart && scheduledAt && actualStart.getTime() !== scheduledAt.getTime();

  const fields: Array<KeyValue<string>> = [
    ['Run ID', run?.run_id || '-'],
    ['Workflow name', run?.display_name || '-'],
    ['Status', run?.state ? statusProtoMap.get(run?.state) : '-'],
    ['Description', run?.description || ''],
    ['Created at', run?.created_at ? formatDateString(run.created_at) : '-'],
    ['Started at', formatDateString(actualStart)],
    ['Finished at', hasFinishedV2(run?.state) ? formatDateString(run?.finished_at) : '-'],
    ['Duration', hasFinishedV2(run?.state) ? getRunDurationV2(run) : '-'],
  ];

  if (startDiffers) {
    const startedAtIndex = fields.findIndex((field) => field[0] === 'Started at');
    const scheduledAtField: KeyValue<string> = ['Scheduled at', formatDateString(scheduledAt)];
    if (startedAtIndex >= 0) {
      fields.splice(startedAtIndex, 0, scheduledAtField);
    } else {
      fields.push(scheduledAtField);
    }
  }

  return fields;
}
