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
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CloudDownloadIcon from '@mui/icons-material/CloudDownload';
import ErrorIcon from '@mui/icons-material/Error';
import ListAltIcon from '@mui/icons-material/ListAlt';
import PauseCircleIcon from '@mui/icons-material/PauseCircle';
import RefreshIcon from '@mui/icons-material/Refresh';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import React, { ReactElement } from 'react';
import { PipelineTaskTaskState } from 'src/apisv2/run';
import { CardTooltip } from 'src/atoms/CardTooltip';
import StopCircle from 'src/icons/StopCircle';
import { classes } from 'typestyle';
import { ExecutionFlowElementData } from './Constants';
import MoreHorizIcon from '@mui/icons-material/MoreHoriz';
import { ReadOnlyNodeHandles } from './ReadOnlyNodeHandles';

export interface ExecutionNodeProps {
  id: string;
  data: ExecutionFlowElementData;
  selected?: boolean;
  // status: ExecutionNodeStatus;
}

function ExecutionNode({ id, data, selected }: ExecutionNodeProps) {
  // A lifecycle message is a status, not a failure: the node only turns red once the task
  // itself is FAILED, so the failed icon takes precedence over the lifecycle status.
  //
  // The pause icon takes precedence over the lifecycle status icon, so while a task is parked there
  // is no lifecycle tooltip to describe: the message only counts when its icon is rendered.
  const lifecycleMessage = data.debugPauseBarrier ? undefined : data.lifecycleMessage;
  const hasLifecycleStatus = !!lifecycleMessage && data.state !== PipelineTaskTaskState.FAILED;
  const lifecycleTooltip = useLifecycleTooltip(!!lifecycleMessage);
  // Pause barrier is an overlay on the task's real state (still RUNNING
  // while parked) - render it instead of the state icon.
  let icon = data.debugPauseBarrier
    ? getDebugPauseIcon()
    : lifecycleMessage
      ? getLifecycleStatusIcon(
          lifecycleMessage,
          data.lifecycleCategory,
          data.state,
          lifecycleTooltip,
        )
      : getIcon(data.state);
  let executionIcon = getExecutionIcon(data.state);

  const fullWidth = icon ? 'w-64' : 'w-56';

  return (
    <>
      <div className='container'>
        <button
          aria-label={
            hasLifecycleStatus
              ? getLifecycleStatusLabel(data.label, data.lifecycleCategory)
              : undefined
          }
          {...lifecycleTooltip.focusProps}
          className={classes(
            'focus:ring flex items-stretch border-0 transform h-12 hover:scale-105 transition relative overflow:hidden bg-white shadow-lg rounded-lg',
            fullWidth,
            selected && 'ring',
          )}
        >
          <div className='flex justify-between flex-row relative w-full h-full'>
            <div className='w-8 pl-2 h-full flex flex-col justify-center rounded-l-lg'>
              {executionIcon}
            </div>
            <div className='px-3 py-4 w-44 h-full flex justify-center items-center'>
              <span className='w-44 text-sm truncate' id={id} title={data.label}>
                {data.label}
              </span>
            </div>
            {icon}
          </div>
        </button>
      </div>
      <ReadOnlyNodeHandles />
    </>
  );
}
export default ExecutionNode;

export function getExecutionIcon(state: PipelineTaskTaskState | undefined) {
  if (state === undefined) {
    return <ListAltIcon data-testid='execution-icon-default' className='text-mui-grey-500' />;
  }
  return <ListAltIcon data-testid='execution-icon-active' className='text-mui-blue-600' />;
}

export function getIcon(state: PipelineTaskTaskState | undefined) {
  if (state === undefined) {
    return null;
  }
  switch (state) {
    case PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED:
      return getStateIconWrapper(
        <MoreHorizIcon className='text-mui-grey-600' />,
        'bg-mui-grey-200',
      );

    case PipelineTaskTaskState.RUNNING:
      return getStateIconWrapper(<RefreshIcon className='text-mui-green-600' />, 'bg-mui-green-50');
    case PipelineTaskTaskState.CACHED:
      return getStateIconWrapper(
        <CloudDownloadIcon className='text-mui-green-600' />,
        'bg-mui-green-50',
      );
    case PipelineTaskTaskState.FAILED:
      return getStateIconWrapper(<ErrorIcon className='text-mui-red-600' />, 'bg-mui-red-50');
    case PipelineTaskTaskState.SKIPPED:
      return getStateIconWrapper(
        <StopCircle colorClass={'text-mui-grey-600'} />,
        'bg-mui-grey-200',
      );
    case PipelineTaskTaskState.SUCCEEDED:
      return getStateIconWrapper(
        <CheckCircleIcon className='text-mui-green-600' />,
        'bg-mui-green-50',
      );
    default:
      console.error('Unknown exeuction state: ' + state);
      return getStateIconWrapper(<RemoveCircleOutlineIcon className='text-white' />, 'bg-black');
  }
}

// The status icon is hidden from screen readers, so the lifecycle status goes into the name
// of the focusable node instead.
export function getLifecycleStatusLabel(label: string, category?: string) {
  return `${label}, pod lifecycle ${isUnclassifiedLifecycle(category) ? 'status' : 'warning'}`;
}

// An unclassified message is a raw status, not a known problem, so it is shown in blue.
export function isUnclassifiedLifecycle(category?: string) {
  return category === 'unknown';
}

const LIFECYCLE_CATEGORY_LABELS: Record<string, string> = {
  'image-pull': 'Image pull problem',
  scheduling: 'Scheduling problem',
  runtime: 'Runtime problem',
  admission: 'Admission problem',
  unknown: 'Pod status',
};

export function getLifecycleCategoryLabel(category?: string) {
  return (category && LIFECYCLE_CATEGORY_LABELS[category]) || 'Pod lifecycle warning';
}

export interface LifecycleTooltipState {
  open: boolean;
  // Stable id for the tooltip content, referenced by aria-describedby on the focusable
  // control so screen readers announce the category + Kubernetes message, not just the label.
  tooltipId: string;
  onOpen: () => void;
  onClose: () => void;
  // Spread onto the node's focusable control so the tooltip also opens on keyboard focus
  // and the control exposes aria-describedby pointing at the tooltip content. Escape is an
  // explicit dismissal; mouse leave alone never closes the tooltip while focus is on.
  focusProps: {
    onFocus: () => void;
    onBlur: () => void;
    onKeyDown: (event: React.KeyboardEvent) => void;
    'aria-describedby'?: string;
  };
}

// The tooltip hangs off the status icon, which is not focusable, so keyboard focus on the
// node's button is tracked here and opens the same tooltip. Escape closes it again.
//
// enabled says whether the node renders a lifecycle tooltip at all. Without one, focusing the
// button must not open anything or point aria-describedby at an id that is not in the DOM, so the
// caller passes the same condition it uses to render the status icon.
export function useLifecycleTooltip(enabled = true): LifecycleTooltipState {
  const tooltipId = React.useId();
  const [hovered, setHovered] = React.useState(false);
  const [focused, setFocused] = React.useState(false);
  const open = enabled && (hovered || focused);
  return {
    open,
    tooltipId,
    onOpen: () => setHovered(true),
    // MUI calls onClose on mouse leave; only clear hovered here so the tooltip stays
    // open while the button still has keyboard focus. Focus is cleared on blur below.
    onClose: () => setHovered(false),
    focusProps: {
      onFocus: () => setFocused(true),
      onBlur: () => setFocused(false),
      // Escape is the explicit dismissal: clear both so a hovered+focused tooltip closes
      // without waiting for the pointer to leave or focus to move.
      onKeyDown: (event: React.KeyboardEvent) => {
        if (event.key === 'Escape') {
          setFocused(false);
          setHovered(false);
        }
      },
      'aria-describedby': open ? tooltipId : undefined,
    },
  };
}

// Status icon for a task with a lifecycle message. The tooltip shows the category and the
// message, clamped so a long message cannot fill the screen. Pass the state from
// useLifecycleTooltip to open it from keyboard focus as well as hover.
export function getLifecycleStatusIcon(
  message: string,
  category?: string,
  state?: PipelineTaskTaskState,
  tooltip?: LifecycleTooltipState,
) {
  const failed = state === PipelineTaskTaskState.FAILED;
  const unclassified = !failed && isUnclassifiedLifecycle(category);
  const icon = failed ? (
    <ErrorIcon className='text-mui-red-600' />
  ) : unclassified ? (
    <InfoOutlinedIcon className='text-mui-blue-600' />
  ) : (
    <WarningAmberIcon className='text-mui-yellow-800' />
  );
  const background = failed
    ? 'bg-mui-red-50'
    : unclassified
      ? 'bg-mui-blue-50'
      : 'bg-mui-yellow-100';
  return (
    <CardTooltip
      open={tooltip?.open}
      onOpen={tooltip?.onOpen}
      onClose={tooltip?.onClose}
      helpText={
        <div data-testid='lifecycle-tooltip' id={tooltip?.tooltipId}>
          <strong>{getLifecycleCategoryLabel(category)}</strong>
          <div style={LIFECYCLE_MESSAGE_STYLE}>{message}</div>
          <small>Select the node, then Details, to see the full message.</small>
        </div>
      }
    >
      <div
        className={classes('px-2 h-full flex flex-col justify-center rounded-r-lg ', background)}
      >
        {icon}
      </div>
    </CardTooltip>
  );
}

const LIFECYCLE_MESSAGE_STYLE: React.CSSProperties = {
  display: '-webkit-box',
  WebkitLineClamp: 6,
  WebkitBoxOrient: 'vertical',
  overflow: 'hidden',
  overflowWrap: 'anywhere',
  whiteSpace: 'pre-wrap',
};

function getStateIconWrapper(element: ReactElement, backgroundClasses: string) {
  return (
    <div
      className={classes(
        'px-2 h-full flex flex-col justify-center rounded-r-lg ',
        backgroundClasses,
      )}
    >
      {element}
    </div>
  );
}

// Grey, distinct from the real state icons - a paused task waits on a person.
export function getDebugPauseIcon() {
  return getStateIconWrapper(
    <PauseCircleIcon data-testid='execution-icon-debug-paused' className='text-mui-grey-600' />,
    'bg-mui-grey-200',
  );
}

// The following code can be used for `canceling state`
// return (
//   <div className='px-2 h-full self-stretch flex flex-col justify-center rounded-r-lg bg-mui-lightblue-100'>
//     <CircularProgress size={24} />
//     {/* <SyncDisabledIcon className=' text-mui-organge-300' /> */}
//   </div>
// );
