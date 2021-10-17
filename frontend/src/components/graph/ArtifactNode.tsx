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
import { Artifact } from 'src/third_party/mlmd';
import { ArtifactFlowElementData } from './Constants';

interface ArtifactNodeProps {
  id: string;
  data: ArtifactFlowElementData;
  // selected: boolean;
  // status: ExecutionNodeStatus;
  // tooltip: string;
}

function ArtifactNode({ id, data }: ArtifactNodeProps) {
  let icon = getIcon(data.state);
  return (
    <>
      <button
        title={data.label}
        className='focus:ring flex items-stretch hover:scale-105 transition transform border-0 shadow-lg rounded-lg w-60 h-12'
      >
        <div className='flex items-center justify-between w-60 rounded-lg shadow-lg bg-white'>
          {icon}
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
        style={{ background: '#000', height: '1px', width: '1px', border: 0 }}
      />
      <Handle
        type='source'
        position={Position.Bottom}
        isValidConnection={() => false}
        style={{ background: '#000', height: '1px', width: '1px', border: 0 }}
      />
    </>
  );
}

export default ArtifactNode;

function getIcon(state: Artifact.State | undefined) {
  if (state === undefined) {
    return getIconWrapper(<FolderIcon className='text-mui-grey-300-dark' />);
  }
  switch (state) {
    case Artifact.State.LIVE:
      return getIconWrapper(<FolderIcon className='text-mui-yellow-800' />);
    default:
      return getIconWrapper(<FolderIcon className='text-mui-grey-300-dark' />);
  }
}

function getIconWrapper(element: React.ReactElement) {
  return (
    <div className='px-2 flex flex-col justify-center items-center rounded-l-lg'>{element}</div>
  );
}
