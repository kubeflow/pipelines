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
import MoreHorizIcon from '@mui/icons-material/MoreHoriz';
import PowerSettingsNewIcon from '@mui/icons-material/PowerSettingsNew';
import RefreshIcon from '@mui/icons-material/Refresh';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import { ReactElement } from 'react';
import { V2beta1RuntimeState } from 'src/apisv2beta1/run';
import StopCircle from 'src/icons/StopCircle';
import { Execution } from 'src/third_party/mlmd';
import { classes } from 'typestyle';
import { ExecutionFlowElementData, FlowElementState } from './Constants';
import { ReadOnlyNodeHandles } from './ReadOnlyNodeHandles';

export interface ExecutionNodeProps {
  id: string;
  data: ExecutionFlowElementData;
  selected?: boolean;
}

function ExecutionNode({ id, data, selected }: ExecutionNodeProps) {
  const icon = getIcon(data.state);
  const executionIcon = getExecutionIcon(data.state);

  const fullWidth = icon ? 'w-64' : 'w-56';

  return (
    <>
      <div title={data.label} className='container'>
        <button
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
              <span className='w-44 text-sm truncate' id={id} data-testid={id}>
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

function isMlmdState(state: FlowElementState | undefined): state is Execution.State {
  return typeof state === 'number';
}

export function getExecutionIcon(state: FlowElementState | undefined) {
  if (state === undefined) {
    return <ListAltIcon data-testid='execution-icon-default' className='text-mui-grey-500' />;
  }
  return <ListAltIcon data-testid='execution-icon-active' className='text-mui-blue-600' />;
}

export function getIcon(state: FlowElementState | undefined) {
  if (state === undefined) {
    return null;
  }
  if (isMlmdState(state)) {
    switch (state) {
      case Execution.State.UNKNOWN:
        return getStateIconWrapper(
          <MoreHorizIcon className='text-mui-grey-600' />,
          'bg-mui-grey-200',
        );
      case Execution.State.NEW:
        return getStateIconWrapper(
          <PowerSettingsNewIcon className='text-mui-blue-600' />,
          'bg-mui-blue-50',
        );
      case Execution.State.RUNNING:
        return getStateIconWrapper(
          <RefreshIcon className='text-mui-green-600' />,
          'bg-mui-green-50',
        );
      case Execution.State.CACHED:
        return getStateIconWrapper(
          <CloudDownloadIcon className='text-mui-green-600' />,
          'bg-mui-green-50',
        );
      case Execution.State.FAILED:
        return getStateIconWrapper(<ErrorIcon className='text-mui-red-600' />, 'bg-mui-red-50');
      case Execution.State.CANCELED:
        return getStateIconWrapper(
          <StopCircle colorClass={'text-mui-grey-600'} />,
          'bg-mui-grey-200',
        );
      case Execution.State.COMPLETE:
        return getStateIconWrapper(
          <CheckCircleIcon className='text-mui-green-600' />,
          'bg-mui-green-50',
        );
      default:
        console.error('Unknown execution state: ' + state);
        return getStateIconWrapper(<RemoveCircleOutlineIcon className='text-white' />, 'bg-black');
    }
  }
  switch (state) {
    case V2beta1RuntimeState.PENDING:
      return getStateIconWrapper(
        <PowerSettingsNewIcon className='text-mui-blue-600' />,
        'bg-mui-blue-50',
      );
    case V2beta1RuntimeState.RUNNING:
      return getStateIconWrapper(<RefreshIcon className='text-mui-green-600' />, 'bg-mui-green-50');
    case V2beta1RuntimeState.SUCCEEDED:
      return getStateIconWrapper(
        <CheckCircleIcon className='text-mui-green-600' />,
        'bg-mui-green-50',
      );
    case V2beta1RuntimeState.FAILED:
      return getStateIconWrapper(<ErrorIcon className='text-mui-red-600' />, 'bg-mui-red-50');
    case V2beta1RuntimeState.CANCELING:
    case V2beta1RuntimeState.CANCELED:
      return getStateIconWrapper(
        <StopCircle colorClass={'text-mui-grey-600'} />,
        'bg-mui-grey-200',
      );
    case V2beta1RuntimeState.SKIPPED:
      return getStateIconWrapper(
        <MoreHorizIcon className='text-mui-grey-600' />,
        'bg-mui-grey-200',
      );
    default:
      return getStateIconWrapper(<ListAltIcon className='text-mui-blue-600' />, 'bg-mui-blue-50');
  }
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
