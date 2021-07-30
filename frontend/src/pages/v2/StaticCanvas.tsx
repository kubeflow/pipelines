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

import React from 'react';
import ReactFlow, {
  Background,
  Controls,
  Elements,
  MiniMap,
  ReactFlowProvider,
} from 'react-flow-renderer';
import { color } from 'src/Css';

export interface StaticCanvasProps {
  elements: Elements;
}

const StaticCanvas = ({ elements }: StaticCanvasProps) => {
  return (
    <div data-testid='StaticCanvas'>
      <ReactFlowProvider>
        <ReactFlow
          style={{ background: color.lightGrey }}
          elements={elements}
          snapToGrid={true}
          nodeTypes={{}}
          edgeTypes={{}}
        >
          <MiniMap />
          <Controls />
          <Background />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
};
export default StaticCanvas;
