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
import { useState } from 'react';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { InspectionTabs } from 'src/components/inspection/InspectionTabs';
import { PipelineVersionCard } from 'src/components/navigators/PipelineVersionCard';
import { PipelineSpecTabContent } from 'src/components/PipelineSpecTabContent';
import { InspectionPanel } from 'src/components/inspection/InspectionPanel';
import { StaticNodeDetails } from 'src/components/tabs/StaticNodeDetails';
import { getNodeName, PipelineFlowElement } from 'src/lib/v2/StaticFlow';

import 'src/components/pipelines/Pipelines.css';
import DagCanvas from './v2/DagCanvas';

const TAB_NAMES = ['Graph', 'Pipeline Spec'];

interface PipelineDetailsViewProps {
  templateString?: string;
  pipelineFlowElements: PipelineFlowElement[];
  setSubDagLayers: (layers: string[]) => void;
  pipeline: V2beta1Pipeline | null;
  selectedVersion: V2beta1PipelineVersion | undefined;
  versions: V2beta1PipelineVersion[];
  handleVersionSelected: (versionId: string) => Promise<void>;
}

function PipelineDetailsView({
  templateString,
  pipelineFlowElements,
  setSubDagLayers,
  pipeline,
  selectedVersion,
  versions,
  handleVersionSelected,
}: PipelineDetailsViewProps) {
  const [layers, setLayers] = useState(['root']);
  const [selectedTab, setSelectedTab] = useState(0);
  const [selectedNode, setSelectedNode] = useState<PipelineFlowElement | null>(null);

  const layerChange = (l: string[]) => {
    setSelectedNode(null);
    setLayers(l);
    setSubDagLayers(l);
  };

  return (
    <div className='kfp-pipeline-detail' data-testid='pipeline-detail-v2'>
      <InspectionTabs
        selectedTab={selectedTab}
        onSwitch={setSelectedTab}
        tabs={TAB_NAMES}
        ariaLabel='Pipeline details'
      >
        {selectedTab === 0 && (
          <div className='kfp-pipeline-graph'>
            <DagCanvas
              layers={layers}
              onLayersUpdate={layerChange}
              elements={pipelineFlowElements}
              onElementClick={(_event, element) => setSelectedNode(element)}
              setFlowElements={() => {}}
              nodesDraggable={false}
            ></DagCanvas>
            <PipelineVersionCard
              pipeline={pipeline}
              selectedVersion={selectedVersion}
              versions={versions}
              handleVersionSelected={handleVersionSelected}
            />
            {templateString && (
              <div className='z-20'>
                <InspectionPanel
                  isOpen={!!selectedNode}
                  title={getNodeName(selectedNode)}
                  onClose={() => setSelectedNode(null)}
                >
                  <div className='kfp-pipeline-node-details'>
                    <div>
                      <StaticNodeDetails
                        templateString={templateString}
                        layers={layers}
                        onLayerChange={layerChange}
                        element={selectedNode}
                      />
                    </div>
                  </div>
                </InspectionPanel>
              </div>
            )}
          </div>
        )}
        {selectedTab === 1 && (
          <div className='kfp-pipeline-spec' data-testid='spec-ir'>
            <PipelineSpecTabContent templateString={templateString || ''} />
          </div>
        )}
      </InspectionTabs>
    </div>
  );
}

export default PipelineDetailsView;
