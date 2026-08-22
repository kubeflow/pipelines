// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as React from 'react';
import { act, render, waitFor } from '@testing-library/react';
import NativeLineageCanvas, { LineageEdge } from './NativeLineageCanvas';

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

it('measures individual ports, updates on resize, drops removed edges and disconnects', async () => {
  let resized: () => void = () => {};
  const disconnect = vi.fn();
  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(callback: () => void) {
        resized = callback;
      }
      observe() {}
      unobserve() {}
      disconnect = disconnect;
    },
  );
  let targetY = 30;
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
    const id = this.getAttribute('data-lineage-node');
    return new DOMRect(
      id === 'a' ? 10 : id === 'b' ? 210 : 0,
      id === 'a' ? 20 : id === 'b' ? targetY : 0,
      100,
      40,
    );
  });
  const content = (showTarget: boolean) => (
    <NativeLineageCanvas>
      <div data-lineage-node='a'>Source</div>
      {showTarget && <div data-lineage-node='b'>Destination</div>}
      <LineageEdge id='ab' from='a' to='b' />
    </NativeLineageCanvas>
  );
  const view = render(content(true));
  const edge = () => view.container.querySelector('path[data-from="a"][data-to="b"]');
  await waitFor(() => expect(edge()).toHaveAttribute('d', 'M110,40 C160,40 160,50 210,50'));
  targetY = 90;
  act(() => resized());
  await waitFor(() => expect(edge()).toHaveAttribute('d', 'M110,40 C160,40 160,110 210,110'));
  view.rerender(content(false));
  await waitFor(() => expect(edge()).toBeNull());
  view.unmount();
  expect(disconnect).toHaveBeenCalled();
});
