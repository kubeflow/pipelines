/*
 * Copyright 2026 The Kubeflow Authors
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

import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router';
import { PlayCircle, Workflow } from 'lucide-react';
import { Apis } from 'src/lib/Apis';
import type { V2beta1ListRunsResponse } from 'src/apisv2beta1/run';
import { CommandPalette } from './CommandPalette';
import { ThemeProvider } from './ThemeProvider';

const items = [
  { id: 'pipelines', label: 'Pipelines', href: '/pipelines', icon: Workflow },
  { id: 'runs', label: 'Runs', href: '/runs', icon: PlayCircle },
];

function LocationProbe() {
  const { pathname } = useLocation();
  return <output aria-label='Location'>{pathname}</output>;
}

function Fixture({
  namespace = 'team-one',
  open = true,
  requireNamespace = false,
  onClose = vi.fn(),
}: {
  namespace?: string;
  open?: boolean;
  requireNamespace?: boolean;
  onClose?: () => void;
}) {
  return (
    <MemoryRouter>
      <ThemeProvider defaultTheme='light'>
        <CommandPalette
          open={open}
          onClose={onClose}
          namespace={namespace || undefined}
          requireNamespace={requireNamespace}
          items={items}
        />
        <LocationProbe />
      </ThemeProvider>
    </MemoryRouter>
  );
}

function search(value: string) {
  fireEvent.change(screen.getByRole('searchbox'), { target: { value } });
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

beforeEach(() => {
  vi.spyOn(Apis.runServiceApiV2, 'listRuns').mockResolvedValue({ runs: [] });
  vi.spyOn(Apis.pipelineServiceApiV2, 'listPipelines').mockResolvedValue({ pipelines: [] });
  vi.spyOn(Apis.experimentServiceApiV2, 'listExperiments').mockResolvedValue({ experiments: [] });
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }));
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('CommandPalette', () => {
  it('debounces names of at least two characters and requests one bounded namespace-scoped page per type', async () => {
    vi.useFakeTimers();
    render(<Fixture />);
    search('t');
    await act(() => vi.advanceTimersByTimeAsync(300));
    expect(Apis.runServiceApiV2.listRuns).not.toHaveBeenCalled();
    search('tr');
    await act(() => vi.advanceTimersByTimeAsync(200));
    search('train');
    await act(() => vi.advanceTimersByTimeAsync(274));
    expect(Apis.runServiceApiV2.listRuns).not.toHaveBeenCalled();
    await act(() => vi.advanceTimersByTimeAsync(1));
    const filter = encodeURIComponent(
      JSON.stringify({
        predicates: [{ key: 'name', operation: 'IS_SUBSTRING', string_value: 'train' }],
      }),
    );
    const init = { signal: expect.any(AbortSignal) };
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledExactlyOnceWith(
      'team-one',
      undefined,
      undefined,
      5,
      'created_at desc',
      filter,
      true,
      undefined,
      init,
    );
    expect(Apis.pipelineServiceApiV2.listPipelines).toHaveBeenCalledExactlyOnceWith(
      'team-one',
      undefined,
      5,
      'created_at desc',
      filter,
      init,
    );
    expect(Apis.experimentServiceApiV2.listExperiments).toHaveBeenCalledExactlyOnceWith(
      undefined,
      5,
      'created_at desc',
      filter,
      'team-one',
      init,
    );
    const runSignal = (vi.mocked(Apis.runServiceApiV2.listRuns).mock.calls[0][8] as RequestInit)
      .signal;
    expect(
      (vi.mocked(Apis.pipelineServiceApiV2.listPipelines).mock.calls[0][5] as RequestInit).signal,
    ).toBe(runSignal);
    expect(
      (vi.mocked(Apis.experimentServiceApiV2.listExperiments).mock.calls[0][5] as RequestInit)
        .signal,
    ).toBe(runSignal);
  });

  it('uses real encoded IDs, labels archived matches, and never follows another results page', async () => {
    vi.mocked(Apis.runServiceApiV2.listRuns).mockResolvedValue({
      runs: [
        { display_name: 'Missing id' },
        ...Array.from({ length: 7 }, (_, index) => ({
          run_id: `run/${index} space`,
          display_name: `Training ${index}`,
          storage_state: 'ARCHIVED' as const,
        })),
      ],
      next_page_token: 'do-not-follow',
    });
    vi.mocked(Apis.pipelineServiceApiV2.listPipelines).mockResolvedValue({
      pipelines: [{ pipeline_id: 'pipeline/id', display_name: 'Training pipeline' }],
    });
    vi.mocked(Apis.experimentServiceApiV2.listExperiments).mockResolvedValue({
      experiments: [
        {
          experiment_id: 'experiment/id',
          display_name: 'Training experiment',
          storage_state: 'ARCHIVED',
        },
      ],
    });
    render(<Fixture />);
    search('Training');
    const run = await screen.findByRole('link', { name: 'Training 0 (Archived)' });
    expect(run).toHaveAttribute('href', '/runs/details/run%2F0%20space');
    expect(within(screen.getByRole('region', { name: 'Runs' })).getAllByRole('link')).toHaveLength(
      5,
    );
    expect(screen.queryByRole('link', { name: 'Missing id' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Training pipeline' })).toHaveAttribute(
      'href',
      '/pipelines/details/pipeline%2Fid',
    );
    expect(screen.getByRole('link', { name: 'Training experiment (Archived)' })).toHaveAttribute(
      'href',
      '/experiments/details/experiment%2Fid',
    );
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledTimes(1);
  });

  it('keeps successful matches on partial failure and retries without retaining stale error UI', async () => {
    vi.mocked(Apis.runServiceApiV2.listRuns).mockResolvedValue({
      runs: [{ run_id: 'run-one', display_name: 'Training run' }],
    });
    vi.mocked(Apis.pipelineServiceApiV2.listPipelines).mockRejectedValueOnce(
      new Error('forbidden'),
    );
    render(<Fixture />);
    search('train');
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not search Pipelines.');
    expect(screen.getByRole('link', { name: 'Training run' })).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: 'Retry search' }));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Training run' })).not.toBeInTheDocument();
    expect(await screen.findByRole('link', { name: 'Training run' })).toBeVisible();
    expect(Apis.pipelineServiceApiV2.listPipelines).toHaveBeenCalledTimes(2);
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('distinguishes an empty successful search from a failed search', async () => {
    render(<Fixture />);
    search('missing');
    expect(await screen.findByText('No matching resources.')).toBeVisible();
    vi.mocked(Apis.runServiceApiV2.listRuns).mockRejectedValue(new Error('unavailable'));
    search('unavailable');
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not search Runs.');
    expect(screen.queryByText('No matching resources.')).not.toBeInTheDocument();
  });

  it('hides old results immediately, aborts obsolete reads, and rejects late query responses', async () => {
    const slow = deferred<V2beta1ListRunsResponse>();
    vi.mocked(Apis.runServiceApiV2.listRuns)
      .mockResolvedValueOnce({ runs: [{ run_id: 'old', display_name: 'Earlier match' }] })
      .mockReturnValueOnce(slow.promise)
      .mockResolvedValue({ runs: [{ run_id: 'new', display_name: 'Current match' }] });
    render(<Fixture />);
    search('first');
    await screen.findByRole('link', { name: 'Earlier match' });
    search('slow');
    expect(screen.queryByRole('link', { name: 'Earlier match' })).not.toBeInTheDocument();
    await waitFor(() => expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledTimes(2));
    const signal = (vi.mocked(Apis.runServiceApiV2.listRuns).mock.calls[1][8] as RequestInit)
      .signal;
    search('first');
    // Returning to an earlier term still requires a fresh response.
    expect(screen.queryByRole('link', { name: 'Earlier match' })).not.toBeInTheDocument();
    expect(signal?.aborted).toBe(true);
    await screen.findByRole('link', { name: 'Current match' });
    await act(async () =>
      slow.resolve({ runs: [{ run_id: 'stale', display_name: 'Stale match' }] }),
    );
    expect(screen.queryByRole('link', { name: 'Stale match' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Current match' })).toBeVisible();
  });

  it('preserves the query across namespace changes but never exposes matches from the old namespace', async () => {
    const slow = deferred<V2beta1ListRunsResponse>();
    vi.mocked(Apis.runServiceApiV2.listRuns)
      .mockReturnValueOnce(slow.promise)
      .mockResolvedValue({ runs: [{ run_id: 'new', display_name: 'New namespace match' }] });
    const view = render(<Fixture />);
    search('match');
    await waitFor(() => expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledTimes(1));
    const signal = (vi.mocked(Apis.runServiceApiV2.listRuns).mock.calls[0][8] as RequestInit)
      .signal;
    view.rerender(<Fixture namespace='team-two' />);
    expect(screen.getByRole('searchbox')).toHaveValue('match');
    expect(signal?.aborted).toBe(true);
    await screen.findByRole('link', { name: 'New namespace match' });
    expect(vi.mocked(Apis.runServiceApiV2.listRuns).mock.calls[1][0]).toBe('team-two');
    await act(async () =>
      slow.resolve({ runs: [{ run_id: 'old', display_name: 'Old namespace match' }] }),
    );
    expect(screen.queryByRole('link', { name: 'Old namespace match' })).not.toBeInTheDocument();
    view.rerender(<Fixture namespace='team-one' />);
    expect(screen.queryByRole('link', { name: 'New namespace match' })).not.toBeInTheDocument();
  });

  it('cancels pending debounce and active requests when closed or unmounted', async () => {
    const view = render(<Fixture />);
    search('pending');
    view.rerender(<Fixture open={false} />);
    await new Promise((resolve) => setTimeout(resolve, 300));
    expect(Apis.runServiceApiV2.listRuns).not.toHaveBeenCalled();
    vi.mocked(Apis.runServiceApiV2.listRuns).mockReturnValue(new Promise(() => {}));
    view.rerender(<Fixture />);
    search('active');
    await waitFor(() => expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledTimes(1));
    const signal = (vi.mocked(Apis.runServiceApiV2.listRuns).mock.calls[0][8] as RequestInit)
      .signal;
    view.unmount();
    expect(signal?.aborted).toBe(true);
  });

  it('waits for a namespace on multi-user deployments while allowing standalone search', async () => {
    const view = render(<Fixture namespace='' requireNamespace />);
    search('train');
    expect(screen.getByText('Select a namespace to search resources.')).toBeVisible();
    await new Promise((resolve) => setTimeout(resolve, 300));
    expect(Apis.runServiceApiV2.listRuns).not.toHaveBeenCalled();
    view.rerender(<Fixture namespace='' />);
    expect(await screen.findByText('No matching resources.')).toBeVisible();
    expect(vi.mocked(Apis.runServiceApiV2.listRuns).mock.calls[0][0]).toBeUndefined();
  });

  it('supports arrow and Tab navigation through real links and opens the selected destination', async () => {
    const onClose = vi.fn();
    render(<Fixture onClose={onClose} />);
    const input = screen.getByRole('searchbox');
    await waitFor(() => expect(input).toHaveFocus());
    await userEvent.keyboard('{ArrowDown}');
    expect(screen.getByRole('link', { name: 'Pipelines' })).toHaveFocus();
    await userEvent.tab();
    expect(screen.getByRole('link', { name: 'Runs' })).toHaveFocus();
    await userEvent.keyboard('{ArrowDown}');
    expect(input).toHaveFocus();
    await userEvent.keyboard('{ArrowUp}{Enter}');
    expect(screen.getByLabelText('Location')).toHaveTextContent('/runs');
    expect(onClose).toHaveBeenCalledOnce();
  });
});
