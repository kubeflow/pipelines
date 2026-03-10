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

import { Meta, StoryObj } from '@storybook/react';
import React from 'react';
import { ReactFlow, ReactFlowProvider, Background, Controls, MiniMap } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import 'src/build/tailwind.output.css';
import {
  ArtifactFlowElementData,
  ExecutionFlowElementData,
  FlowElementDataBase,
} from 'src/components/graph/Constants';
import { NodeTypeNames, NODE_TYPES } from 'src/lib/v2/StaticFlow';
import { Artifact, Execution } from 'src/third_party/mlmd';

const nodes = [
  {
    id: '2',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 100 },
    data: { label: 'Default execution node' } as ExecutionFlowElementData,
  },
  {
    id: '3',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 200 },
    data: {
      label: 'UNKNOWN execution node',
      state: Execution.State.UNKNOWN,
    } as ExecutionFlowElementData,
  },
  {
    id: '4',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 300 },
    data: {
      label: 'NEW execution node',
      state: Execution.State.NEW,
    } as ExecutionFlowElementData,
  },
  {
    id: '5',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 400 },
    data: {
      label: 'RUNNING execution node',
      state: Execution.State.RUNNING,
    } as ExecutionFlowElementData,
  },
  {
    id: '6',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 500 },
    data: {
      label: 'COMPLETE execution node',
      state: Execution.State.COMPLETE,
    } as ExecutionFlowElementData,
  },
  {
    id: '7',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 600 },
    data: {
      label: 'CACHED execution node',
      state: Execution.State.CACHED,
    } as ExecutionFlowElementData,
  },
  {
    id: '8',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 700 },
    data: {
      label: 'CANCELED execution node',
      state: Execution.State.CANCELED,
    } as ExecutionFlowElementData,
  },
  {
    id: '9',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 800 },
    data: {
      label: 'FAILED execution node',
      state: Execution.State.FAILED,
    } as ExecutionFlowElementData,
  },
  {
    id: '9',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 900 },
    data: {
      label: 'invalid execution node',
      state: 8 as Execution.State,
    } as ExecutionFlowElementData,
  },
  {
    id: '101',
    type: NodeTypeNames.ARTIFACT,
    position: { x: 400, y: 100 },
    data: {
      label: 'DEFAULT artifact node',
      state: Artifact.State.UNKNOWN,
    } as ArtifactFlowElementData,
  },
  {
    id: '102',
    type: NodeTypeNames.ARTIFACT,
    position: { x: 400, y: 200 },
    data: {
      label: 'LIVE artifact node',
      state: Artifact.State.LIVE,
    } as ArtifactFlowElementData,
  },
  {
    id: '201',
    type: NodeTypeNames.SUB_DAG,
    position: { x: 700, y: 72 },
    data: {
      label: 'Sub-DAG node',
    } as FlowElementDataBase,
  },
];

function WrappedNodeGallery({}) {
  return (
    <div style={{ width: '1200px', height: '1000px' }}>
      {/* // className='flex container mx-auto' */}
      <ReactFlowProvider>
        <ReactFlow
          style={{ background: '#F5F5F5' }}
          nodes={nodes}
          edges={[]}
          snapToGrid={true}
          nodeTypes={NODE_TYPES}
          edgeTypes={{}}
          onInit={(instance) => instance.fitView()}
        >
          <MiniMap />
          <Controls />
          <Background />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
}

const meta: Meta<typeof WrappedNodeGallery> = {
  title: 'v2/NodeGallery',
  component: WrappedNodeGallery,
  argTypes: {
    backgroundColor: { control: 'color' },
  },
};

export default meta;
type Story = StoryObj<typeof WrappedNodeGallery>;

export const Primary: Story = {
  args: {},
};
