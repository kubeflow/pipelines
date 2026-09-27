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

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useContext, useState } from 'react';
import { MemoryRouter } from 'react-router';
import { PlayCircle } from 'lucide-react';
import type { BuildInfo } from 'src/lib/Apis';
import { BuildInfoContext } from 'src/lib/BuildInfo';
import { GkeMetadataContext } from 'src/lib/GkeMetadata';
import type { GkeMetadata } from 'src/lib/GkeMetadata';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { KFP_FLAGS } from 'src/lib/Flags';
import { ApplicationShell } from './ApplicationShell';

const items = [{ id: 'runs', label: 'Runs', href: '/runs', icon: PlayCircle }];
const originalHideSideNav = KFP_FLAGS.HIDE_SIDENAV;

function StatefulPage() {
  const namespace = useContext(NamespaceContext);
  const [value, setValue] = useState('');
  return (
    <>
      <output aria-label='Page namespace'>{namespace}</output>
      <input
        aria-label='Draft name'
        value={value}
        onChange={(event) => setValue(event.target.value)}
      />
    </>
  );
}

function ShellFixture({
  buildInfo,
  namespace = 'team-one',
  metadata = {},
}: {
  buildInfo?: BuildInfo;
  namespace?: string;
  metadata?: GkeMetadata;
}) {
  return (
    <MemoryRouter>
      <NamespaceContext.Provider value={namespace}>
        <BuildInfoContext.Provider value={buildInfo}>
          <GkeMetadataContext.Provider value={metadata}>
            <ApplicationShell items={items} currentPath='/runs'>
              <StatefulPage />
            </ApplicationShell>
          </GkeMetadataContext.Provider>
        </BuildInfoContext.Provider>
      </NamespaceContext.Provider>
    </MemoryRouter>
  );
}

beforeEach(() => {
  localStorage.clear();
  KFP_FLAGS.HIDE_SIDENAV = false;
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1280 });
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }));
});

afterEach(() => {
  KFP_FLAGS.HIDE_SIDENAV = originalHideSideNav;
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe('ApplicationShell', () => {
  it('prefers API build metadata and retains the full commit destination', () => {
    render(
      <ShellFixture
        buildInfo={{
          apiServerTagName: 'api-version',
          apiServerCommitHash: '123456789abcdef',
          frontendTagName: 'frontend-version',
          frontendCommitHash: 'abcdef123456789',
          buildDate: '2026-09-26T00:00:00Z',
        }}
      />,
    );
    expect(screen.getByRole('link', { name: 'api-version' })).toHaveAttribute(
      'href',
      'https://github.com/kubeflow/pipelines/commit/123456789abcdef',
    );
    expect(screen.getByRole('link', { name: 'api-version' }).parentElement).toHaveAttribute(
      'title',
      '9/26/2026 · 1234567',
    );
    expect(screen.queryByText('frontend-version')).not.toBeInTheDocument();
  });

  it('falls back to frontend metadata and then an unknown version with a repository link', () => {
    const view = render(
      <ShellFixture
        buildInfo={{ frontendTagName: 'frontend-version', frontendCommitHash: 'abc123456' }}
      />,
    );
    expect(screen.getByRole('link', { name: 'frontend-version' })).toHaveAttribute(
      'href',
      'https://github.com/kubeflow/pipelines/commit/abc123456',
    );
    view.rerender(<ShellFixture />);
    expect(screen.getByRole('link', { name: 'unknown' })).toHaveAttribute(
      'href',
      'https://github.com/kubeflow/pipelines',
    );
  });

  it('retains cluster navigation when project and cluster metadata are both available', () => {
    const view = render(<ShellFixture metadata={{ clusterName: 'cluster one' }} />);
    expect(screen.queryByText('Cluster: cluster one')).not.toBeInTheDocument();
    view.rerender(
      <ShellFixture metadata={{ clusterName: 'cluster one', projectId: 'project+one' }} />,
    );
    const link = screen.getByRole('link', { name: 'Cluster: cluster one' });
    const url = new URL(link.getAttribute('href')!);
    expect(url.origin).toBe('https://console.cloud.google.com');
    expect(url.pathname).toBe('/kubernetes/list');
    expect(url.searchParams.get('project')).toBe('project+one');
    expect(url.searchParams.get('filter')).toBe('name:cluster one');
    expect(link).toHaveAttribute('target', '_blank');
  });

  it('updates namespace and metadata without resetting page or shell preferences', async () => {
    const view = render(<ShellFixture />);
    await userEvent.type(screen.getByRole('textbox', { name: 'Draft name' }), 'keep this draft');
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Theme' }), 'dark');
    await userEvent.click(screen.getByRole('button', { name: 'Collapse navigation' }));
    view.rerender(
      <ShellFixture namespace='team-two' buildInfo={{ frontendTagName: 'next-version' }} />,
    );
    expect(screen.getByLabelText('Page namespace')).toHaveTextContent('team-two');
    expect(screen.getByRole('textbox', { name: 'Draft name' })).toHaveValue('keep this draft');
    expect(screen.getByRole('complementary')).toHaveAttribute('data-collapsed', 'true');
    expect(screen.getByRole('combobox', { name: 'Theme' })).toHaveValue('dark');
    await userEvent.click(screen.getByRole('button', { name: 'Expand navigation' }));
    expect(screen.getByRole('link', { name: 'next-version' })).toBeVisible();
  });

  it('honors the host hidden-navigation flag while retaining context and content', () => {
    KFP_FLAGS.HIDE_SIDENAV = true;
    render(<ShellFixture namespace='embedded-team' />);
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument();
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument();
    expect(screen.getByLabelText('Page namespace')).toHaveTextContent('embedded-team');
    expect(screen.getByRole('main')).toContainElement(screen.getByRole('textbox'));
  });
});
