// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { ChevronsDownUp as UnfoldLessIcon, ChevronsUpDown as UnfoldMoreIcon } from 'lucide-react';
import { Node, NodeProps } from '@xyflow/react';
import { SubDagFlowElementData } from './Constants';
import { TaskStateIndicator } from './ExecutionNode';
import { getTaskStatus } from 'src/lib/StatusUtils';
import { ReadOnlyNodeHandles } from './ReadOnlyNodeHandles';
import SubDagTypeIcon from './SubDagTypeIcon';
import { GROUP_HEADER_HEIGHT } from 'src/lib/v2/GroupedFlow';

export const SUB_DAG_HEADER_CLASS_NAME = 'subdag-header';

export default function SubDagGroupNode({
  id,
  data,
  selected,
  draggable = false,
}: Pick<NodeProps<Node<SubDagFlowElementData>>, 'id' | 'data' | 'selected'> & {
  draggable?: boolean;
}) {
  const collapsed = !!data.collapsed;
  const status = getTaskStatus(data.state);
  return (
    <>
      <div
        data-testid='subdag-box'
        className='kfp-subdag-box'
        data-selected={selected}
        data-tone={status.tone}
      >
        <div
          data-testid='subdag-header'
          className={`${SUB_DAG_HEADER_CLASS_NAME} kfp-subdag-header`}
          style={{ height: data.headerHeight ?? GROUP_HEADER_HEIGHT, pointerEvents: 'auto' }}
          title={data.expansionError}
        >
          <div
            data-testid='subdag-type'
            className='w-8 pl-2 h-full flex items-center flex-shrink-0'
          >
            <SubDagTypeIcon kind={data.groupKind} />
          </div>
          <button
            type='button'
            className='focus:outline-none focus:ring-0 px-3 flex flex-1 min-w-0 items-center justify-center text-sm'
            title={data.label}
            style={{ background: 'transparent', border: 0, fontWeight: 400 }}
          >
            <span className='truncate' data-testid='subdag-label'>
              {data.label}
            </span>
          </button>
          <button
            type='button'
            className={`nodrag kfp-graph-expand ${draggable ? 'nopan' : ''}`}
            aria-label={`${collapsed ? 'Expand' : 'Collapse'} ${data.label}`}
            aria-expanded={!collapsed}
            aria-description={data.expansionDeferred}
            title={data.expansionDeferred || `${collapsed ? 'Expand' : 'Collapse'} sub-DAG`}
            onClick={(event) => {
              event.stopPropagation();
              data.expand(id);
            }}
            style={{ background: 'transparent', border: 0, padding: 0, width: '2rem' }}
          >
            {collapsed ? <UnfoldMoreIcon size={16} /> : <UnfoldLessIcon size={16} />}
          </button>
          {data.state && (
            <div title={data.state} className='kfp-subdag-status' data-testid='subdag-status'>
              <TaskStateIndicator state={data.state} />
            </div>
          )}
        </div>
        {!collapsed && (data.expansionError || data.empty) && (
          <div
            style={{ padding: 12, fontSize: 12, color: 'var(--muted-foreground)' }}
            title={data.expansionError}
          >
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
