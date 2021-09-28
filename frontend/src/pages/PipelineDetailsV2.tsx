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
import React, { useState } from 'react';
import { ApiPipeline, ApiPipelineVersion } from 'src/apis/pipeline';
import MD2Tabs from 'src/atoms/MD2Tabs';
import Editor from 'src/components/Editor';
import { PipelineVersionCard } from 'src/components/navigators/PipelineVersionCard';
import { isSafari } from 'src/lib/Utils';
import { PipelineFlowElement } from 'src/lib/v2/StaticFlow';
import { commonCss } from '../Css';
import StaticCanvas from './v2/StaticCanvas';

const TAB_NAMES = ['Graph', 'Pipeline Spec'];

interface PipelineDetailsV2Props {
  templateString?: string;
  pipelineFlowElements: PipelineFlowElement[];
  setSubDagLayers: (layers: string[]) => void;
  apiPipeline: ApiPipeline | null;
  selectedVersion: ApiPipelineVersion | undefined;
  versions: ApiPipelineVersion[];
  handleVersionSelected: (versionId: string) => Promise<void>;
}

function PipelineDetailsV2({
  templateString,
  pipelineFlowElements,
  setSubDagLayers,
  apiPipeline,
  selectedVersion,
  versions,
  handleVersionSelected,
}: PipelineDetailsV2Props) {
  const [layers, setLayers] = useState(['root']);
  const [selectedTab, setSelectedTab] = useState(0);

  const layerChange = (l: string[]) => {
    setLayers(l);
    setSubDagLayers(l);
  };
  const editorHeightWidth = isSafari() ? '640px' : '100%';

  return (
    <div className={commonCss.page} data-testid={'pipeline-detail-v2'}>
      <MD2Tabs selectedTab={selectedTab} onSwitch={setSelectedTab} tabs={TAB_NAMES} />
      {selectedTab === 0 && (
        <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
          <StaticCanvas
            layers={layers}
            onLayersUpdate={layerChange}
            elements={pipelineFlowElements}
          ></StaticCanvas>
          <PipelineVersionCard
            apiPipeline={apiPipeline}
            selectedVersion={selectedVersion}
            versions={versions}
            handleVersionSelected={handleVersionSelected}
          />
        </div>
      )}
      {selectedTab === 1 && (
        <div className={commonCss.codeEditor} data-testid={'spec-ir'}>
          <Editor
            value={templateString || ''}
            height={editorHeightWidth}
            width={editorHeightWidth}
            mode='json'
            theme='github'
            editorProps={{ $blockScrolling: true }}
            readOnly={true}
            highlightActiveLine={true}
            showGutter={true}
          />
        </div>
      )}
    </div>
  );
}

export default PipelineDetailsV2;
