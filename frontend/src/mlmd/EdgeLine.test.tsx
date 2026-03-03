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
import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { EdgeLine } from './EdgeLine';

describe('EdgeLine', () => {
  it('renders a curved connector between edge ports', () => {
    const { container } = render(<EdgeLine height={68} width={80} y1={0} y4={67} />);
    const path = container.querySelector('path');
    expect(path).not.toBeNull();
    expect(path!.getAttribute('d')).toBe('M0,68 C30,68 50,1 80,1');
  });

  it('keeps horizontal edges horizontal', () => {
    const { container } = render(<EdgeLine height={1} width={80} y1={0} y4={0} />);
    const path = container.querySelector('path');
    expect(path).not.toBeNull();
    expect(path!.getAttribute('d')).toBe('M0,1 C30,1 50,1 80,1');
  });
});
