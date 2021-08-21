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
import React from 'react';
import { Handle, Position } from 'react-flow-renderer';

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

export interface ExecutionNodeData {
  label: string;
}

export interface ExecutionNodeProps {
  id: string;
  data: ExecutionNodeData;
  // selected: boolean;
  // status: ExecutionNodeStatus;
}

function ExecutionNode({ id, data }: ExecutionNodeProps) {
  return (
    <>
      <div title={data.label} className='container w-60'>
        <button className='focus:ring flex items-stretch border-0 transform h-12 hover:scale-105 transition relative overflow:hidden bg-white shadow-lg rounded-lg w-60'>
          <div className='flex justify-between flex-row relative w-full h-full'>
            <div className='sm:px-3 py-4 w-48 h-full flex justify-center items-center'>
              <span className='w-full text-sm truncate ...' id={id}>
                {data.label}
              </span>
            </div>
            <div className=' sm:px-2 h-full flex flex-col justify-center rounded-r-lg bg-mui-green-200-light'>
              <CheckCircleIcon className='text-mui-green-500-strong' />
            </div>
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
