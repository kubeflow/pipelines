// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import UnfoldLessIcon from '@mui/icons-material/UnfoldLess';
import UnfoldMoreIcon from '@mui/icons-material/UnfoldMore';
import { Node, NodeProps } from '@xyflow/react';
import { classes, style } from 'typestyle';
import { SubDagFlowElementData } from './Constants';
import { getIcon } from './ExecutionNode';
import { ReadOnlyNodeHandles } from './ReadOnlyNodeHandles';
import SubDagTypeIcon from './SubDagTypeIcon';
import { GROUP_HEADER_HEIGHT } from 'src/lib/v2/GroupedFlow';

// The group outline clips the outer corners; do not expose the white header
// behind the status cell's own rounded lower-right corner.
const groupStatus = style({
  $nest: { '& > div': { borderTopRightRadius: 0, borderBottomRightRadius: 0 } },
});

export default function SubDagGroupNode({
  id,
  data,
  selected,
}: Pick<NodeProps<Node<SubDagFlowElementData>>, 'id' | 'data' | 'selected'>) {
  const collapsed = !!data.collapsed;
  const status = getIcon(data.state);
  return (
    <>
      <div
        data-testid='subdag-box'
        className={classes(
          'shadow-lg',
          style({
            width: '100%',
            height: '100%',
            outline: selected ? '2px solid #1a73e8' : '1px solid #bdc1c6',
            borderRadius: 8,
            background: 'rgba(219, 234, 254, 0.28)',
            overflow: 'hidden',
            $nest: { '&:focus-within': { outline: '2px solid #1a73e8' } },
          }),
        )}
      >
        <div
          data-testid='subdag-header'
          className='flex items-stretch bg-white'
          style={{ height: data.headerHeight ?? GROUP_HEADER_HEIGHT }}
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
            className='nodrag nopan focus:outline-none focus:ring-0 text-mui-grey-600 flex items-center justify-center flex-shrink-0'
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
            {collapsed ? <UnfoldMoreIcon fontSize='small' /> : <UnfoldLessIcon fontSize='small' />}
          </button>
          {status && (
            <div
              title={data.state}
              className={classes('h-full', groupStatus)}
              data-testid='subdag-status'
            >
              {status}
            </div>
          )}
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
