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

import * as React from 'react';

interface StopCircleProps {
  colorClass: string;
}

export default function StopCircle({ colorClass }: StopCircleProps) {
  return (
    <div className={colorClass}>
      <svg
        xmlns='http://www.w3.org/2000/svg'
        enable-background='new 0 0 24 24'
        height='24px'
        viewBox='0 0 24 24'
        width='24px'
        fill='#455A64'
      >
        <rect fill='none' height='24' width='24' />
        <path d='M12,2C6.48,2,2,6.48,2,12c0,5.52,4.48,10,10,10s10-4.48,10-10C22,6.48,17.52,2,12,2z M12,20c-4.42,0-8-3.58-8-8s3.58-8,8-8 s8,3.58,8,8S16.42,20,12,20z M16,16H8V8h8V16z' />
      </svg>
    </div>
  );
}
