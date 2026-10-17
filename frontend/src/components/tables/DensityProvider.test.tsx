/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { ThemeProvider } from '../shell/ThemeProvider';
import { LocalStorage, LocalStorageKey } from '../../lib/LocalStorage';
import { DensityMenu } from './DensityMenu';
import { useDensity } from './DensityProvider';

function Probe() {
  const preference = useDensity();
  return <output aria-label='Current density'>{preference?.density}</output>;
}
function App({ page = 'Pipelines' }: { page?: string }) {
  return (
    <ThemeProvider>
      <div key={page}>
        <h1>{page}</h1>
        <DensityMenu />
        <Probe />
      </div>
    </ThemeProvider>
  );
}
async function choose(value: string) {
  const user = userEvent.setup();
  await user.tab();
  await user.keyboard('{Enter}');
  await user.click(await screen.findByRole('menuitemradio', { name: value }));
}
beforeEach(() => {
  localStorage.clear();
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }));
});
function emitStorage(key: string | null) {
  const event = new Event('storage');
  Object.defineProperties(event, { key: { value: key }, storageArea: { value: localStorage } });
  window.dispatchEvent(event);
}
afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  localStorage.clear();
});

it('defaults to comfortable and keeps density independent of the remembered card view', async () => {
  LocalStorage.savePipelineView('cards');
  const app = render(<App />);
  expect(screen.getByLabelText('Current density')).toHaveTextContent('comfortable');
  await choose('Compact');
  expect(LocalStorage.getPipelineView()).toBe('cards');
  expect(LocalStorage.getCollectionDensity()).toBe('compact');
  app.rerender(<App page='Runs' />);
  expect(screen.getByLabelText('Current density')).toHaveTextContent('compact');
  app.unmount();
  render(<App />);
  expect(screen.getByLabelText('Current density')).toHaveTextContent('compact');
  await choose('Comfortable');
  expect(LocalStorage.getCollectionDensity()).toBe('comfortable');
  expect(LocalStorage.getPipelineView()).toBe('cards');
});

it('keeps the control usable when storage reads and writes are blocked', async () => {
  vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
    throw new Error('Blocked');
  });
  vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
    throw new Error('Blocked');
  });
  render(<App />);
  await choose('Compact');
  expect(screen.getByLabelText('Current density')).toHaveTextContent('compact');
});

it('normalizes invalid preferences and receives changes from another tab', () => {
  localStorage.setItem(LocalStorageKey.collectionDensity, 'invalid');
  render(<App />);
  expect(screen.getByLabelText('Current density')).toHaveTextContent('comfortable');
  act(() => {
    localStorage.setItem(LocalStorageKey.collectionDensity, 'compact');
    emitStorage(LocalStorageKey.collectionDensity);
  });
  expect(screen.getByLabelText('Current density')).toHaveTextContent('compact');
  act(() => {
    localStorage.clear();
    emitStorage(null);
  });
  expect(screen.getByLabelText('Current density')).toHaveTextContent('comfortable');
});
