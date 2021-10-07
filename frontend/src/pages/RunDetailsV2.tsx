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
import { useState } from 'react';
import { Elements, FlowElement } from 'react-flow-renderer';
import { useQuery } from 'react-query';
import MD2Tabs from 'src/atoms/MD2Tabs';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import SidePanel from 'src/components/SidePanel';
import { commonCss, padding } from 'src/Css';
import { updateFlowElementsState } from 'src/lib/v2/DynamicFlow';
import { convertFlowElements } from 'src/lib/v2/StaticFlow';
import * as WorkflowUtils from 'src/lib/v2/WorkflowUtils';
import {
  getArtifactsFromContext,
  getEventsByExecutions,
  getExecutionsFromContext,
  getKfpV2RunContext,
} from 'src/mlmd/MlmdUtils';
import { Artifact, Event, Execution } from 'src/third_party/mlmd';
import { classes } from 'typestyle';
import { RunDetailsProps } from './RunDetails';
import DagCanvas from './v2/DagCanvas';

interface MlmdPackage {
  executions: Execution[];
  artifacts: Artifact[];
  events: Event[];
}

interface RunDetailsV2Info {
  pipeline_job: string;
  runId: string;
}

export type RunDetailsV2Props = RunDetailsV2Info & RunDetailsProps;

export function RunDetailsV2(props: RunDetailsV2Props) {
  const pipelineJobStr = props.pipeline_job;
  const pipelineSpec = WorkflowUtils.convertJsonToV2PipelineSpec(pipelineJobStr);
  const elements = convertFlowElements(pipelineSpec);

  const [flowElements, setFlowElements] = useState(elements);
  const [layers, setLayers] = useState(['root']);
  const [selectedTab, setSelectedTab] = useState(0);
  const [selectedNode, setSelectedNode] = useState<FlowElement<FlowElementDataBase> | null>(null);

  // TODO(zijianjoy): Update elements and states when layers change.
  const layerChange = (layers: string[]) => {
    setSelectedNode(null);
    setLayers(layers);
  };

  const onSelectionChange = (elements: Elements<FlowElementDataBase> | null) => {
    if (!elements || elements?.length === 0) {
      setSelectedNode(null);
      return;
    }
    if (elements && elements.length === 1) {
      setSelectedNode(elements[0]);
    }
  };

  const getNodeName = function(element: FlowElement<FlowElementDataBase> | null): string {
    if (element && element.data && element.data.label) {
      return element.data.label;
    }

    return 'unknown';
  };

  // Retrieves MLMD states from the MLMD store.
  const { isSuccess, data } = useQuery<MlmdPackage, Error>(
    ['mlmd_package', { id: props.runId }],
    async () => {
      const context = await getKfpV2RunContext(props.runId);
      const executions = await getExecutionsFromContext(context);
      const artifacts = await getArtifactsFromContext(context);
      const events = await getEventsByExecutions(executions);

      return { executions, artifacts, events };
    },
    {
      staleTime: 10000,
      onError: error =>
        props.updateBanner({
          message: 'Cannot get MLMD objects from Metadata store.',
          additionalInfo: error.message,
          mode: 'error',
        }),
      onSuccess: () => props.updateBanner({}),
    },
  );

  if (isSuccess && data && data.executions && data.events && data.artifacts) {
    updateFlowElementsState(flowElements, data.executions, data.events, data.artifacts);
  }

  return (
    <div className={classes(commonCss.page, padding(20, 't'))}>
      <MD2Tabs selectedTab={selectedTab} tabs={['Graph', 'Detail']} onSwitch={setSelectedTab} />
      {selectedTab === 0 && (
        <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
          <DagCanvas
            layers={layers}
            onLayersUpdate={layerChange}
            elements={flowElements}
            onSelectionChange={onSelectionChange}
            setFlowElements={elems => setFlowElements(elems)}
          ></DagCanvas>

          <div className='z-20'>
            <SidePanel
              isOpen={!!selectedNode}
              title={getNodeName(selectedNode)}
              onClose={() => onSelectionChange(null)}
              defaultWidth={'50%'}
            ></SidePanel>
          </div>
        </div>
      )}
    </div>
  );
}
