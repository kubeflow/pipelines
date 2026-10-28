// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { ButtonHTMLAttributes, useId } from 'react';
import { ControlButton, Panel, useReactFlow, useStore } from '@xyflow/react';
import { GRAPH_FIT_OPTIONS } from './Constants';
import { Tooltip } from '@base-ui/react/tooltip';
import {
  Plus as AddIcon,
  Minus as RemoveIcon,
  Scan as CropFreeIcon,
  Lock as LockIcon,
  LockOpen as LockOpenIcon,
  ChevronsUpDown as UnfoldMoreIcon,
  ChevronsDownUp as UnfoldLessIcon,
} from 'lucide-react';

type PaletteButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'title'> & {
  label: string;
  tooltip?: string;
};

function PaletteButton({ label, tooltip = label, ...props }: PaletteButtonProps) {
  const tooltipId = useId();
  return (
    <Tooltip.Root disableHoverablePopup>
      <Tooltip.Trigger render={<span style={{ display: 'flex' }} />} delay={500}>
        <ControlButton
          {...props}
          aria-label={label}
          aria-describedby={tooltipId}
          style={{
            borderBottom:
              '1px solid var(--xy-controls-button-border-color, var(--xy-controls-button-border-color-default))',
          }}
        />
      </Tooltip.Trigger>
      <Tooltip.Portal>
        <Tooltip.Positioner side='right' sideOffset={6}>
          <Tooltip.Popup role='tooltip' id={tooltipId} className='kfp-graph-tooltip'>
            {tooltip}
          </Tooltip.Popup>
        </Tooltip.Positioner>
      </Tooltip.Portal>
    </Tooltip.Root>
  );
}

export interface GraphControlsProps {
  showSubDagControls: boolean;
  hasSubDags: boolean;
  canExpand?: boolean;
  canCollapse?: boolean;
  locked: boolean;
  onLockChange: (locked: boolean) => void;
  onExpandAll: () => void;
  onCollapseAll: () => void;
  onViewportInteraction?: () => void;
}

export default function GraphControls({
  showSubDagControls,
  hasSubDags,
  canExpand = true,
  canCollapse = true,
  locked,
  onLockChange,
  onExpandAll,
  onCollapseAll,
  onViewportInteraction,
}: GraphControlsProps) {
  const { zoomIn, zoomOut, fitView } = useReactFlow();
  const maxZoomReached = useStore((state) => state.transform[2] >= state.maxZoom);
  const minZoomReached = useStore((state) => state.transform[2] <= state.minZoom);
  return (
    <Panel
      position='bottom-left'
      className='react-flow__controls'
      role='group'
      aria-label='Graph controls'
      data-testid='rf__controls'
    >
      {showSubDagControls && hasSubDags && (
        <>
          <PaletteButton
            label='Expand all'
            tooltip={canExpand ? 'Expand all sub-DAGs' : 'All sub-DAGs are expanded'}
            disabled={!canExpand}
            onClick={onExpandAll}
          >
            <UnfoldMoreIcon />
          </PaletteButton>
          <PaletteButton
            label='Collapse all'
            tooltip={canCollapse ? 'Collapse all sub-DAGs' : 'All sub-DAGs are collapsed'}
            disabled={!canCollapse}
            onClick={onCollapseAll}
          >
            <UnfoldLessIcon />
          </PaletteButton>
        </>
      )}
      <PaletteButton
        label='Zoom in'
        className='react-flow__controls-zoomin'
        disabled={maxZoomReached}
        onClick={() => {
          onViewportInteraction?.();
          void zoomIn();
        }}
      >
        <AddIcon />
      </PaletteButton>
      <PaletteButton
        label='Zoom out'
        className='react-flow__controls-zoomout'
        disabled={minZoomReached}
        onClick={() => {
          onViewportInteraction?.();
          void zoomOut();
        }}
      >
        <RemoveIcon />
      </PaletteButton>
      <PaletteButton
        label='Fit view'
        tooltip='Fit graph to view (F)'
        aria-keyshortcuts='F'
        className='react-flow__controls-fitview'
        onClick={() => {
          onViewportInteraction?.();
          void fitView(GRAPH_FIT_OPTIONS);
        }}
      >
        <CropFreeIcon />
      </PaletteButton>
      <PaletteButton
        label={locked ? 'Unlock node positions' : 'Lock node positions'}
        tooltip={locked ? 'Unlock node dragging' : 'Lock nodes to pan with dragging and arrow keys'}
        aria-pressed={locked}
        className='react-flow__controls-interactive'
        onClick={() => onLockChange(!locked)}
      >
        {locked ? <LockIcon /> : <LockOpenIcon />}
      </PaletteButton>
    </Panel>
  );
}
