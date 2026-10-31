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

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation, useNavigate } from 'react-router';
import type { ReactNode } from 'react';
import RunsPage, { RunsPageProps, RunsPageTab } from './RunsPage';

vi.mock('./AllRunsList', () => ({
  default: ({ scopeControl }: { scopeControl?: ReactNode }) => (
    <div>
      {scopeControl}
      <div>Active run content</div>
    </div>
  ),
}));
vi.mock('./ArchivedRuns', () => ({
  default: ({ scopeControl }: { scopeControl?: ReactNode }) => (
    <div>
      {scopeControl}
      <div>Archived run content</div>
    </div>
  ),
}));
function RoutedRuns() {
  const navigate = useNavigate();
  const location = useLocation();
  return (
    <>
      <RunsPage
        {...props(location.pathname === '/archive/runs' ? RunsPageTab.ARCHIVE : RunsPageTab.RUNS)}
        navigate={navigate}
      />
      <button onClick={() => navigate(-1)}>Back</button>
    </>
  );
}
function Location() {
  return <output aria-label='Location'>{useLocation().pathname}</output>;
}
function props(view = RunsPageTab.RUNS): RunsPageProps {
  return {
    navigate: vi.fn(),
    location: { pathname: '/runs', search: '', hash: '', state: null, key: 'test' },
    params: {},
    toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
    view,
  };
}
it.each([RunsPageTab.RUNS, RunsPageTab.ARCHIVE])(
  'renders the requested view %s and marks its scope toggle',
  (view) => {
    render(
      <MemoryRouter>
        <RunsPage {...props(view)} />
      </MemoryRouter>,
    );
    const label = view === RunsPageTab.RUNS ? 'Active' : 'Archived';
    expect(screen.getByRole('button', { name: label })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText(`${label} run content`)).toBeVisible();
  },
);
it('preserves active and archived route destinations with keyboard-operable controls', async () => {
  render(
    <MemoryRouter initialEntries={['/runs']}>
      <RoutedRuns />
      <Location />
    </MemoryRouter>,
  );
  await userEvent.click(screen.getByRole('button', { name: 'Archived' }));
  expect(screen.getByLabelText('Location')).toHaveTextContent('/archive/runs');
  await userEvent.click(screen.getByRole('button', { name: 'Back' }));
  expect(screen.getByLabelText('Location')).toHaveTextContent('/runs');
  await userEvent.click(screen.getByRole('button', { name: 'Archived' }));
  screen.getByRole('button', { name: 'Active' }).focus();
  await userEvent.keyboard('{Enter}');
  expect(screen.getByLabelText('Location')).toHaveTextContent('/runs');
});
