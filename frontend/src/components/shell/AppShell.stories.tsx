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

import type { Meta, StoryObj } from '@storybook/react';
import { FlaskConical, Package, PlayCircle, Repeat, Workflow } from 'lucide-react';
import { MemoryRouter, useLocation } from 'react-router';
import { AppShell } from './AppShell';
import type { AppShellProps } from './AppShell';
import { ThemeProvider } from './ThemeProvider';

const navigationItems = [
  { id: 'pipelines', label: 'Pipelines', href: '/pipelines', icon: Workflow },
  { id: 'experiments', label: 'Experiments', href: '/experiments', icon: FlaskConical },
  { id: 'runs', label: 'Runs', href: '/runs', icon: PlayCircle },
  { id: 'recurring', label: 'Recurring runs', href: '/recurringruns', icon: Repeat },
  { id: 'artifacts', label: 'Artifacts', href: '/artifacts', icon: Package },
];

function ShellPreview(props: AppShellProps) {
  const { pathname } = useLocation();
  const page = navigationItems.find((item) => item.href === pathname)?.label ?? 'Workspace';
  return (
    <AppShell {...props} currentPath={pathname}>
      <header style={{ padding: '28px 32px 24px' }}>
        <h1 style={{ margin: 0, fontSize: 22, fontWeight: 600, letterSpacing: '-0.02em' }}>
          {page}
        </h1>
        <p style={{ margin: '4px 0 0', color: 'var(--muted-foreground)' }}>
          Build, share, and run machine learning workflows.
        </p>
      </header>
      <section
        aria-label='Workspace'
        style={{
          margin: '0 32px 32px',
          padding: 32,
          border: '1px solid var(--border)',
          borderRadius: 12,
          background: 'var(--card)',
          maxWidth: 760,
        }}
      >
        <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>Your workspace</h2>
        <p style={{ margin: '8px 0 0', color: 'var(--foreground-2)', lineHeight: 1.7 }}>
          Use the navigation to explore pipelines, experiments, runs, and artifacts.
        </p>
      </section>
    </AppShell>
  );
}

const meta = {
  title: 'Modernization/App shell',
  component: AppShell,
  parameters: { layout: 'fullscreen' },
  args: {
    items: navigationItems,
    currentPath: '/runs',
    namespace: 'team-ml',
    version: 'v2.19.0',
    children: null,
  },
  decorators: [
    (Story, context) => (
      <MemoryRouter initialEntries={['/runs']}>
        <ThemeProvider
          defaultTheme={context.parameters.theme === 'dark' ? 'dark' : 'light'}
          storageKey={`kfp.storybook.shell.${context.id}`}
        >
          <Story />
        </ThemeProvider>
      </MemoryRouter>
    ),
  ],
  render: (args) => <ShellPreview {...args} />,
} satisfies Meta<typeof AppShell>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Expanded: Story = {};

export const Dark: Story = {
  parameters: { theme: 'dark' },
};

export const Narrow: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'Use a canvas viewport below 1024 pixels to see automatic collapse. Returning to a wider viewport restores the saved navigation preference.',
      },
    },
  },
};

export const HiddenNavigation: Story = {
  args: { hideSideNav: true },
};
