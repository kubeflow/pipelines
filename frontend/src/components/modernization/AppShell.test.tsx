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

import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router';
import { FlaskConical, PlayCircle, Settings, Workflow } from 'lucide-react';
import { AppShell } from './AppShell';
import type { AppShellProps } from './AppShell';
import { ThemeProvider } from './ThemeProvider';

const items = [
  { id: 'pipelines', label: 'Pipelines', href: '/pipelines', icon: Workflow },
  { id: 'experiments', label: 'Experiments', href: '/experiments', icon: FlaskConical },
  { id: 'runs', label: 'Runs', href: '/runs', icon: PlayCircle },
];

function LocationProbe() {
  const location = useLocation();
  return <output aria-label='Current location'>{location.pathname + location.search}</output>;
}

function renderShell(overrides: Partial<AppShellProps> = {}) {
  return render(
    <MemoryRouter initialEntries={['/runs?runlist=first,second']}>
      <ThemeProvider defaultTheme='light'>
        <AppShell items={items} currentPath='/runs' namespace='team-ml' {...overrides}>
          <h1>Runs</h1>
          <LocationProbe />
        </AppShell>
      </ThemeProvider>
    </MemoryRouter>,
  );
}

function resize(width: number) {
  act(() => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: width });
    window.dispatchEvent(new Event('resize'));
  });
}

function sidebar() {
  return screen.getByRole('complementary', { name: 'Pipelines sidebar' });
}

beforeEach(() => {
  localStorage.clear();
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1280 });
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }));
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe('AppShell', () => {
  it('preserves navigation destinations and accepts explicit active states for nested routes', async () => {
    renderShell({
      currentPath: '/runs/details/id',
      items: items.map((item) => ({ ...item, active: item.id === 'runs' })),
    });
    expect(screen.getByRole('link', { name: 'Runs' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Pipelines' })).not.toHaveAttribute('aria-current');
    await userEvent.click(screen.getByRole('link', { name: 'Pipelines' }));
    expect(screen.getByLabelText('Current location')).toHaveTextContent('/pipelines');
  });

  it('persists manual collapse in the existing key without changing other preferences', async () => {
    localStorage.setItem('tablePageSize_runs', '50');
    localStorage.setItem('flags', '[{"name":"v2","active":true}]');
    const view = renderShell();
    await userEvent.click(screen.getByRole('button', { name: 'Collapse navigation' }));
    expect(localStorage.getItem('navbarCollapsed')).toBe('true');
    expect(sidebar()).toHaveAttribute('data-collapsed', 'true');
    expect(localStorage.getItem('tablePageSize_runs')).toBe('50');
    expect(localStorage.getItem('flags')).toBe('[{"name":"v2","active":true}]');
    view.unmount();
    renderShell();
    expect(sidebar()).toHaveAttribute('data-collapsed', 'true');
    await userEvent.click(screen.getByRole('button', { name: 'Expand navigation' }));
    expect(localStorage.getItem('navbarCollapsed')).toBe('false');
  });

  it.each([null, 'false', 'invalid'])(
    'collapses narrowly without overwriting preference %s',
    (value) => {
      if (value !== null) localStorage.setItem('navbarCollapsed', value);
      const save = vi.spyOn(localStorage, 'setItem');
      renderShell();
      expect(sidebar()).toHaveAttribute('data-collapsed', 'false');
      resize(1023);
      expect(sidebar()).toHaveAttribute('data-collapsed', 'true');
      expect(screen.getByRole('button', { name: /available on wider screens/ })).toBeDisabled();
      resize(1024);
      expect(sidebar()).toHaveAttribute('data-collapsed', 'false');
      expect(save).not.toHaveBeenCalled();
      expect(localStorage.getItem('navbarCollapsed')).toBe(value);
    },
  );

  it('restores a manually collapsed wide-screen choice after a narrow viewport', () => {
    localStorage.setItem('navbarCollapsed', 'true');
    renderShell();
    resize(800);
    resize(1280);
    expect(sidebar()).toHaveAttribute('data-collapsed', 'true');
    expect(localStorage.getItem('navbarCollapsed')).toBe('true');
  });

  it('starts collapsed on a narrow viewport even with a saved expanded choice', () => {
    localStorage.setItem('navbarCollapsed', 'false');
    resize(800);
    renderShell();
    expect(sidebar()).toHaveAttribute('data-collapsed', 'true');
    resize(1280);
    expect(sidebar()).toHaveAttribute('data-collapsed', 'false');
  });

  it('keeps navigation usable when storage access and persistence fail', async () => {
    vi.spyOn(localStorage, 'getItem').mockImplementation(() => {
      throw new DOMException('Storage unavailable', 'SecurityError');
    });
    vi.spyOn(localStorage, 'setItem').mockImplementation(() => {
      throw new DOMException('Storage full', 'QuotaExceededError');
    });
    renderShell();
    expect(sidebar()).toHaveAttribute('data-collapsed', 'false');
    await userEvent.click(screen.getByRole('button', { name: 'Collapse navigation' }));
    expect(sidebar()).toHaveAttribute('data-collapsed', 'true');
  });

  it('keeps every collapsed navigation link and theme choice named', async () => {
    localStorage.setItem('navbarCollapsed', 'true');
    renderShell();
    for (const name of [
      'Pipelines',
      'Experiments',
      'Runs',
      'Documentation',
      'GitHub',
      'Report an issue',
    ]) {
      expect(screen.getByRole('link', { name })).toHaveAccessibleName(name);
    }
    const theme = screen.getByRole('combobox', { name: 'Theme' });
    expect(screen.getByRole('option', { name: 'System' })).toHaveValue('system');
    expect(screen.getByRole('option', { name: 'Light' })).toHaveValue('light');
    expect(screen.getByRole('option', { name: 'Dark' })).toHaveValue('dark');
    await userEvent.selectOptions(theme, 'dark');
    expect(theme).toHaveValue('dark');
    expect(localStorage.getItem('kfp.theme')).toBe('dark');
  });

  it('focuses main content from the skip link without changing the route or query', async () => {
    renderShell();
    const user = userEvent.setup();
    await user.tab();
    expect(screen.getByRole('link', { name: 'Skip to main content' })).toHaveFocus();
    await user.keyboard('{Enter}');
    expect(screen.getByRole('main')).toHaveFocus();
    expect(screen.getByLabelText('Current location')).toHaveTextContent(
      '/runs?runlist=first,second',
    );
  });

  it('omits standalone navigation when the embedding host hides the sidebar', () => {
    renderShell({ hideSideNav: true });
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument();
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument();
    expect(screen.getByRole('main')).toHaveTextContent('Runs');
  });

  it('renders supplied secondary destinations and deployment metadata', () => {
    renderShell({
      secondaryItems: [
        { id: 'features', label: 'Frontend features', href: '/frontend_features', icon: Settings },
      ],
      version: 'v2.19.0',
      versionHref: 'https://github.com/kubeflow/pipelines/commit/example',
      metadata: {
        clusterName: 'cluster-a',
        clusterHref: 'https://example.test/cluster-a',
        projectId: 'project-a',
      },
    });
    expect(screen.getByText('team-ml')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Frontend features' })).toHaveAttribute(
      'href',
      '/frontend_features',
    );
    expect(screen.getByRole('link', { name: 'v2.19.0' })).toHaveAttribute(
      'href',
      'https://github.com/kubeflow/pipelines/commit/example',
    );
    expect(screen.getByRole('link', { name: 'Cluster: cluster-a' })).toHaveAttribute(
      'href',
      'https://example.test/cluster-a',
    );
    expect(screen.getByText('Project: project-a')).toBeInTheDocument();
  });

  it('removes each viewport subscription when unmounted', () => {
    const add = vi.spyOn(window, 'addEventListener');
    const remove = vi.spyOn(window, 'removeEventListener');
    const view = renderShell();
    view.unmount();
    const registrations = add.mock.calls.filter(([event]) => event === 'resize');
    const removals = remove.mock.calls.filter(([event]) => event === 'resize');
    expect(registrations.length).toBeGreaterThan(0);
    expect(removals).toEqual(registrations);
  });
});
