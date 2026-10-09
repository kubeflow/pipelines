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

import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router';
import { FlaskConical, PlayCircle, Settings, Workflow } from 'lucide-react';
import { AppShell } from './AppShell';
import type { AppShellProps } from './AppShell';
import { ThemeProvider } from './ThemeProvider';

const items = [
  {
    id: 'pipelines',
    elementId: 'pipelinesBtn',
    label: 'Pipelines',
    href: '/pipelines',
    icon: Workflow,
  },
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
    expect(screen.getByRole('link', { name: 'Pipelines' })).toHaveAttribute('id', 'pipelinesBtn');
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
    await userEvent.click(screen.getByRole('button', { name: 'Theme: Light' }));
    for (const name of ['System', 'Light', 'Dark']) {
      expect(screen.getByRole('menuitemradio', { name })).toBeVisible();
    }
    expect(screen.getByRole('menuitemradio', { name: 'Light' })).toHaveAttribute(
      'aria-checked',
      'true',
    );
    await userEvent.click(await screen.findByRole('menuitemradio', { name: 'Dark' }));
    expect(screen.getByRole('button', { name: 'Theme: Dark' })).toBeVisible();
    expect(localStorage.getItem('kfp.theme')).toBe('dark');
  });

  it('opens the theme menu by keyboard and restores focus on Escape', async () => {
    renderShell();
    const user = userEvent.setup();
    const trigger = screen.getByRole('button', { name: 'Theme: Light' });
    for (let step = 0; step < 15 && document.activeElement !== trigger; step++) await user.tab();
    expect(trigger).toHaveFocus();
    await user.keyboard('{Enter}');
    expect(await screen.findByRole('menu', { name: 'Appearance' })).toBeVisible();
    await user.keyboard('{Escape}');
    await waitFor(() => expect(screen.queryByRole('menu')).not.toBeInTheDocument());
    expect(trigger).toHaveFocus();
    expect(localStorage.getItem('kfp.theme')).toBeNull();
  });

  it('keeps the theme menu in the selected palette and persists all choices', async () => {
    const view = renderShell();
    for (const name of ['Dark', 'System', 'Light']) {
      await userEvent.click(screen.getByRole('button', { name: /^Theme: / }));
      await userEvent.click(await screen.findByRole('menuitemradio', { name }));
      await waitFor(() => expect(screen.queryByRole('menu')).not.toBeInTheDocument());
      expect(localStorage.getItem('kfp.theme')).toBe(name.toLowerCase());
      expect(screen.getByRole('button', { name: `Theme: ${name}` })).toBeVisible();
    }
    await userEvent.click(screen.getByRole('button', { name: 'Theme: Light' }));
    await userEvent.click(await screen.findByRole('menuitemradio', { name: 'Dark' }));
    view.unmount();
    renderShell();
    await userEvent.click(screen.getByRole('button', { name: 'Theme: Dark' }));
    expect(await screen.findByRole('menu', { name: 'Appearance' })).toBeVisible();
    expect(screen.getByRole('menu').closest('.kfp-theme')).toHaveClass('dark');
  });

  it('exposes a tooltip on keyboard focus for the utility links', async () => {
    renderShell();
    const user = userEvent.setup();
    const link = screen.getByRole('link', { name: 'Documentation' });
    for (let step = 0; step < 15 && document.activeElement !== link; step++) await user.tab();
    expect(link).toHaveFocus();
    const tooltip = await screen.findByRole('tooltip');
    expect(tooltip).toHaveTextContent('Documentation');
    expect(link).toHaveAttribute('aria-describedby', tooltip.id);
    expect(link).toHaveAccessibleDescription('Documentation');
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

  it('reserves the expanded metadata container without fabricating unavailable values or links', () => {
    const { container } = renderShell();
    const metadata = container.querySelector('.kfp-shell-metadata');
    expect(metadata).toBeEmptyDOMElement();
    expect(metadata).not.toHaveAttribute('role');
    expect(container.querySelector('a:not([href]), a[href=""]')).not.toBeInTheDocument();
  });

  it('preserves metadata slots as optional deployment values arrive and disappear', () => {
    function Fixture({ metadata }: Pick<AppShellProps, 'metadata'>) {
      return (
        <MemoryRouter>
          <ThemeProvider defaultTheme='light'>
            <AppShell
              items={items}
              currentPath='/runs'
              version='v2.19.0'
              versionHref='https://example.test/version'
              metadata={metadata}
            >
              <h1>Runs</h1>
            </AppShell>
          </ThemeProvider>
        </MemoryRouter>
      );
    }
    const { container, rerender } = render(<Fixture />);
    const metadata = container.querySelector('.kfp-shell-metadata');
    const version = screen.getByRole('link', { name: 'v2.19.0' });
    expect(version).toHaveClass('kfp-shell-metadata-version');
    expect(metadata?.children).toHaveLength(1);
    rerender(
      <Fixture
        metadata={{
          clusterName: 'cluster-a',
          clusterHref: 'https://example.test/cluster-a',
          projectId: 'project-a',
        }}
      />,
    );
    expect(container.querySelector('.kfp-shell-metadata')).toBe(metadata);
    expect(screen.getByRole('link', { name: 'v2.19.0' })).toBe(version);
    expect(screen.getByRole('link', { name: 'Cluster: cluster-a' })).toHaveClass(
      'kfp-shell-metadata-cluster',
    );
    expect(screen.getByText('Project: project-a')).toHaveClass('kfp-shell-metadata-project');
    rerender(<Fixture />);
    expect(metadata?.children).toHaveLength(1);
    expect(screen.queryByText('Project: project-a')).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Cluster: cluster-a' })).not.toBeInTheDocument();
  });

  it('omits metadata reservations from manually and responsively collapsed navigation', async () => {
    const { container } = renderShell({
      version: 'v2.19.0',
      metadata: { projectId: 'project-a' },
    });
    await userEvent.click(screen.getByRole('button', { name: 'Collapse navigation' }));
    expect(container.querySelector('.kfp-shell-metadata')).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Expand navigation' }));
    expect(screen.getByText('Project: project-a')).toBeInTheDocument();
    resize(600);
    expect(container.querySelector('.kfp-shell-metadata')).not.toBeInTheDocument();
    resize(1280);
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
  it('keeps Search available with both expanded and collapsed navigation', async () => {
    const onSearch = vi.fn();
    renderShell({ onSearch });
    const search = screen.getByRole('button', { name: 'Search', exact: true });
    expect(search).toHaveAttribute('aria-keyshortcuts', 'Control+k Meta+k');
    expect(search).toHaveAccessibleDescription('Search (Ctrl/Cmd+K)');
    await userEvent.click(search);
    await userEvent.click(screen.getByRole('button', { name: 'Collapse navigation' }));
    await userEvent.click(screen.getByRole('button', { name: 'Search' }));
    expect(onSearch).toHaveBeenCalledTimes(2);
  });
});
