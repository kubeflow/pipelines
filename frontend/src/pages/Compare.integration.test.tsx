/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
 */
import { act, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { forwardRef, useImperativeHandle } from 'react';
import { MemoryRouter } from 'react-router';
import { Apis } from 'src/lib/Apis';
import { queryKeys } from 'src/hooks/queryKeys';
import Compare from './Compare';
import { PageProps } from './Page';

vi.mock('./RunList', () => ({
  default: forwardRef(function MockRunList(_props, ref) {
    useImperativeHandle(ref, () => ({ refresh: vi.fn() }));
    return <div>Run overview</div>;
  }),
}));

let client: QueryClient;
let props: PageProps;
beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  props = {
    navigate: vi.fn(),
    params: {},
    location: { search: '?runlist=run-1,run-2' } as any,
    toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
  };
  vi.spyOn(Apis.runServiceApiV2, 'tasks').mockResolvedValue({ tasks: [] });
});
afterEach(() => {
  client.clear();
  vi.restoreAllMocks();
});
function page() {
  return (
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <Compare {...props} />
      </QueryClientProvider>
    </MemoryRouter>
  );
}
function run(id: string) {
  return {
    run_id: id,
    display_name: id,
    state: 'SUCCEEDED' as const,
    runtime_config: { parameters: { input: `value-${id}` } },
  };
}

it('loads native runs once without a version-routing preflight', async () => {
  const getRun = vi.spyOn(Apis.runServiceApiV2, 'getRun').mockImplementation(async (id) => run(id));
  render(page());
  await screen.findByText('value-run-1');
  await screen.findByText('value-run-2');
  expect(getRun.mock.calls).toEqual([['run-1'], ['run-2']]);
});

it('shows loading indicators while native comparison data is pending', () => {
  vi.spyOn(Apis.runServiceApiV2, 'getRun').mockReturnValue(new Promise(() => {}));
  render(page());
  expect(screen.getAllByRole('circularprogress').length).toBeGreaterThan(0);
});

it('retains successful cached runs while retrying only a failed native query', async () => {
  const getRun = vi.spyOn(Apis.runServiceApiV2, 'getRun').mockImplementation(async (id) => {
    if (id === 'run-2') throw new Error('Temporarily unavailable');
    return run(id);
  });
  render(page());
  await screen.findByText('value-run-1');
  await waitFor(() =>
    expect(props.updateBanner).toHaveBeenCalledWith(expect.objectContaining({ mode: 'warning' })),
  );
  getRun.mockImplementation(async (id) => run(id));
  await act(async () => {
    await client.invalidateQueries({ queryKey: queryKeys.v2RunComparison('run-2') });
  });
  await screen.findByText('value-run-2');
  expect(getRun.mock.calls.filter(([id]) => id === 'run-1')).toHaveLength(1);
  expect(getRun.mock.calls.filter(([id]) => id === 'run-2')).toHaveLength(2);
  expect(props.updateBanner).toHaveBeenLastCalledWith({});
});

it('reports failed native queries once even if the page shell rerenders', async () => {
  vi.spyOn(Apis.runServiceApiV2, 'getRun').mockRejectedValue(new Error('Unavailable'));
  const view = render(page());
  await waitFor(() =>
    expect(props.updateBanner).toHaveBeenCalledWith(expect.objectContaining({ mode: 'error' })),
  );
  expect(screen.queryAllByRole('circularprogress')).toHaveLength(0);
  const calls = vi.mocked(props.updateBanner).mock.calls.length;
  view.rerender(page());
  expect(props.updateBanner).toHaveBeenCalledTimes(calls);
});
