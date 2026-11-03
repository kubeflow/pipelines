// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { loadGraphExpansion, saveGraphExpansion } from './GraphExpansionState';

beforeEach(() => localStorage.clear());
afterEach(() => vi.restoreAllMocks());

it.each(['default', 'all', 'none'] as const)(
  'round-trips %s mode and individual choices',
  (mode) => {
    const state = { mode, collapsed: new Set(['parent']), expanded: new Set(['child']) };
    saveGraphExpansion('run:1', state);
    expect(loadGraphExpansion('run:1')).toEqual(state);
    expect(loadGraphExpansion('run:2').mode).toBe('default');
    expect(loadGraphExpansion('pipeline-version:1').collapsed.size).toBe(0);
  },
);

it.each([
  '{invalid',
  'null',
  '{}',
  '{"mode":"unknown","collapsed":[],"expanded":[]}',
  '{"mode":"all","collapsed":[1],"expanded":[]}',
  '{"mode":"none","collapsed":[],"expanded":{}}',
])('ignores malformed preferences: %s', (saved) => {
  localStorage.setItem('kfp:graph-expansion:v1:run:1', saved);
  expect(loadGraphExpansion('run:1')).toEqual({
    mode: 'default',
    collapsed: new Set(),
    expanded: new Set(),
  });
});

it('does not read or write storage for anonymous graphs', () => {
  const read = vi.spyOn(localStorage, 'getItem');
  const write = vi.spyOn(localStorage, 'setItem');
  saveGraphExpansion(undefined, loadGraphExpansion());
  expect(read).not.toHaveBeenCalled();
  expect(write).not.toHaveBeenCalled();
});

it('tolerates blocked reads and full storage', () => {
  vi.spyOn(localStorage, 'getItem').mockImplementation(() => {
    throw new Error('Blocked');
  });
  expect(loadGraphExpansion('run:1').mode).toBe('default');
  vi.spyOn(localStorage, 'setItem').mockImplementation(() => {
    throw new Error('Quota exceeded');
  });
  expect(() => saveGraphExpansion('run:1', loadGraphExpansion())).not.toThrow();
});
