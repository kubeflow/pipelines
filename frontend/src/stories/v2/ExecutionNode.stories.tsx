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
import { ReactFlow, ReactFlowProvider, Background, Controls, MiniMap } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { NodeTypeNames, NODE_TYPES } from 'src/lib/v2/StaticFlow';
import { Execution } from 'src/third_party/mlmd';

interface WrappedExecutionNodeProps {
  id: string;
  label: string;
  state: Execution.State;
}

function WrappedExecutionNode({ id, label, state }: WrappedExecutionNodeProps) {
  const nodes = [
    {
      id: id,
      type: NodeTypeNames.EXECUTION,
      position: { x: 100, y: 100 },
      data: { label, state },
    },
  ];

  return (
    <div style={{ width: '1350px', height: '550px' }}>
      <ReactFlowProvider>
        <ReactFlow
          className='bg-gray-100'
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

const meta: Meta<typeof WrappedExecutionNode> = {
  title: 'v2/ExecutionNode',
  component: WrappedExecutionNode,
  argTypes: {
    backgroundColor: { control: 'color' },
  },
};

export default meta;
type Story = StoryObj<typeof WrappedExecutionNode>;

export const Primary: Story = {
  args: {
    id: 'id',
    label: 'This is an ExecutionNode',
    state: Execution.State.NEW,
  },
};

export const Secondary: Story = {
  args: {
    id: 'id',
    label: 'This is an ExecutionNode with long name',
  },
};
