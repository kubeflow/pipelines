// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { ChevronsDownUp as UnfoldLessIcon, ChevronsUpDown as UnfoldMoreIcon } from 'lucide-react';
import { Tooltip } from '@base-ui/react/tooltip';
import { useTheme } from 'src/components/shell/ThemeProvider';
import { Node, NodeProps } from '@xyflow/react';
import { SubDagFlowElementData } from './Constants';
import { TaskStatusTooltip, hasLifecycleWarning, getLifecycleWarningLabel } from './ExecutionNode';
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
  const { themeClassName } = useTheme();
  const showWarning = hasLifecycleWarning(data);
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
          <div data-testid='subdag-type' className='kfp-subdag-type'>
            <SubDagTypeIcon kind={data.groupKind} />
          </div>
          <Tooltip.Root>
            <Tooltip.Trigger
              render={<button type='button' />}
              delay={150}
              className='kfp-subdag-label'
              aria-describedby={`${id}-status-description`}
              aria-label={showWarning ? getLifecycleWarningLabel(data.label) : data.label}
              style={{ background: 'transparent', border: 0, fontWeight: 400 }}
            >
              <span className='truncate' data-testid='subdag-label'>
                {data.label}
              </span>
              <span className='sr-only' id={`${id}-status-description`}>
                {data.state ? status.label : data.groupKind || 'Sub-DAG'}
                {showWarning && ', pod lifecycle warning'}
              </span>
            </Tooltip.Trigger>
            <Tooltip.Portal className={themeClassName}>
              <Tooltip.Positioner side='top' sideOffset={6}>
                <Tooltip.Popup role='tooltip' className='kfp-graph-tooltip'>
                  {data.label}
                  {data.state && ` · ${status.label}`}
                  {showWarning && ', pod lifecycle warning'}
                </Tooltip.Popup>
              </Tooltip.Positioner>
            </Tooltip.Portal>
          </Tooltip.Root>
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
            style={{ background: 'transparent', border: 0, padding: 0 }}
          >
            {collapsed ? <UnfoldMoreIcon size={16} /> : <UnfoldLessIcon size={16} />}
          </button>
          <TaskStatusTooltip
            state={data.state}
            warning={showWarning}
            id={`${id}-status`}
            className='kfp-subdag-status'
            testId='subdag-status'
          />
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
