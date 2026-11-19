// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { type ReactNode, useCallback, useMemo } from 'react';
import { commonCss, padding } from 'src/Css';
import SidePanel from 'src/components/SidePanel';
import type {
  LayerElementsResolver,
  LayerFlowElement,
  PipelineFlowElement,
} from 'src/lib/v2/FlowTypes';
import { getNodeName } from 'src/lib/v2/StaticFlow';
import DagCanvas from 'src/pages/v2/DagCanvas';

const ROOT_LAYERS = ['root'];
interface PipelineGraphProps {
  elements: PipelineFlowElement[];
  selection: LayerFlowElement | null;
  onSelectionChange: (selection: LayerFlowElement | null) => void;
  graphId?: string;
  resolveLayers?: LayerElementsResolver;
  renderDetails?: (selection: LayerFlowElement) => ReactNode;
  children?: ReactNode;
}

/** Graph and inspector shared by native run details and offline exports. */
export default function PipelineGraph({
  elements,
  selection,
  onSelectionChange,
  graphId,
  resolveLayers,
  renderDetails,
  children,
}: PipelineGraphProps) {
  const getSubDagElements = useCallback<LayerElementsResolver>(
    (layers) => resolveLayers?.(layers) ?? [],
    [resolveLayers],
  );
  const currentSelection = useMemo(() => {
    if (!selection) return null;
    try {
      const layer = selection.layers.length === 1 ? elements : getSubDagElements(selection.layers);
      const element = layer.find((candidate) => candidate.id === selection.element.id);
      return element ? { ...selection, element } : null;
    } catch {
      return null;
    }
  }, [selection, elements, getSubDagElements]);
  return (
    <div className={commonCss.page} style={{ position: 'relative', overflow: 'hidden' }}>
      <DagCanvas
        graphId={graphId}
        layers={ROOT_LAYERS}
        elements={elements}
        getSubDagElements={resolveLayers ? getSubDagElements : undefined}
        selectedElement={currentSelection}
        onElementClick={(_event, selected) => onSelectionChange(selected)}
      />
      {children}
      {renderDetails && (
        <div className='z-20'>
          <SidePanel
            isOpen={!!currentSelection}
            title={getNodeName(currentSelection?.element ?? null)}
            onClose={() => onSelectionChange(null)}
            defaultWidth='50%'
          >
            <div className={commonCss.page}>
              <div className={padding(20, 'lr')}>
                {currentSelection && renderDetails(currentSelection)}
              </div>
            </div>
          </SidePanel>
        </div>
      )}
    </div>
  );
}
