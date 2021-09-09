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
import CheckCircleIcon from '@material-ui/icons/CheckCircle';
import CloudDownloadIcon from '@material-ui/icons/CloudDownload';
import ErrorIcon from '@material-ui/icons/Error';
import ListAltIcon from '@material-ui/icons/ListAlt';
import PowerSettingsNewIcon from '@material-ui/icons/PowerSettingsNew';
import RefreshIcon from '@material-ui/icons/Refresh';
import RemoveCircleOutlineIcon from '@material-ui/icons/RemoveCircleOutline';
import React, { ReactElement } from 'react';
import { Handle, Position } from 'react-flow-renderer';
import StopCircle from 'src/icons/StopCircle';
import { Execution } from 'src/third_party/mlmd';
import { classes } from 'typestyle';
import { ExecutionFlowElementData } from './Constants';

export interface ExecutionNodeProps {
  id: string;
  data: ExecutionFlowElementData;
  // selected: boolean;
  // status: ExecutionNodeStatus;
}

function ExecutionNode({ id, data }: ExecutionNodeProps) {
  let icon = getIcon(data.state);

  const fullWidth = icon ? 'w-64' : 'w-56';

  return (
    <>
      <div title={data.label} className='container'>
        <button
          className={classes(
            'focus:ring flex items-stretch border-0 transform h-12 hover:scale-105 transition relative overflow:hidden bg-white shadow-lg rounded-lg',
            fullWidth,
          )}
        >
          <div className='flex justify-between flex-row relative w-full h-full'>
            <div className='w-8 pl-2 h-full flex flex-col justify-center rounded-l-lg'>
              <ListAltIcon className='text-mui-grey-500' />
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
      <Handle
        type='target'
        position={Position.Top}
        isValidConnection={() => false}
        style={{ background: '#000', width: '4px', height: '4px' }}
      />
      <Handle
        type='source'
        position={Position.Bottom}
        isValidConnection={() => false}
        style={{ background: '#000', width: '4px', height: '4px' }}
      />
    </>
  );
}
export default ExecutionNode;

function getIcon(state: Execution.State | undefined) {
  if (state === undefined) {
    return null;
  }
  switch (state) {
    case Execution.State.UNKNOWN:
      return getStateIconWrapper(
        <RemoveCircleOutlineIcon className='text-mui-grey-600' />,
        'bg-mui-grey-200',
      );
    case Execution.State.NEW:
      return getStateIconWrapper(
        <PowerSettingsNewIcon className='text-mui-blue-600' />,
        'bg-mui-blue-50',
      );
    case Execution.State.RUNNING:
      return getStateIconWrapper(<RefreshIcon className='text-mui-green-600' />, 'bg-mui-green-50');
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
      throw new Error('Unknown exeuction state: ' + state);
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

// The following code can be used for `canceling state`
// return (
//   <div className='px-2 h-full self-stretch flex flex-col justify-center rounded-r-lg bg-mui-lightblue-100'>
//     <CircularProgress size={24} />
//     {/* <SyncDisabledIcon className=' text-mui-organge-300' /> */}
//   </div>
// );
