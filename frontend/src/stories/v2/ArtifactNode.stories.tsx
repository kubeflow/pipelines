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
import 'src/build/tailwind.output.css';
import { color } from 'src/Css';
import { Artifact } from 'src/third_party/mlmd';
import ArtifactNode from '../../components/graph/ArtifactNode';

const nodeTypes = {
  artifact: ArtifactNode,
};

interface WrappedArtifactNodeProps {
  id: string;
  label: string;
  state: Artifact.State;
}

function WrappedArtifactNode({ id, label, state }: WrappedArtifactNodeProps) {
  const onLoad = (reactFlowInstance: OnLoadParams) => {
    reactFlowInstance.fitView();
  };

  const elements = [
    {
      id: id,
      type: 'artifact',
      position: { x: 100, y: 100 },
      data: { label, state },
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
  title: 'v2/ArtifactNode',
  component: WrappedArtifactNode,
  argTypes: {
    backgroundColor: { control: 'color' },
  },
} as ComponentMeta<typeof WrappedArtifactNode>;

const Template: ComponentStory<typeof WrappedArtifactNode> = args => (
  <WrappedArtifactNode {...args} />
);

export const Primary = Template.bind({});
Primary.args = {
  id: 'id',
  label: 'This is an ArtifactNode',
  state: Artifact.State.LIVE,
};

export const Secondary = Template.bind({});
Secondary.args = {
  id: 'id',
  label: 'This is an ArtifactNode with long name',
};
