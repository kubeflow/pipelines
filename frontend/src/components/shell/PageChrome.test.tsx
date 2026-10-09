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

import { act, fireEvent, render, screen, within, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { useState } from 'react';
import { PageChrome, Toolbar } from './PageChrome';
import type { PageChromeProps } from './PageChrome';
import { ThemeProvider } from './ThemeProvider';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import type { PageProps } from 'src/pages/Page';
import {
  Archive,
  ArrowUpFromLine,
  ChevronsDownUp,
  ChevronsUpDown,
  Copy,
  GitCompareArrows,
  Pause,
  Play,
  Plus,
  Repeat,
  RefreshCw,
  RotateCcw,
  Square,
  Trash2,
  Upload,
} from 'lucide-react';

const defaults: PageChromeProps = {
  children: <p>Workspace content</p>,
  toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: 'Runs' },
  bannerProps: {},
  dialogProps: { open: false },
  snackbarProps: { open: false },
  onDialogClose: vi.fn(),
  onSnackbarClose: vi.fn(),
};

function element(overrides: Partial<PageChromeProps> = {}) {
  return (
    <MemoryRouter>
      <ThemeProvider defaultTheme='light'>
        <PageChrome {...defaults} {...overrides} />
      </ThemeProvider>
    </MemoryRouter>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  localStorage.clear();
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
  localStorage.clear();
});

describe('PageChrome', () => {
  it('reserves an inert top-level header while breadcrumb metadata loads', () => {
    const emptyToolbar = { actions: {}, breadcrumbs: [], pageTitle: '' };
    const view = render(element({ toolbarProps: emptyToolbar, reserveBreadcrumbHeader: true }));
    const header = view.container.querySelector('.kfp-page-header');
    expect(header).toHaveAttribute('data-reserve-breadcrumb-header', 'true');
    expect(header).toHaveAttribute('aria-hidden', 'true');
    expect(screen.queryByRole('navigation', { name: 'Breadcrumbs' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
    view.rerender(
      element({
        reserveBreadcrumbHeader: true,
        toolbarProps: {
          ...emptyToolbar,
          pageTitle: 'Run details',
          breadcrumbs: [{ displayName: 'Runs', href: '/runs' }],
        },
      }),
    );
    expect(view.container.querySelector('.kfp-page-header')).toBe(header);
    expect(header).not.toHaveAttribute('aria-hidden');
    expect(screen.getByRole('link', { name: 'Runs' })).toHaveAttribute('href', '/runs');
    expect(screen.getByRole('heading', { name: 'Run details' })).toBeVisible();
  });

  it('does not reserve empty ordinary or embedded toolbars', () => {
    const toolbarProps = { actions: {}, breadcrumbs: [], pageTitle: '' };
    const view = render(element({ toolbarProps }));
    expect(view.container.querySelector('.kfp-page-header')).toBeNull();
    view.rerender(
      element({
        reserveBreadcrumbHeader: true,
        toolbarProps: { ...toolbarProps, topLevelToolbar: false },
      }),
    );
    expect(view.container.querySelector('.kfp-page-header')).toBeNull();
    view.rerender(
      <MemoryRouter>
        <Toolbar {...toolbarProps} topLevelToolbar={false} />
      </MemoryRouter>,
    );
    expect(view.container.querySelector('.kfp-page-header')).toBeNull();
  });

  it('retains action order, IDs, eligibility, busy state and callbacks', async () => {
    const refresh = vi.fn();
    render(
      element({
        toolbarProps: {
          breadcrumbs: [],
          pageTitle: 'Runs',
          actions: {
            newRun: {
              title: 'Create run',
              tooltip: 'Create a new run',
              id: 'createNewRunBtn',
              action: vi.fn(),
              variant: 'default',
            },
            compare: {
              title: 'Compare',
              tooltip: 'Compare runs',
              disabledTitle: 'Select 2–10 runs',
              action: vi.fn(),
              disabled: true,
            },
            archive: {
              title: 'Archive',
              tooltip: 'Archive selected runs',
              action: vi.fn(),
              busy: true,
            },
            refresh: { title: 'Refresh', tooltip: 'Reload runs', action: refresh },
          },
        },
      }),
    );
    expect(screen.getByRole('heading', { name: 'Runs', level: 1 })).toBeVisible();
    const buttons = screen.getAllByRole('button');
    expect(buttons.map((button) => button.textContent)).toEqual([
      'Create run',
      'Compare',
      'Archive',
      'Refresh',
    ]);
    expect(buttons[0]).toHaveAttribute('id', 'createNewRunBtn');
    expect(buttons[1]).toBeDisabled();
    expect(buttons[1].parentElement).toHaveAttribute('title', 'Select 2–10 runs');
    expect(buttons[2]).toBeDisabled();
    expect(buttons[2]).toHaveAttribute('aria-busy', 'true');
    await userEvent.click(buttons[3]);
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it('forwards one confirmation action to its owner without invoking it separately', async () => {
    const confirm = vi.fn();
    const close = vi.fn();
    render(
      element({
        dialogProps: {
          open: true,
          title: 'Archive 2 runs?',
          content: 'Selected runs will move to the archive.',
          buttons: [{ text: 'Cancel' }, { text: 'Archive', onClick: confirm }],
        },
        onDialogClose: close,
      }),
    );
    const dialog = screen.getByRole('dialog', { name: 'Archive 2 runs?' });
    expect(dialog).toHaveAccessibleDescription('Selected runs will move to the archive.');
    await userEvent.click(within(dialog).getByRole('button', { name: 'Archive' }));
    expect(close).toHaveBeenCalledExactlyOnceWith(confirm);
    expect(confirm).not.toHaveBeenCalled();
  });

  it('traps keyboard focus in confirmation and restores it to the invoking action', async () => {
    function Workspace() {
      const [open, setOpen] = useState(false);
      return (
        <PageChrome
          {...defaults}
          toolbarProps={{
            actions: {
              archive: {
                title: 'Archive',
                tooltip: 'Archive selected runs',
                action: () => setOpen(true),
              },
            },
            breadcrumbs: [],
            pageTitle: 'Runs',
          }}
          dialogProps={{
            open,
            title: 'Archive 1 run?',
            buttons: [{ text: 'Cancel' }, { text: 'Confirm' }],
          }}
          onDialogClose={() => setOpen(false)}
        />
      );
    }
    render(
      <MemoryRouter>
        <ThemeProvider defaultTheme='light'>
          <Workspace />
        </ThemeProvider>
      </MemoryRouter>,
    );
    const user = userEvent.setup();
    const trigger = screen.getByRole('button', { name: 'Archive' });
    await user.click(trigger);
    const dialog = screen.getByRole('dialog');
    await waitFor(() =>
      expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus(),
    );
    await user.tab({ shift: true });
    await waitFor(() =>
      expect(within(dialog).getByRole('button', { name: 'Confirm' })).toHaveFocus(),
    );
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    await waitFor(() => expect(trigger).toHaveFocus());
  });

  it('dismisses a confirmation through Escape without selecting an action', async () => {
    const close = vi.fn();
    render(
      element({
        dialogProps: {
          open: true,
          title: 'Delete 1 run?',
          buttons: [{ text: 'Cancel' }, { text: 'Delete' }],
        },
        onDialogClose: close,
      }),
    );
    await userEvent.keyboard('{Escape}');
    expect(close).toHaveBeenCalledExactlyOnceWith();
  });

  it('keeps portaled dialogs in the active palette and exposes theme in embedded layouts', async () => {
    localStorage.setItem('kfp.theme', 'dark');
    const view = render(element({ showThemeControl: true }));
    expect(screen.getByRole('combobox', { name: 'Theme' })).toHaveValue('dark');
    view.rerender(
      element({
        showThemeControl: true,
        dialogProps: { open: true, title: 'Archive 1 run?', buttons: [{ text: 'Cancel' }] },
      }),
    );
    expect(screen.getByRole('dialog').closest('.kfp-theme')).toHaveClass('dark');
    view.rerender(element({ showThemeControl: true }));
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Theme' }), 'light');
    expect(localStorage.getItem('kfp.theme')).toBe('light');
  });

  it('keeps error details, troubleshooting and refresh available and clears recovered errors', async () => {
    const refresh = vi.fn();
    const view = render(
      element({
        bannerProps: {
          message: 'Unable to load runs.',
          mode: 'error',
          additionalInfo: '403: Access denied.',
          refresh,
        },
      }),
    );
    const alert = screen.getByRole('alert');
    expect(within(alert).getByRole('link', { name: 'Troubleshooting guide' })).toHaveAttribute(
      'href',
      'https://www.kubeflow.org/docs/pipelines/troubleshooting',
    );
    await userEvent.click(within(alert).getByRole('button', { name: 'Refresh' }));
    expect(refresh).toHaveBeenCalledTimes(1);
    await userEvent.click(within(alert).getByRole('button', { name: 'Details' }));
    expect(screen.getByRole('dialog', { name: 'An error occurred' })).toHaveTextContent(
      '403: Access denied.',
    );
    await userEvent.click(screen.getByRole('button', { name: 'Dismiss' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    view.rerender(element());
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('does not invent a refresh action for an informational notice', () => {
    render(
      element({
        bannerProps: { message: 'An informational notice.', mode: 'info' },
      }),
    );
    expect(screen.getByRole('status')).toHaveTextContent('An informational notice.');
    expect(screen.queryByRole('button', { name: 'Refresh' })).not.toBeInTheDocument();
  });

  it('does not reset notification dismissal on unrelated renders and uses the latest callback', () => {
    vi.useFakeTimers();
    const first = vi.fn();
    const latest = vi.fn();
    const snackbarProps = {
      open: true,
      message: 'Archive succeeded for 1 run',
      autoHideDuration: 5000,
    };
    const view = render(element({ snackbarProps, onSnackbarClose: first }));
    act(() => vi.advanceTimersByTime(3000));
    view.rerender(
      element({ snackbarProps, onSnackbarClose: latest, children: <p>Refreshed rows</p> }),
    );
    act(() => vi.advanceTimersByTime(2000));
    expect(first).not.toHaveBeenCalled();
    expect(latest).toHaveBeenCalledTimes(1);
  });

  it('keeps notifications paused while focus remains after the pointer leaves', () => {
    vi.useFakeTimers();
    const close = vi.fn();
    render(
      element({
        snackbarProps: { open: true, message: 'Updated runs', autoHideDuration: 5000 },
        onSnackbarClose: close,
      }),
    );
    const notice = screen.getByRole('status');
    fireEvent.mouseEnter(notice);
    fireEvent.focus(screen.getByRole('button', { name: 'Dismiss notification' }));
    fireEvent.mouseLeave(notice);
    act(() => vi.advanceTimersByTime(6000));
    expect(close).not.toHaveBeenCalled();
  });

  it('pauses in another browser window and resumes on return', () => {
    vi.useFakeTimers();
    const close = vi.fn();
    render(
      element({
        snackbarProps: { open: true, message: 'Updated runs', autoHideDuration: 5000 },
        onSnackbarClose: close,
      }),
    );
    act(() => vi.advanceTimersByTime(1000));
    vi.mocked(document.hasFocus).mockReturnValue(false);
    fireEvent(window, new Event('blur'));
    act(() => vi.advanceTimersByTime(6000));
    expect(close).not.toHaveBeenCalled();
    vi.mocked(document.hasFocus).mockReturnValue(true);
    fireEvent(window, new Event('focus'));
    act(() => vi.advanceTimersByTime(5000));
    expect(close).toHaveBeenCalledTimes(1);
  });

  it('pauses notification dismissal while focused and offers an explicit close action', () => {
    vi.useFakeTimers();
    const close = vi.fn();
    render(
      element({
        snackbarProps: { open: true, message: 'Restored 1 run', autoHideDuration: 5000 },
        onSnackbarClose: close,
      }),
    );
    const button = screen.getByRole('button', { name: 'Dismiss notification' });
    fireEvent.focus(button);
    act(() => vi.advanceTimersByTime(6000));
    expect(close).not.toHaveBeenCalled();
    fireEvent.click(button);
    expect(close).toHaveBeenCalledTimes(1);
  });
});

function factoryActions() {
  const navigate = vi.fn();
  const props: PageProps = {
    location: { pathname: '/runs', search: '', hash: '', state: null, key: 'test' },
    navigate,
    params: {},
    toolbarProps: defaults.toolbarProps,
    updateBanner: vi.fn(),
    updateDialog: vi.fn(),
    updateSnackbar: vi.fn(),
    updateToolbar: vi.fn(),
  };
  const callback = vi.fn();
  const refresh = vi.fn();
  const selected = () => ['run-one', 'run-two'];
  const factory = new Buttons(props, refresh)
    .archive('run', selected, false, callback)
    .cloneRun(selected, false)
    .cloneRecurringRun(selected, false)
    .retryRun(selected, false, callback)
    .collapseSections(callback)
    .compareRuns(selected)
    .delete(selected, 'run', callback, false)
    .disableRecurringRun(() => 'schedule')
    .enableRecurringRun(() => 'schedule')
    .expandSections(callback)
    .newExperiment()
    .newPipelineVersion('Upload version')
    .newRun()
    .newRecurringRun('experiment')
    .newRunFromPipelineVersion(
      () => 'pipeline',
      () => 'version',
    )
    .refresh(refresh)
    .restore('run', selected, false, callback)
    .terminateRun(selected, false, callback)
    .upload(callback);
  return { factory, actions: factory.getToolbarActionMap(), refresh, navigate, props };
}

// Explicit expectations validate the factory-to-renderer boundary, not an independent mock map.
const actionContracts = [
  [ButtonKeys.ARCHIVE, Archive, 'secondary'],
  [ButtonKeys.CLONE_RUN, Copy, 'secondary'],
  [ButtonKeys.CLONE_RECURRING_RUN, Copy, 'secondary'],
  [ButtonKeys.RETRY, RotateCcw, 'secondary'],
  [ButtonKeys.COLLAPSE, ChevronsDownUp, 'secondary'],
  [ButtonKeys.COMPARE, GitCompareArrows, 'secondary'],
  [ButtonKeys.DELETE_RUN, Trash2, 'secondary'],
  [ButtonKeys.DISABLE_RECURRING_RUN, Pause, 'secondary'],
  [ButtonKeys.ENABLE_RECURRING_RUN, Play, 'secondary'],
  [ButtonKeys.EXPAND, ChevronsUpDown, 'secondary'],
  [ButtonKeys.NEW_EXPERIMENT, Plus, 'default'],
  [ButtonKeys.NEW_PIPELINE_VERSION, Upload, 'secondary'],
  [ButtonKeys.NEW_RUN, Plus, 'default'],
  [ButtonKeys.NEW_RECURRING_RUN, Repeat, 'secondary'],
  [ButtonKeys.NEW_RUN_FROM_PIPELINE_VERSION, Plus, 'default'],
  [ButtonKeys.REFRESH, RefreshCw, 'secondary'],
  [ButtonKeys.RESTORE, ArrowUpFromLine, 'secondary'],
  [ButtonKeys.TERMINATE_RUN, Square, 'secondary'],
  [ButtonKeys.UPLOAD_PIPELINE, Upload, 'secondary'],
] as const;

describe('Buttons factory → page chrome contract', () => {
  it('covers every public action key', () => {
    expect(actionContracts.map(([key]) => key).sort()).toEqual(Object.values(ButtonKeys).sort());
  });

  it.each(actionContracts)(
    'renders %s with its declared icon and variant',
    (key, Icon, variant) => {
      const { actions } = factoryActions();
      const action = actions[key];
      expect(action.icon).toBe(Icon);
      render(element({ toolbarProps: { ...defaults.toolbarProps, actions: { [key]: action } } }));
      const button = screen.getByRole('button', { name: action.title });
      expect(button.id).toBe(action.id);
      expect(button).toHaveClass(variant === 'default' ? 'bg-primary' : 'bg-card');
      expect(button.querySelector('svg')).toBeInTheDocument();
      expect(button).toHaveAttribute('aria-description', action.disabledTitle || action.tooltip);
      if (action.disabled) expect(button).toBeDisabled();
      else expect(button).toBeEnabled();
      if (action.style?.minWidth)
        expect(button.parentElement).toHaveStyle({ minWidth: action.style.minWidth });
    },
  );

  it('keeps a primary recurring creation variant and both version deletion builders iconized', () => {
    const { factory } = factoryActions();
    factory.newRecurringRunPrimary('experiment');
    expect(factory.getToolbarActionMap()[ButtonKeys.NEW_RECURRING_RUN]).toMatchObject({
      icon: Repeat,
      variant: 'default',
    });
    factory.deletePipelineVersion(() => new Map(), vi.fn(), false);
    expect(factory.getToolbarActionMap()[ButtonKeys.DELETE_RUN].icon).toBe(Trash2);
    factory.deletePipelinesAndPipelineVersions(
      () => [],
      () => ({}),
      vi.fn(),
      false,
    );
    expect(factory.getToolbarActionMap()[ButtonKeys.DELETE_RUN].icon).toBe(Trash2);
  });

  it('uses the supplied icon even for a recognized key and suppresses actions while busy', () => {
    const { actions, refresh } = factoryActions();
    const action = { ...actions[ButtonKeys.REFRESH], icon: Archive };
    const view = render(
      element({ toolbarProps: { ...defaults.toolbarProps, actions: { refresh: action } } }),
    );
    const button = screen.getByRole('button', { name: 'Refresh' });
    expect(button.querySelector('.lucide-archive')).toBeInTheDocument();
    fireEvent.click(button);
    expect(refresh).toHaveBeenCalledTimes(1);
    view.rerender(
      element({
        toolbarProps: { ...defaults.toolbarProps, actions: { refresh: { ...action, busy: true } } },
      }),
    );
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute('aria-busy', 'true');
    expect(button.querySelector('.kfp-page-spinner')).toBeInTheDocument();
    fireEvent.click(button);
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it.each([
    ['error', undefined, true],
    ['error', false, false],
    ['warning', true, true],
    ['info', false, false],
  ] as const)(
    'honors troubleshooting visibility for %s with override %s',
    (mode, showTroubleshootingGuideLink, visible) => {
      render(element({ bannerProps: { message: 'Status', mode, showTroubleshootingGuideLink } }));
      expect(Boolean(screen.queryByRole('link', { name: 'Troubleshooting guide' }))).toBe(visible);
    },
  );
});

it('renders the real archive confirmation contract without executing the mutation on dismissal', async () => {
  const { factory, props } = factoryActions();
  const callback = vi.fn();
  factory.archive('run', () => ['run-one'], true, callback);
  const action = factory.getToolbarActionMap()[ButtonKeys.ARCHIVE];
  const close = vi.fn();
  const view = render(
    element({ toolbarProps: { ...defaults.toolbarProps, actions: { archive: action } } }),
  );
  fireEvent.click(screen.getByRole('button', { name: 'Archive' }));
  const dialogProps = vi.mocked(props.updateDialog).mock.calls[0][0];
  view.rerender(element({ dialogProps, onDialogClose: close }));
  const dialog = screen.getByRole('dialog', { name: 'Archive this run?' });
  expect(within(dialog).getByText(/will be moved to the Archive section/)).toBeVisible();
  fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));
  expect(close).toHaveBeenCalledWith(dialogProps.buttons![0].onClick);
  await dialogProps.buttons![0].onClick!();
  expect(callback).not.toHaveBeenCalled();
});

it('exposes a caller-provided refresh action for an informational banner', () => {
  const refresh = vi.fn();
  render(element({ bannerProps: { mode: 'info', message: 'Waiting for data', refresh } }));
  fireEvent.click(within(screen.getByRole('status')).getByRole('button', { name: 'Refresh' }));
  expect(refresh).toHaveBeenCalledOnce();
});
