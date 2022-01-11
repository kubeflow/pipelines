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
// import ExpandLessIcon from '@material-ui/icons/ExpandLess';
// import ExpandMoreIcon from '@material-ui/icons/ExpandMore';

interface SubDagNodeProps {
  id: string;
  data: SubDagFlowElementData;
  // status: ExecutionNodeStatus;
  // tooltip: string;
  // isSelected: boolean;
}

function SubDagNode({ id, data }: SubDagNodeProps) {
  // TODO(zijianjoy): Implements interaction with expand and sidepanel
  const handleClick = (event: React.MouseEvent) => {
    event.stopPropagation();
    data.expand(id);
  };

  return (
    <>
      <button
        title={data.label}
        className='group focus:border-blue-500 rounded-xl border-gray-300 border-2 border-dashed'
      >
        <div className='container items-stretch h-24 w-72 relative grid '>
          <div className='flex justify-center place-self-center self-center relative h-14 w-72 '>
            <div className='transition transform hover:scale-105'>
              <div className=' flex justify-between flex-row h-14 relative overflow:hidden bg-white shadow-lg rounded-xl w-60 z-20'>
                <div className='px-6 py-4 w-60 flex flex-col justify-center items-center '>
                  <span className='w-full truncate' id={id}>
                    {data.label}
                  </span>
                </div>
              </div>
              <div className='flex absolute top-0 overflow:hidden bg-white shadow-lg rounded-xl h-14 w-60 ml-1 mt-1 z-10'></div>
            </div>
          </div>

          <div
            onClick={handleClick}
            className='transition transform hover:shadow-inner hover:scale-110 flex flex-col absolute rounded-full h-9 w-9 z-30 group-focus:border-blue-500 hover:border-blue-500  border-2 bg-white -right-5 top-8 items-center justify-center justify-items-center'
          >
            <div className='group-focus:text-blue-500 hover:text-blue-500 text-gray-300'>
              <CropFreeIcon style={{ fontSize: 15 }} />
            </div>
            {/* The following is alternative to the expand icon  */}
            {/* <ExpandLessIcon style={{ fontSize: 14, color: '#3B82F6', opacity: 1 }}></ExpandLessIcon>
            <ExpandMoreIcon style={{ fontSize: 14, color: '#3B82F6', opacity: 1 }}></ExpandMoreIcon> */}
          </div>
        </div>
      </button>
      <Handle
        type='target'
        position={Position.Top}
        isValidConnection={connection => connection.source === 'some-id'}
        onConnect={params => console.log('handle onConnect', params)}
        style={{ background: '#000', height: '1px', width: '1px', border: 0 }}
      />
      <Handle
        type='source'
        position={Position.Bottom}
        isValidConnection={connection => connection.source === 'some-id'}
        onConnect={params => console.log('handle onConnect', params)}
        style={{ background: '#000', height: '1px', width: '1px', border: 0 }}
      />
    </>
  );
}

export default SubDagNode;
