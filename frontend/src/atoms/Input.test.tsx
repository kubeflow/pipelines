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
import { vi } from 'vitest';
import Input from './Input';

describe('Input', () => {
  const handleChange = vi.fn();
  const value = 'some input value';

  it('renders with the right styles by default', () => {
    const { asFragment } = render(
      <Input onChange={handleChange('fieldname')} value={value} variant='outlined' />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('accepts height and width as prop overrides', () => {
    const { asFragment } = render(
      <Input
        height={123}
        width={456}
        onChange={handleChange('fieldname')}
        value={value}
        variant='outlined'
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });
});
