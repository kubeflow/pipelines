/*
 * Copyright 2018 The Kubeflow Authors
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

export default class PipelinesIcon extends React.Component<{ color: string }> {
  public render(): JSX.Element {
    return (
      <svg
        width='20px'
        height='20px'
        viewBox='0 0 20 20'
        version='1.1'
        xmlns='http://www.w3.org/2000/svg'
        xmlnsXlink='http://www.w3.org/1999/xlink'
      >
        <g id='Symbols' stroke='none' strokeWidth='1' fill='none' fillRule='evenodd'>
          <g transform='translate(-2.000000, -4.000000)'>
            <polygon id='Shape' points='0 0 24 0 24 24 0 24' />
            <path
              d='M12.7244079,9.74960425 L17.4807112,9.74960425 C17.7675226,9.74960425
                      18,9.51894323 18,9.23437272 L18,4.51523153 C18,4.23066102 17.7675226,4
                      17.4807112,4 L12.7244079,4 C12.4375965,4 12.2051191,4.23066102
                      12.2051191,4.51523153 L12.2051191,6.06125154 L9.98218019,6.06125154
                      C9.52936032,6.06125154 9.16225043,6.42549311 9.16225043,6.87477501
                      L9.16225043,11.2135669 L7.05995053,11.2135669 C6.71661861,10.189612
                      5.74374462,9.45093267 4.59644424,9.45093267 C3.16249641,9.45093267
                      2,10.6043462 2,12.0270903 C2,13.4498886 3.16249641,14.603248
                      4.59644424,14.603248 C5.74379928,14.603248 6.71661861,13.8645687
                      7.06000519,12.8406138 L9.16225043,12.8406138 L9.16225043,17.1794057
                      C9.16225043,17.6286875 9.52936032,17.9929291 9.98218019,17.9929291
                      L12.2051191,17.9929291 L12.2051191,19.4847685 C12.2051191,19.769339
                      12.4375965,20 12.7244079,20 L17.4807112,20 C17.7675226,20 18,19.769339
                      18,19.4847685 L18,14.7656273 C18,14.4810568 17.7675226,14.2503957
                      17.4807112,14.2503957 L12.7244079,14.2503957 C12.4375965,14.2503957
                      12.2051191,14.4810568 12.2051191,14.7656273 L12.2051191,16.3658822
                      L10.80211,16.3658822 L10.80211,7.68829848 L12.2051191,7.68829848
                      L12.2051191,9.23437272 C12.2051191,9.51894323 12.4375965,9.74960425
                      12.7244079,9.74960425 Z'
              id='Path'
              fill={this.props.color}
            />
          </g>
        </g>
      </svg>
    );
  }
}
