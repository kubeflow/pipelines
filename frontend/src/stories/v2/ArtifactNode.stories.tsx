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
import 'src/build/tailwind.output.css';
import ArtifactNode from '../../components/graph/ArtifactNode';

const nodeTypes = {
  artifact: ArtifactNode,
};

interface WrappedArtifactNodeProps {
  id: string;
  label: string;
  hasArtifact: boolean;
}

function WrappedArtifactNode({ id, label, hasArtifact }: WrappedArtifactNodeProps) {
  const nodes = [
    {
      id: id,
      type: 'artifact',
      position: { x: 100, y: 100 },
      data: { label, hasArtifact },
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

const meta: Meta<typeof WrappedArtifactNode> = {
  title: 'v2/ArtifactNode',
  component: WrappedArtifactNode,
};

export default meta;
type Story = StoryObj<typeof WrappedArtifactNode>;

export const Primary: Story = {
  args: {
    id: 'id',
    label: 'This is an ArtifactNode',
    hasArtifact: true,
  },
};

export const Secondary: Story = {
  args: {
    id: 'id',
    label: 'This is an ArtifactNode with long name',
  },
};
