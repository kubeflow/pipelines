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

import CropFreeIcon from '@material-ui/icons/CropFree';
import React from 'react';
import { Handle, Position } from 'react-flow-renderer';
import { SubDagFlowElementData } from './Constants';

interface SubDagNodeProps {
  id: string;
  data: SubDagFlowElementData;
  // status: ExecutionNodeStatus;
  // tooltip: string;
  // isSelected: boolean;
}

function SubDagNode({ id, data }: SubDagNodeProps) {
  const handleClick = (event: React.MouseEvent) => {
    event.stopPropagation();
    data.expand(id);
  };

  return (
    <>
      <button
        title={data.label}
        className='focus:border-blue-500 rounded-xl border-gray-300 border-2 border-dashed'
      >
        <div className='container items-stretch h-24 w-80 relative grid '>
          <div className='flex justify-items-center place-self-center self-center relative h-14 w-72 '>
            <div className=' flex justify-between flex-row transform h-14 transition relative overflow:hidden hover:scale-105 bg-white shadow-lg rounded-xl  w-72 z-10'>
              <button onClick={handleClick}>
                <div className='sm:px-4 sm:py-4 rounded-l-xl justify-items-center hover:shadow-inner '>
                  <CropFreeIcon style={{ color: '#63B3ED' }} />
                </div>
              </button>
              <div className='sm:px-6 sm:py-4 w-60 flex flex-col justify-center items-center '>
                <span className='w-full truncate' id={id}>
                  {data.label}
                </span>
              </div>
            </div>
          </div>
        </div>
      </button>
      <Handle
        type='target'
        position={Position.Top}
        isValidConnection={connection => connection.source === 'some-id'}
        onConnect={params => console.log('handle onConnect', params)}
        style={{ background: '#000', width: '4px', height: '6px' }}
      />
      <Handle
        type='source'
        position={Position.Bottom}
        isValidConnection={connection => connection.source === 'some-id'}
        onConnect={params => console.log('handle onConnect', params)}
        style={{ background: '#000', width: '4px', height: '6px' }}
      />
    </>
  );
}

export default SubDagNode;
