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
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import { ReactElement } from 'react';
import { PipelineTaskTaskState } from 'src/apisv2/run';
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
  // A lifecycle message is a warning, not a failure: the node only turns red once the
  // task itself is FAILED, so the failed icon takes precedence over the warning.
  const showWarning = !!data.lifecycleMessage && data.state !== PipelineTaskTaskState.FAILED;
  // A debug-pause barrier is a separate overlay on top of the task's real
  // state (which continues to report RUNNING while parked) - it takes
  // visual precedence here rather than being merged into getIcon's switch,
  // so a paused task is unmistakably distinct from a normally-running one.
  let icon = data.debugPauseBarrier
    ? getDebugPauseIcon()
    : showWarning
      ? getWarningIcon()
      : getIcon(data.state);
  let executionIcon = getExecutionIcon(data.state);

  const fullWidth = icon ? 'w-64' : 'w-56';

  return (
    <>
      <div title={data.label} className='container'>
        <button
          aria-label={showWarning ? getLifecycleWarningLabel(data.label) : undefined}
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
              <span className='w-44 text-sm truncate' id={id}>
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

// The amber icon is hidden from screen readers, so the warning goes into the name of the
// focusable node instead.
export function getLifecycleWarningLabel(label: string) {
  return `${label}, pod lifecycle warning`;
}

export function getWarningIcon() {
  return getStateIconWrapper(
    <WarningAmberIcon className='text-mui-yellow-800' />,
    'bg-mui-yellow-100',
  );
}

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

// Deliberately grey, distinct from any color used by getIcon's real state
// icons - a paused task is neither succeeding, failing, nor merely running;
// it is waiting for a person to resume it.
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
