// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Node, NodeProps } from '@xyflow/react';
import { SubDagFlowElementData } from './Constants';
import { getIcon } from './ExecutionNode';
import { ReadOnlyNodeHandles } from './ReadOnlyNodeHandles';
import { GROUP_HEADER_HEIGHT } from 'src/lib/v2/GroupedFlow';

export default function SubDagGroupNode({
  id,
  data,
  selected,
}: NodeProps<Node<SubDagFlowElementData>>) {
  const collapsed = !!data.collapsed;
  return (
    <>
      <div
        style={{
          width: '100%',
          height: '100%',
          border: `2px solid ${selected ? '#1a73e8' : '#9aafc6'}`,
          borderRadius: 10,
          background: 'rgba(232, 240, 250, 0.45)',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            height: GROUP_HEADER_HEIGHT - 4,
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '0 12px',
            borderBottom: collapsed ? undefined : '1px solid #c8d5e5',
            background: '#edf3fa',
          }}
        >
          <button
            type='button'
            className='nodrag focus:ring'
            title={data.label}
            style={{ flex: 1, minWidth: 0, textAlign: 'left', background: 'none', border: 0 }}
          >
            <span
              title={data.expansionDeferred}
              style={{
                display: 'block',
                fontSize: 11,
                color: '#52677f',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {data.expansionDeferred || data.groupKind || 'Sub-DAG'}
            </span>
            <span
              style={{
                display: 'block',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                fontWeight: 600,
              }}
            >
              {data.label}
            </span>
          </button>
          <div title={data.state} style={{ height: 32 }}>
            {getIcon(data.state)}
          </div>
          <button
            type='button'
            className='nodrag nopan focus:ring rounded'
            aria-label={`${collapsed ? 'Expand' : 'Collapse'} ${data.label}`}
            aria-expanded={!collapsed}
            title={`${collapsed ? 'Expand' : 'Collapse'} sub-DAG`}
            onClick={(event) => {
              event.stopPropagation();
              data.expand(id);
            }}
            style={{
              background: 'white',
              border: '1px solid #9aafc6',
              color: '#315b86',
              width: 28,
              height: 28,
            }}
          >
            {collapsed ? <ExpandMoreIcon fontSize='small' /> : <ExpandLessIcon fontSize='small' />}
          </button>
        </div>
        {!collapsed && (data.expansionError || data.empty) && (
          <div style={{ padding: 12, fontSize: 12, color: '#52677f' }} title={data.expansionError}>
            {data.expansionError
              ? 'Unable to display sub-DAG. Open its details to inspect the spec.'
              : 'No tasks in this scope'}
          </div>
        )}
      </div>
      <ReadOnlyNodeHandles />
    </>
  );
}
