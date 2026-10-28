/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { type ReactNode, useCallback, useMemo } from 'react';
import { InspectionPanel } from 'src/components/inspection/InspectionPanel';
import { StaticNodeDetails } from 'src/components/tabs/StaticNodeDetails';
import 'src/components/pipelines/Pipelines.css';
import type {
  LayerElementsResolver,
  LayerFlowElement,
  PipelineFlowElement,
} from 'src/lib/v2/FlowTypes';
import { convertSubDagToFlowElements, getNodeName } from 'src/lib/v2/StaticFlow';
import { tryConvertYamlToV2PipelineSpec } from 'src/lib/v2/WorkflowUtils';
import DagCanvas from 'src/pages/v2/DagCanvas';

const ROOT_LAYERS = ['root'];

interface PipelineGraphProps {
  templateString?: string;
  elements: PipelineFlowElement[];
  selection: LayerFlowElement | null;
  onSelectionChange: (selection: LayerFlowElement | null) => void;
  /** Omit for an ephemeral preview, without persisted expansion preferences. */
  graphId?: string;
  children?: ReactNode;
}

/** The pipeline graph and inspector shared by the application and offline exports. */
export default function PipelineGraph({
  templateString,
  elements,
  selection,
  onSelectionChange,
  graphId,
  children,
}: PipelineGraphProps) {
  const pipelineSpec = useMemo(
    () => (templateString ? tryConvertYamlToV2PipelineSpec(templateString) : undefined),
    [templateString],
  );
  const getSubDagElements = useCallback<LayerElementsResolver>(
    (layers) => (pipelineSpec ? convertSubDagToFlowElements(pipelineSpec, layers) : []),
    [pipelineSpec],
  );
  const currentSelection = useMemo(() => {
    if (!selection) return null;
    // A live update can rename a selected node or remove its containing DAG.
    try {
      const layer = selection.layers.length === 1 ? elements : getSubDagElements(selection.layers);
      const element = layer.find((candidate) => candidate.id === selection.element.id);
      return element ? { ...selection, element } : null;
    } catch {
      return null;
    }
  }, [selection, elements, getSubDagElements]);

  return (
    <div className='kfp-pipeline-graph'>
      <DagCanvas
        graphId={graphId}
        layers={ROOT_LAYERS}
        elements={elements}
        getSubDagElements={pipelineSpec ? getSubDagElements : undefined}
        selectedElement={currentSelection}
        onElementClick={(_event, selected) => onSelectionChange(selected)}
      />
      {children}
      {templateString && (
        <InspectionPanel
          isOpen={!!currentSelection}
          title={getNodeName(currentSelection?.element ?? null)}
          onClose={() => onSelectionChange(null)}
        >
          <div className='kfp-pipeline-node-details'>
            <StaticNodeDetails
              templateString={templateString}
              layers={currentSelection?.layers ?? ROOT_LAYERS}
              element={currentSelection?.element ?? null}
            />
          </div>
        </InspectionPanel>
      )}
    </div>
  );
}
