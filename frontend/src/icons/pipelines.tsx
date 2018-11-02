/*
 * Copyright 2018 Google LLC
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

export default class PipelinesIcon extends React.Component<{color: string}> {
  public render() {
    return (
      <svg width='20' height='20' viewBox='0 0 20 18' xmlns='http://www.w3.org/2000/svg'>
        <g id='Symbols' fill='none' fillRule='evenodd'>
          <g id='collapsed-nav' transform='translate(-26 -25)'>
            <g id='Group-17' transform='translate(0 12)'>
              <g id='Group-4'>
                <g id='ic_schedule_black_24dp-copy-2' transform='translate(26 12)'>
                  <polygon id='Shape' points='0 0 20 0 20 20 0 20' />
                  <g id='Group' transform='translate(1.667 2.5)' stroke={this.props.color} strokeWidth='1.5'>
                    <path d='M10,7.5 L13.3333333,7.5' id='Line-Copy-2' strokeLinecap='square'
                    />
                    <circle id='Oval-2-Copy-6' cx='15' cy='7.5' r='1.667' />
                    <path d='M3.33333333,7.5 L6.66666667,7.5' id='Line-Copy-3' strokeLinecap='square'
                    />
                    <circle id='Oval-2-Copy-7' cx='8.333' cy='7.5' r='1.667' />
                    <path d='M10,13.3333333 L14.1666667,13.3333333' id='Line-Copy-5' strokeLinecap='round'
                    />
                    <path d='M2.5,13.3333333 L6.66666667,13.3333333' id='Line-Copy-4' strokeLinecap='round'
                    />
                    <circle id='Oval-2-Copy-9' cx='8.333' cy='13.333' r='1.667' />
                    <path d='M10,1.66666667 L14.1666667,1.66666667' id='Line-Copy-7' strokeLinecap='round'
                    />
                    <path d='M2.5,1.66666667 L6.66666667,1.66666667' id='Line-Copy-6' strokeLinecap='round'
                    />
                    <circle id='Oval-2-Copy-10' cx='8.333' cy='1.667' r='1.667' />
                    <circle id='Oval-2-Copy-8' cx='1.667' cy='7.5' r='1.667' />
                  </g>
                </g>
              </g>
            </g>
          </g>
        </g>
      </svg>
    );
  }
}
