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
  ReactFlowInstance,
  ReactFlowProvider,
} from 'reactflow';
import 'src/build/tailwind.output.css';
import SubDagNode from '../../components/graph/SubDagNode';

const nodeTypes = {
  subDag: SubDagNode,
};

interface WrappedSubDagNodeProps {
  id: string;
  label: string;
}

function WrappedSubDagNode({ id, label }: WrappedSubDagNodeProps) {
  const onInit = (reactFlowInstance: ReactFlowInstance) => {
    reactFlowInstance.fitView();
  };

  const nodes = [
    {
      id: id,
      type: 'subDag',
      position: { x: 100, y: 100 },
      data: { label },
    },
  ];

  return (
    <div
      // className='flex container mx-auto'
      data-testid='StaticCanvas'
      style={{ width: '1200px', height: '600px' }}
    >
      <ReactFlowProvider>
        <ReactFlow
          style={{ background: '#F5F5F5' }}
          nodes={nodes}
          edges={[]}
          snapToGrid={true}
          nodeTypes={nodeTypes}
          edgeTypes={{}}
          onInit={onInit}
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
  title: 'v2/SubDagNode',
  component: WrappedSubDagNode,
  argTypes: {
    backgroundColor: { control: 'color' },
  },
} as ComponentMeta<typeof WrappedSubDagNode>;

const Template: ComponentStory<typeof WrappedSubDagNode> = args => <WrappedSubDagNode {...args} />;

export const Primary = Template.bind({});
Primary.args = {
  id: 'id',
  label: 'This is a SubDagNode',
};

export const Secondary = Template.bind({});
Secondary.args = {
  id: 'id',
  label: 'This is a SubDagNode with long name',
};
