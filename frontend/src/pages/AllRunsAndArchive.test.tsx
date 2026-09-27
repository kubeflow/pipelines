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
import { MemoryRouter, useLocation } from 'react-router';
import AllRunsAndArchive, {
  AllRunsAndArchiveProps,
  AllRunsAndArchiveTab,
} from './AllRunsAndArchive';

vi.mock('./AllRunsList', () => ({ default: () => <div>Active run content</div> }));
vi.mock('./ArchivedRuns', () => ({ default: () => <div>Archived run content</div> }));
function Location() {
  return <output aria-label='Location'>{useLocation().pathname}</output>;
}
function props(view = AllRunsAndArchiveTab.RUNS): AllRunsAndArchiveProps {
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
it.each([AllRunsAndArchiveTab.RUNS, AllRunsAndArchiveTab.ARCHIVE])(
  'renders the requested view %s and marks its navigation link',
  (view) => {
    render(
      <MemoryRouter>
        <AllRunsAndArchive {...props(view)} />
      </MemoryRouter>,
    );
    const label = view === AllRunsAndArchiveTab.RUNS ? 'Active' : 'Archived';
    expect(screen.getByRole('link', { name: label })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByText(`${label} run content`)).toBeVisible();
  },
);
it('preserves active and archived route destinations with keyboard-operable links', async () => {
  render(
    <MemoryRouter initialEntries={['/runs']}>
      <AllRunsAndArchive {...props()} />
      <Location />
    </MemoryRouter>,
  );
  await userEvent.click(screen.getByRole('link', { name: 'Archived' }));
  expect(screen.getByLabelText('Location')).toHaveTextContent('/archive/runs');
  screen.getByRole('link', { name: 'Active' }).focus();
  await userEvent.keyboard('{Enter}');
  expect(screen.getByLabelText('Location')).toHaveTextContent('/runs');
});
