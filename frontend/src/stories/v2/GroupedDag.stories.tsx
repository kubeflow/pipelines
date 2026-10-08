// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { Meta, StoryObj } from '@storybook/react';
import { useMemo, useState } from 'react';
import '@xyflow/react/dist/style.css';
import DagCanvas from 'src/pages/v2/DagCanvas';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import {
  conditionSpec,
  exitHandlerSpec,
  loopSpec,
  loopTasks,
  nestedArtifactSpec,
} from 'src/data/test/groupedFlow';
import { V2beta1PipelineTask } from 'src/apisv2beta1/run';
import { convertSubDagToFlowElements, PipelineFlowElement } from 'src/lib/v2/StaticFlow';
import { createRuntimeLayerResolver } from 'src/lib/v2/DynamicFlow';
import { convertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import nestedLoops from 'src/data/test/pipeline_with_loops_and_conditions.yaml?raw';

function GroupedDag({
  spec,
  tasks,
  title,
  initialLayers = ['root'],
  nodesDraggable = true,
}: {
  spec: PipelineSpec;
  tasks?: V2beta1PipelineTask[];
  title: string;
  initialLayers?: string[];
  nodesDraggable?: boolean;
}) {
  const [layers, setLayers] = useState(initialLayers);
  const [selection, setSelection] = useState<{ element: PipelineFlowElement; layers: string[] }>();
  const resolve = useMemo(
    () =>
      tasks
        ? createRuntimeLayerResolver(spec, tasks)
        : (scope: string[], maxNodes?: number) =>
            convertSubDagToFlowElements(spec, scope, maxNodes),
    [spec, tasks],
  );
  const elements = useMemo(() => resolve(layers), [resolve, layers]);
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
          Expanded sub-DAGs · Select a task for details · Use the controls to expand or collapse
          groups
        </p>
      </header>
      <div style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        <DagCanvas
          layers={layers}
          onLayersUpdate={setLayers}
          elements={elements}
          getSubDagElements={resolve}
          setFlowElements={() => {}}
          nodesDraggable={nodesDraggable}
          selectedNodeId={selection?.element.id}
          selectedNodeLayers={selection?.layers}
          onElementClick={(_event, element, scope) => setSelection({ element, layers: scope })}
        />
      </div>
      {selection && (
        <div style={{ padding: 12 }}>
          Selected: {selection.layers.join(' / ')} /{' '}
          {String(selection.element.data?.label ?? selection.element.id)}
          <button
            style={{ marginLeft: 16 }}
            onClick={() => {
              setLayers(selection.layers);
              setSelection(undefined);
            }}
          >
            Focus selected scope
          </button>
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
export const LargeLoop: Story = {
  args: {
    spec: loopSpec,
    tasks: loopTasks.map((task) =>
      task.task_id === 'sweep' ? { ...task, type_attributes: { iteration_count: '10000' } } : task,
    ),
    title: 'Large loop — automatic expansion is bounded',
  },
};
export const DraggableScopes: Story = {
  args: {
    spec: nestedArtifactSpec,
    initialLayers: ['root', 'workflow', 'fit'],
    nodesDraggable: true,
    title: 'Drag positions in focused and nested scopes',
  },
};

export const ExitHandler: Story = {
  args: {
    spec: exitHandlerSpec,
    title: 'Exit handler with conditional notification',
  },
};
