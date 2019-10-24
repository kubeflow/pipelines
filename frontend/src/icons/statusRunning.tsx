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
import { CSSProperties } from 'jss/css';

export default class StatusRunning extends React.Component<{ style: CSSProperties }> {
  public render(): JSX.Element {
    const { style } = this.props;
    return (
      <svg width={style.width as string} height={style.height as string} viewBox='0 0 18 18'>
        <g transform='translate(-450, -307)' fill={style.color as string} fillRule='nonzero'>
          <g transform='translate(450, 266)'>
            <g transform='translate(0, 41)'>
              <path
                d='M9,4 C6.23857143,4 4,6.23857143 4,9 C4,11.7614286 6.23857143,14 9,14
                C11.7614286,14 14,11.7614286 14,9 C14,8.40214643 13.8950716,7.8288007
                13.702626,7.29737398 L15.2180703,5.78192967 C15.7177126,6.74539838
                16,7.83973264 16,9 C16,12.866 12.866,16 9,16 C5.134,16 2,12.866 2,9 C2,5.134
                5.134,2 9,2 C10.933,2 12.683,2.7835 13.94975,4.05025 L12.7677679,5.23223214
                L9,9 L9,4 Z'
              />
            </g>
          </g>
        </g>
      </svg>
    );
  }
}
