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
import { PipelineFlowElement } from 'src/lib/v2/StaticFlow';
import { commonCss } from '../Css';
import StaticCanvas from './v2/StaticCanvas';

interface PipelineDetailsV2Props {
  pipelineFlowElements: PipelineFlowElement[];
  setSubDagLayers: (layers: string[]) => void;
}

const PipelineDetailsV2: React.FC<PipelineDetailsV2Props> = ({
  pipelineFlowElements,
  setSubDagLayers,
}: PipelineDetailsV2Props) => {
  const [layers, setLayers] = useState(['root']);

  const layerChange = (l: string[]) => {
    setLayers(l);
    setSubDagLayers(l);
  };

  return (
    <div className={commonCss.page} data-testid={'pipeline-detail-v2'}>
      <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
        <StaticCanvas
          layers={layers}
          onLayersUpdate={layerChange}
          elements={pipelineFlowElements}
        ></StaticCanvas>
      </div>
    </div>
  );
};

export default PipelineDetailsV2;
