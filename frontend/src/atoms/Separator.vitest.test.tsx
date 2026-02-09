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
import { render } from '@testing-library/react';
import Separator from './Separator';

describe('Separator', () => {
  it('renders with the right styles by default', () => {
    const { asFragment } = render(<Separator />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders with the specified orientation', () => {
    const { asFragment } = render(<Separator orientation='vertical' />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders with the specified units', () => {
    const { asFragment } = render(<Separator units={123} />);
    expect(asFragment()).toMatchSnapshot();
  });
});
