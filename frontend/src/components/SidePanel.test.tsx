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
import SidePanel from './SidePanel';

describe('SidePanel', () => {
  it('does not emit legacy componentWillReceiveProps warnings from Resizable', () => {
    const consoleErrorSpy = vi.spyOn(console, 'error');

    try {
      render(
        <SidePanel isOpen={true} onClose={vi.fn()} title='Node details'>
          side panel content
        </SidePanel>,
      );

      const legacyLifecycleWarnings = consoleErrorSpy.mock.calls.filter(([message]) =>
        typeof message === 'string'
          ? message.includes('componentWillReceiveProps') && message.includes('Resizable')
          : false,
      );

      expect(legacyLifecycleWarnings).toHaveLength(0);
    } finally {
      consoleErrorSpy.mockRestore();
    }
  });
});
