// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { ButtonHTMLAttributes } from 'react';
import { ControlButton, Panel, useReactFlow, useStore } from '@xyflow/react';
import Tooltip from '@mui/material/Tooltip';
import AddIcon from '@mui/icons-material/Add';
import RemoveIcon from '@mui/icons-material/Remove';
import CropFreeIcon from '@mui/icons-material/CropFree';
import LockIcon from '@mui/icons-material/Lock';
import LockOpenIcon from '@mui/icons-material/LockOpen';
import FullscreenIcon from '@mui/icons-material/Fullscreen';
import FullscreenExitIcon from '@mui/icons-material/FullscreenExit';
import UnfoldMoreIcon from '@mui/icons-material/UnfoldMore';
import UnfoldLessIcon from '@mui/icons-material/UnfoldLess';

type PaletteButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'title'> & {
  label: string;
  tooltip?: string;
};

function PaletteButton({ label, tooltip = label, ...props }: PaletteButtonProps) {
  return (
    <Tooltip
      title={tooltip}
      enterDelay={500}
      enterNextDelay={500}
      placement='right'
      disableInteractive
    >
      {/* Disabled buttons do not emit mouse events; the wrapper still explains the action. */}
      <span style={{ display: 'flex' }}>
        <ControlButton
          {...props}
          aria-label={label}
          style={{
            borderBottom:
              '1px solid var(--xy-controls-button-border-color, var(--xy-controls-button-border-color-default))',
          }}
        />
      </span>
    </Tooltip>
  );
}

export interface GraphControlsProps {
  showSubDagControls: boolean;
  hasSubDags: boolean;
  renderSubdags: boolean;
  onRenderSubdagsChange: (enabled: boolean) => void;
  locked: boolean;
  onLockChange: (locked: boolean) => void;
  onExpandAll: () => void;
  onCollapseAll: () => void;
}

export default function GraphControls({
  showSubDagControls,
  hasSubDags,
  renderSubdags,
  onRenderSubdagsChange,
  locked,
  onLockChange,
  onExpandAll,
  onCollapseAll,
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
      {showSubDagControls && (
        <>
          <PaletteButton
            label='Expand all'
            tooltip={
              renderSubdags ? 'Expand all sub-DAGs' : 'Enable sub-DAG rendering to expand all'
            }
            disabled={!renderSubdags || !hasSubDags}
            onClick={onExpandAll}
          >
            <UnfoldMoreIcon />
          </PaletteButton>
          <PaletteButton
            label='Collapse all'
            tooltip={
              renderSubdags ? 'Collapse all sub-DAGs' : 'Enable sub-DAG rendering to collapse all'
            }
            disabled={!renderSubdags || !hasSubDags}
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
          void zoomOut();
        }}
      >
        <RemoveIcon />
      </PaletteButton>
      <PaletteButton
        label='Fit view'
        tooltip='Fit graph to view'
        className='react-flow__controls-fitview'
        onClick={() => {
          void fitView();
        }}
      >
        <CropFreeIcon />
      </PaletteButton>
      <PaletteButton
        label={locked ? 'Unlock graph' : 'Lock graph'}
        tooltip={locked ? 'Unlock node dragging and selection' : 'Lock node dragging and selection'}
        className='react-flow__controls-interactive'
        onClick={() => onLockChange(!locked)}
      >
        {locked ? <LockIcon /> : <LockOpenIcon />}
      </PaletteButton>
      {showSubDagControls && (
        <PaletteButton
          label='Render subdags'
          aria-pressed={renderSubdags}
          tooltip={renderSubdags ? 'Disable inline sub-DAG rendering' : 'Render sub-DAGs inline'}
          onClick={() => onRenderSubdagsChange(!renderSubdags)}
        >
          {renderSubdags ? <FullscreenExitIcon /> : <FullscreenIcon />}
        </PaletteButton>
      )}
    </Panel>
  );
}
