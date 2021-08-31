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

import FolderIcon from '@material-ui/icons/Folder';
import React from 'react';
import { Handle, Position } from 'react-flow-renderer';
import { FlowElementDataBase } from './Constants';

interface ArtifactNodeProps {
  id: string;
  data: FlowElementDataBase;
  // selected: boolean;
  // status: ExecutionNodeStatus;
  // tooltip: string;
}

function ArtifactNode({ id, data }: ArtifactNodeProps) {
  return (
    <>
      <button
        title={data.label}
        className='focus:ring flex items-stretch hover:scale-105 transition transform border-0 shadow-lg rounded-lg w-60 h-12'
      >
        <div className='flex items-center justify-between w-60 rounded-lg shadow-lg bg-white'>
          <div className='px-2 flex flex-col justify-center items-center rounded-l-lg'>
            <FolderIcon className='text-mui-yellow-600' />
          </div>
          <div className='flex flex-grow justify-center items-center rounded-r-lg overflow-hidden'>
            <span className='text-sm truncate' id={id}>
              {data.label}
            </span>
          </div>
        </div>
      </button>
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

export default ArtifactNode;
