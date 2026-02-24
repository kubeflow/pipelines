/*
 * Copyright 2021 The Kubeflow Authors
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
import React, { MouseEvent as ReactMouseEvent, useState } from 'react';
import { Edge, FlowElement, Node } from 'react-flow-renderer';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import MD2Tabs from 'src/atoms/MD2Tabs';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import { PipelineVersionCard } from 'src/components/navigators/PipelineVersionCard';
import { PipelineSpecTabContent } from 'src/components/PipelineSpecTabContent';
import SidePanel from 'src/components/SidePanel';
import { StaticNodeDetailsV2 } from 'src/components/tabs/StaticNodeDetailsV2';
import { PipelineFlowElement } from 'src/lib/v2/StaticFlow';
import { commonCss, padding } from 'src/Css';
import DagCanvas from './v2/DagCanvas';

const TAB_NAMES = ['Graph', 'Pipeline Spec'];

interface PipelineDetailsV2Props {
  templateString?: string;
  pipelineFlowElements: PipelineFlowElement[];
  setSubDagLayers: (layers: string[]) => void;
  pipeline: V2beta1Pipeline | null;
  selectedVersion: V2beta1PipelineVersion | undefined;
  versions: V2beta1PipelineVersion[];
  handleVersionSelected: (versionId: string) => Promise<void>;
}

function PipelineDetailsV2({
  templateString,
  pipelineFlowElements,
  setSubDagLayers,
  pipeline,
  selectedVersion,
  versions,
  handleVersionSelected,
}: PipelineDetailsV2Props) {
  const [layers, setLayers] = useState(['root']);
  const [selectedTab, setSelectedTab] = useState(0);
  const [selectedNode, setSelectedNode] = useState<FlowElement<FlowElementDataBase> | null>(null);

  const layerChange = (l: string[]) => {
    setSelectedNode(null);
    setLayers(l);
    setSubDagLayers(l);
  };

  const getNodeName = function(element: FlowElement<FlowElementDataBase> | null): string {
    if (element && element.data && element.data.label) {
      return element.data.label;
    }

    return 'unknown';
  };

  return (
    <div className={commonCss.page} data-testid={'pipeline-detail-v2'}>
      <MD2Tabs selectedTab={selectedTab} onSwitch={setSelectedTab} tabs={TAB_NAMES} />
      {selectedTab === 0 && (
        <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
          <DagCanvas
            layers={layers}
            onLayersUpdate={layerChange}
            elements={pipelineFlowElements}
            onElementClick={(event: ReactMouseEvent, element: Node | Edge) =>
              setSelectedNode(element)
            }
            setFlowElements={() => {}}
          ></DagCanvas>
          <PipelineVersionCard
            pipeline={pipeline}
            selectedVersion={selectedVersion}
            versions={versions}
            handleVersionSelected={handleVersionSelected}
          />
          {templateString && (
            <div className='z-20'>
              <SidePanel
                isOpen={!!selectedNode}
                title={getNodeName(selectedNode)}
                onClose={() => setSelectedNode(null)}
                defaultWidth={'50%'}
              >
                <div className={commonCss.page}>
                  <div className={padding(20, 'lr')}>
                    <StaticNodeDetailsV2
                      templateString={templateString}
                      layers={layers}
                      onLayerChange={layerChange}
                      element={selectedNode}
                    />
                  </div>
                </div>
              </SidePanel>
            </div>
          )}
        </div>
      )}
      {selectedTab === 1 && (
        <div className={commonCss.codeEditor} data-testid={'spec-ir'}>
          <PipelineSpecTabContent templateString={templateString || ''} />
        </div>
      )}
    </div>
  );
}

export default PipelineDetailsV2;
