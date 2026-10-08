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

import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createRef } from 'react';
import { MemoryRouter } from 'react-router';
import RunList, { RunListProps } from './RunList';
import { Apis, ListRequest } from 'src/lib/Apis';
import { V2beta1ListRunsResponse, V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1PipelineVersion, ResponseError } from 'src/apisv2beta1/pipeline';

class TestRunList extends RunList {
  load(request: ListRequest) {
    return this._loadRuns(request);
  }
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}
const initial: V2beta1Run = { run_id: 'initial', display_name: 'Initial run', state: 'SUCCEEDED' };
const latest: V2beta1Run = { run_id: 'latest', display_name: 'Latest run', state: 'PAUSED' };
function renderList(overrides: Partial<RunListProps> = {}) {
  const ref = createRef<TestRunList>();
  const props: RunListProps = {
    navigate: vi.fn(),
    params: {},
    onError: vi.fn(),
    location: { pathname: '/runs', search: '', hash: '', state: null, key: 'test' },
    ...overrides,
  };
  render(
    <MemoryRouter>
      <TestRunList ref={ref} {...props} />
    </MemoryRouter>,
  );
  return { ref, props };
}
beforeEach(() => {
  vi.spyOn(Apis.runServiceApiV2, 'listRuns').mockResolvedValue({ runs: [initial] });
  vi.spyOn(Apis.experimentServiceApiV2, 'listExperiments').mockResolvedValue({ experiments: [] });
});
afterEach(() => {
  vi.restoreAllMocks();
});

it('keeps the newest rows when an older list response finishes later', async () => {
  const { ref } = renderList();
  await screen.findByRole('link', { name: 'Initial run' });
  const old = deferred<V2beta1ListRunsResponse>();
  vi.mocked(Apis.runServiceApiV2.listRuns)
    .mockImplementationOnce(() => old.promise)
    .mockResolvedValueOnce({ runs: [latest] });
  let older!: Promise<string>;
  act(() => {
    older = ref.current!.load({});
  });
  await act(async () => {
    await ref.current!.load({});
  });
  expect(screen.getByRole('link', { name: 'Latest run' })).toBeVisible();
  await act(async () => {
    old.resolve({ runs: [initial] });
    await older;
  });
  expect(screen.getByRole('link', { name: 'Latest run' })).toBeVisible();
  expect(screen.queryByRole('link', { name: 'Initial run' })).toBeNull();
});

it('does not let slow related-resource enrichment replace newer runs', async () => {
  const { ref } = renderList();
  await screen.findByRole('link', { name: 'Initial run' });
  const oldVersion = deferred<V2beta1PipelineVersion>();
  const versionSpy = vi
    .spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion')
    .mockImplementation(() => oldVersion.promise);
  vi.mocked(Apis.runServiceApiV2.listRuns)
    .mockResolvedValueOnce({
      runs: [
        {
          ...initial,
          pipeline_version_reference: { pipeline_id: 'pipeline', pipeline_version_id: 'version' },
        },
      ],
    })
    .mockResolvedValueOnce({ runs: [latest] });
  let older!: Promise<string>;
  act(() => {
    older = ref.current!.load({});
  });
  await waitFor(() => expect(versionSpy).toHaveBeenCalled());
  await act(async () => {
    await ref.current!.load({});
  });
  await act(async () => {
    oldVersion.resolve({
      pipeline_id: 'pipeline',
      pipeline_version_id: 'version',
      display_name: 'Version',
    });
    await older;
  });
  expect(screen.getByRole('link', { name: 'Latest run' })).toBeVisible();
  expect(screen.queryByRole('link', { name: 'Initial run' })).toBeNull();
});

it('does not report a stale list failure after a newer request succeeds', async () => {
  const { ref, props } = renderList();
  await screen.findByRole('link', { name: 'Initial run' });
  const old = deferred<V2beta1ListRunsResponse>();
  vi.mocked(Apis.runServiceApiV2.listRuns)
    .mockImplementationOnce(() => old.promise)
    .mockResolvedValueOnce({ runs: [latest] });
  let older!: Promise<string>;
  act(() => {
    older = ref.current!.load({});
  });
  await act(async () => {
    await ref.current!.load({});
  });
  await act(async () => {
    old.reject(new Error('old failure'));
    await older;
  });
  expect(props.onError).not.toHaveBeenCalled();
  expect(screen.getByRole('link', { name: 'Latest run' })).toBeVisible();
});

it('retains embedded pipeline and recurring-run links alongside named state and full run ID', async () => {
  vi.mocked(Apis.runServiceApiV2.listRuns).mockResolvedValue({
    runs: [{ ...initial, recurring_run_id: 'schedule', pipeline_spec: {} }],
  });
  renderList();
  await screen.findByRole('link', { name: 'Initial run' });
  expect(screen.getByText('initial')).toBeVisible();
  expect(screen.getByText('Succeeded')).toBeVisible();
  expect(screen.getByRole('link', { name: '[View config]' })).toHaveAttribute(
    'href',
    '/recurringrun/details/schedule',
  );
  expect(screen.getByRole('link', { name: '[View pipeline]' }).getAttribute('href')).toContain(
    'fromRecurringRun=schedule',
  );
  expect(screen.getByRole('searchbox', { name: 'Filter runs by name' })).toBeVisible();
  expect(vi.mocked(Apis.runServiceApiV2.listRuns).mock.calls[0][6]).toBe(true);
});

it.each([404, 403, 500])(
  'distinguishes a removed version from an HTTP %s failure',
  async (status) => {
    vi.mocked(Apis.runServiceApiV2.listRuns).mockResolvedValue({
      runs: [
        {
          ...initial,
          pipeline_version_reference: { pipeline_id: 'pipeline', pipeline_version_id: 'version' },
        },
      ],
    });
    vi.spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion').mockRejectedValue(
      new ResponseError(new Response('Version unavailable', { status })),
    );
    renderList();
    await screen.findByRole('link', { name: 'Initial run' });
    if (status === 404) {
      expect(screen.getByText('Unavailable')).toBeVisible();
      expect(screen.queryByRole('note')).toBeNull();
    } else {
      expect(screen.getByRole('note')).toHaveTextContent(
        'Failed to get associated pipeline version',
      );
      expect(screen.queryByText('Unavailable')).toBeNull();
    }
  },
);

it('keeps failed reads distinct from empty results and signals recovery through a filter request', async () => {
  vi.mocked(Apis.runServiceApiV2.listRuns).mockRejectedValue(new Error('Access denied'));
  const onLoadSuccess = vi.fn();
  renderList({ onLoadSuccess, storageState: 'AVAILABLE' });
  await screen.findByText('Runs could not be loaded. Use Refresh to try again.');
  expect(screen.queryByText('No available runs found.')).toBeNull();
  expect(onLoadSuccess).not.toHaveBeenCalled();
  vi.mocked(Apis.runServiceApiV2.listRuns).mockResolvedValue({ runs: [latest] });
  fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'Latest' } });
  await screen.findByRole('link', { name: 'Latest run' });
  expect(screen.queryByText('Runs could not be loaded. Use Refresh to try again.')).toBeNull();
  expect(onLoadSuccess).toHaveBeenCalledOnce();
});

it('selects raw run IDs through rows and encodes detail links once', async () => {
  const id = 'run/percent%value';
  vi.mocked(Apis.runServiceApiV2.listRuns).mockResolvedValue({
    runs: [{ ...initial, run_id: id }],
  });
  const onSelectionChange = vi.fn();
  const { props } = renderList({ onSelectionChange });
  const link = await screen.findByRole('link', { name: 'Initial run' });
  const destination = '/runs/details/run%2Fpercent%25value';
  expect(link).toHaveAttribute('href', destination);
  expect(link).toHaveAttribute('data-run-id', id);
  fireEvent.click(screen.getByTestId('table-row'));
  expect(onSelectionChange).toHaveBeenCalledExactlyOnceWith([id]);
  expect(props.navigate).not.toHaveBeenCalled();
});
