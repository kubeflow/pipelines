// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { Meta, StoryObj } from '@storybook/react';
import { useMemo, useState } from 'react';
import '@xyflow/react/dist/style.css';
import DagCanvas from 'src/pages/v2/DagCanvas';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import {
  conditionSpec,
  deeplyNestedSpec,
  exitHandlerSpec,
  loopSpec,
  loopTasks,
  manySubDagsSpec,
  nestedArtifactSpec,
} from 'src/data/test/groupedFlow';
import { V2PipelineTask } from 'src/apisv2/run';
import { convertSubDagToFlowElements } from 'src/lib/v2/StaticFlow';
import type { LayerElementsResolver, LayerFlowElement } from 'src/lib/v2/FlowTypes';
import { createRuntimeLayerResolver } from 'src/lib/v2/DynamicFlow';
import { convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import nestedLoops from 'src/data/test/pipeline_with_loops_and_conditions.yaml?raw';

function GroupedDag({
  spec,
  tasks,
  title,
  nodesDraggable = true,
}: {
  spec: PipelineSpec;
  tasks?: V2PipelineTask[];
  title: string;
  nodesDraggable?: boolean;
}) {
  const [selection, setSelection] = useState<LayerFlowElement>();
  const resolve = useMemo<LayerElementsResolver>(
    () =>
      tasks
        ? createRuntimeLayerResolver(spec, tasks)
        : (layers) => convertSubDagToFlowElements(spec, layers),
    [spec, tasks],
  );
  const elements = useMemo(() => resolve(['root']), [resolve]);
  return (
    <div
      style={{
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
        fontFamily: 'Arial, sans-serif',
        color: '#24364b',
      }}
    >
      <header style={{ padding: '16px 24px', borderBottom: '1px solid #d8e0ea' }}>
        <h2 style={{ margin: '0 0 6px', fontSize: 20 }}>{title}</h2>
        <p style={{ margin: 0, color: '#52677f', fontSize: 13 }}>
          Up to 10 sub-DAGs expand by default · Expand all shows every group · Collapse all hides
          every group
        </p>
      </header>
      <div style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        <DagCanvas
          layers={['root']}
          elements={elements}
          getSubDagElements={resolve}
          nodesDraggable={nodesDraggable}
          selectedElement={selection}
          onElementClick={(_event, selection) => setSelection(selection)}
        />
      </div>
      {selection && (
        <div style={{ padding: 12 }}>
          Selected: {selection.layers.join(' / ')} /{' '}
          {String(selection.element.data?.label ?? selection.element.id)}
        </div>
      )}
    </div>
  );
}

const meta: Meta<typeof GroupedDag> = {
  title: 'v2/GroupedDag',
  component: GroupedDag,
  parameters: { layout: 'fullscreen' },
};
export default meta;
type Story = StoryObj<typeof GroupedDag>;

export const NestedArtifacts: Story = {
  args: { spec: nestedArtifactSpec, title: 'Nested pipelines with artifact dependencies' },
};
export const Conditions: Story = {
  args: { spec: conditionSpec, title: 'Conditional branches with reused components' },
};
export const ParallelIterations: Story = {
  args: {
    spec: loopSpec,
    tasks: loopTasks,
    title: 'ParallelFor — live iteration states and artifacts',
  },
};
export const NestedLoops: Story = {
  args: {
    spec: convertYamlToV2PipelineSpec(nestedLoops),
    title: 'Nested loops — SDK compiler fixture',
  },
};
export const TwentyFiveSubDags: Story = {
  args: {
    spec: manySubDagsSpec(25, 5),
    title: '25 sub-DAGs — five parallel branches',
  },
};

export const TwentyFiveNestedSubDags: Story = {
  args: {
    spec: deeplyNestedSpec(),
    title: '25 nested sub-DAGs — two components per level',
  },
};

export const LargeLoop: Story = {
  args: {
    spec: loopSpec,
    tasks: loopTasks.map((task) =>
      task.task_id === 'sweep' ? { ...task, type_attributes: { iteration_count: '30' } } : task,
    ),
    title: 'Large loop — automatic expansion is bounded',
  },
};
export const DraggableLayers: Story = {
  args: {
    spec: nestedArtifactSpec,
    nodesDraggable: true,
    title: 'Drag inline groups and their tasks',
  },
};

export const ExitHandler: Story = {
  args: {
    spec: exitHandlerSpec,
    title: 'Exit handler with conditional notification',
  },
};
