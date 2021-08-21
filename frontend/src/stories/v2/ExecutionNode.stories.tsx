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

import { ComponentMeta, ComponentStory } from '@storybook/react';
import React from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  OnLoadParams,
  ReactFlowProvider,
} from 'react-flow-renderer';
import ExecutionNodeFailed from 'src/components/graph/ExecutionNodeFailed';
import ExecutionNodePending from 'src/components/graph/ExecutionNodePending';
import ExecutionNode from '../../components/graph/ExecutionNode';
import './ExecutionNode.css';

const nodeTypes = {
  execution: ExecutionNode,
  executionPending: ExecutionNodePending,
  executionFailed: ExecutionNodeFailed,
};

interface WrappedExecutionNodeProps {
  id: string;
  label: string;
}

function WrappedExecutionNode({ id, label }: WrappedExecutionNodeProps) {
  const onLoad = (reactFlowInstance: OnLoadParams) => {
    reactFlowInstance.fitView();
  };

  const elements = [
    {
      id: id,
      type: 'execution',
      position: { x: 100, y: 100 },
      data: { label },
    },
  ];

  return (
    <div style={{ width: '1350px', height: '550px' }}>
      <ReactFlowProvider>
        <ReactFlow
          className='bg-gray-100'
          elements={elements}
          snapToGrid={true}
          nodeTypes={nodeTypes}
          edgeTypes={{}}
          onLoad={onLoad}
        >
          <MiniMap />
          <Controls />
          <Background />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
}

export default {
  title: 'v2/ExecutionNode',
  component: WrappedExecutionNode,
  argTypes: {
    backgroundColor: { control: 'color' },
  },
} as ComponentMeta<typeof WrappedExecutionNode>;

const Template: ComponentStory<typeof WrappedExecutionNode> = args => (
  <WrappedExecutionNode {...args} />
);

export const Primary = Template.bind({});
Primary.args = {
  id: 'id',
  label: 'This is an ExecutionNode',
};

export const Secondary = Template.bind({});
Secondary.args = {
  id: 'id',
  label: 'This is an ExecutionNode with long name',
};
