/*
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Tooltip } from '@base-ui/react/tooltip';
import { useTheme } from 'src/components/shell/ThemeProvider';
import {
  Box,
  CircleCheck,
  CircleX,
  LoaderCircle,
  SkipForward,
  Database,
  CircleHelp,
  TriangleAlert,
} from 'lucide-react';
import { PipelineTaskTaskState } from 'src/apisv2/run';
import { ExecutionFlowElementData, GRAPH_NODE_HEIGHT, GRAPH_NODE_WIDTH } from './Constants';
import { ReadOnlyNodeHandles } from './ReadOnlyNodeHandles';
import { getTaskStatus } from 'src/lib/StatusUtils';
import './Graph.css';

export interface ExecutionNodeProps {
  id: string;
  selected?: boolean;
  data: ExecutionFlowElementData;
}

export function TaskStateIndicator({
  state,
  warning = false,
}: {
  state?: PipelineTaskTaskState;
  warning?: boolean;
}) {
  if (warning) {
    return (
      <TriangleAlert
        className='kfp-graph-state-icon'
        data-state='WARNING'
        style={{ color: 'var(--status-warning)' }}
        size={16}
        aria-hidden='true'
      />
    );
  }
  if (state === undefined) return null;
  const Icon =
    {
      SUCCEEDED: CircleCheck,
      FAILED: CircleX,
      RUNNING: LoaderCircle,
      SKIPPED: SkipForward,
      CACHED: Database,
      RUNTIME_STATE_UNSPECIFIED: CircleHelp,
    }[state] || CircleHelp;
  return <Icon className='kfp-graph-state-icon' data-state={state} size={16} aria-hidden='true' />;
}

/** Focusable status help is shared by task cards and nested group headers. */
export function TaskStatusTooltip({
  state,
  warning = false,
  className,
  id,
  testId,
}: {
  state?: PipelineTaskTaskState;
  warning?: boolean;
  className: string;
  id?: string;
  testId?: string;
}) {
  const { themeClassName } = useTheme();
  const label = warning ? 'Pod lifecycle warning' : getTaskStatus(state).label;
  if (state === undefined && !warning) {
    return (
      <span
        className={className}
        id={id}
        data-testid={testId}
        data-empty='true'
        aria-hidden='true'
      />
    );
  }
  return (
    <Tooltip.Root>
      <Tooltip.Trigger
        render={<span />}
        tabIndex={0}
        role='img'
        aria-label={label}
        className={className}
        id={id}
        data-testid={testId}
        delay={150}
      >
        <TaskStateIndicator state={state} warning={warning} />
      </Tooltip.Trigger>
      <Tooltip.Portal className={themeClassName}>
        <Tooltip.Positioner side='top' sideOffset={6}>
          <Tooltip.Popup role='tooltip' className='kfp-graph-tooltip'>
            {label}
          </Tooltip.Popup>
        </Tooltip.Positioner>
      </Tooltip.Portal>
    </Tooltip.Root>
  );
}

export function hasLifecycleWarning(
  data: Pick<ExecutionFlowElementData, 'state' | 'lifecycleMessage'>,
) {
  // A pod lifecycle warning must never replace the failed task indicator.
  return !!data.lifecycleMessage && data.state !== PipelineTaskTaskState.FAILED;
}

export function getLifecycleWarningLabel(label: string) {
  return `${label}, pod lifecycle warning`;
}

export default function ExecutionNode({ id, data, selected }: ExecutionNodeProps) {
  const { themeClassName } = useTheme();
  const showWarning = hasLifecycleWarning(data);
  const status = getTaskStatus(data.state);
  const statusLabel = showWarning ? `${status.label}, pod lifecycle warning` : status.label;
  return (
    <>
      <Tooltip.Root>
        <Tooltip.Trigger
          render={<button type='button' />}
          delay={150}
          className='kfp-graph-node'
          data-selected={selected || undefined}
          style={{ width: GRAPH_NODE_WIDTH, height: GRAPH_NODE_HEIGHT }}
          data-tone={showWarning ? 'warning' : status.tone}
          title={data.state === undefined ? data.label : undefined}
          aria-label={showWarning ? getLifecycleWarningLabel(data.label) : data.label}
          aria-describedby={`${id}-status`}
        >
          <Box
            className='kfp-graph-type-icon'
            data-kind='component'
            size={16}
            role='img'
            aria-label='Task'
          />
          <span className='kfp-graph-node-copy'>
            <span className='kfp-graph-node-name' id={id}>
              {data.label}
            </span>
          </span>
          <TaskStatusTooltip
            className='kfp-graph-node-status'
            state={data.state}
            warning={showWarning}
          />
          <span className='sr-only' id={`${id}-status`}>
            {statusLabel}
          </span>
        </Tooltip.Trigger>
        <Tooltip.Portal className={themeClassName}>
          <Tooltip.Positioner side='top' sideOffset={6}>
            <Tooltip.Popup role='tooltip' className='kfp-graph-tooltip'>
              {data.label} · {statusLabel}
            </Tooltip.Popup>
          </Tooltip.Positioner>
        </Tooltip.Portal>
      </Tooltip.Root>
      <ReadOnlyNodeHandles />
    </>
  );
}
