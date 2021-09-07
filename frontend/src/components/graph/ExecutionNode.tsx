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
import SyncDisabledIcon from '@material-ui/icons/SyncDisabled';
import React from 'react';
import { Handle, Position } from 'react-flow-renderer';
import { Execution } from 'src/third_party/mlmd';
import { ExecutionFlowElementData } from './Constants';
// TODO(zijianjoy): Introduce execution node status separately.
// enum ExecutionNodeStatus {
//   UNSPECIFIED = 'UNSPECIFIED',
//   PENDING = 'PENDING',
//   RUNNING = 'RUNNING',
//   SUCCEEDED = 'SUCCEEDED',
//   CANCELLING = 'CANCELLING',
//   CANCELLED = 'CANCELLED',
//   FAILED = 'FAILED',
//   SKIPPED = 'SKIPPED',
//   NOT_READY = 'NOT_READY', // TBD: Represent QUEUED, NOT_TRIGGERED and UNSCHEDULABLE
// }
// enum State {
//   UNKNOWN = 0;
//   NEW = 1;
//   RUNNING = 2;
//   COMPLETE = 3;
//   FAILED = 4;
//   CACHED = 5;
//   CANCELED = 6;
// }

export interface ExecutionNodeProps {
  id: string;
  data: ExecutionFlowElementData;
  // selected: boolean;
  // status: ExecutionNodeStatus;
}

function ExecutionNode({ id, data }: ExecutionNodeProps) {
  let icon = getIcon(data.state);

  return (
    <>
      <div title={data.label} className='container w-60'>
        <button className='focus:ring flex items-stretch border-0 transform h-12 hover:scale-105 transition relative overflow:hidden bg-white shadow-lg rounded-lg w-60'>
          <div className='flex justify-between flex-row relative w-full h-full'>
            <div className='sm:px-3 py-4 w-48 h-full flex justify-center items-center'>
              <span className='w-full text-sm truncate' id={id}>
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
    // state is undefined or UNKNOWN.
    return (
      <div className='px-2 h-full flex flex-col justify-center rounded-r-lg bg-mui-grey-200'>
        <ListAltIcon className='text-mui-grey-500' />
      </div>
    );
  }
  switch (state) {
    case Execution.State.UNKNOWN:
      return (
        <div className='px-2 h-full flex flex-col justify-center rounded-r-lg bg-mui-grey-300'>
          <RemoveCircleOutlineIcon className='text-black' />
        </div>
      );
    case Execution.State.NEW:
      return (
        <div className='px-2 h-full flex flex-col justify-center rounded-r-lg bg-mui-lightblue-100 '>
          <PowerSettingsNewIcon className='text-mui-blue-500' />
        </div>
      );
    case Execution.State.RUNNING:
      return (
        <div className='px-2 h-full flex flex-col justify-center rounded-r-lg bg-mui-green-200-light'>
          <RefreshIcon className='text-mui-green-500-dark' />
        </div>
      );
    case Execution.State.CACHED:
      return (
        <div className='px-2 h-full flex flex-col justify-center rounded-r-lg bg-mui-green-200-light'>
          <CloudDownloadIcon className='text-mui-green-500-dark' />
        </div>
      );
    case Execution.State.FAILED:
      return (
        <div className='px-2 h-full flex flex-col justify-center rounded-r-lg bg-mui-red-100'>
          <ErrorIcon className='text-mui-red-500' />
        </div>
      );
    case Execution.State.CANCELED:
      return (
        <div className='px-2 h-full flex flex-col justify-center rounded-r-lg bg-mui-yellow-100'>
          <SyncDisabledIcon className=' text-mui-organge-300' />
        </div>
      );
    case Execution.State.COMPLETE:
      return (
        <div className='px-2 h-full flex flex-col justify-center rounded-r-lg bg-mui-green-200-light'>
          <CheckCircleIcon className='text-mui-green-500-dark' />
        </div>
      );
    default:
      throw new Error('Unknown exeuction state: ' + state);
  }
}
