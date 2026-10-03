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

import { useState } from 'react';
import { act, cleanup, fireEvent, render, screen, within, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { InspectionTabs } from './InspectionTabs';
import { InspectionFields } from './InspectionFields';
import { InspectionPanel } from './InspectionPanel';
import { InspectionNotice } from './InspectionNotice';
import { ThemeProvider } from './ThemeProvider';

class ViewportMedia extends EventTarget implements MediaQueryList {
  matches = false;
  media = '(max-width: 899px)';
  onchange: ((event: MediaQueryListEvent) => void) | null = null;
  addListener = vi.fn();
  removeListener = vi.fn();
  setNarrow(narrow: boolean) {
    this.matches = narrow;
    this.dispatchEvent(new Event('change'));
  }
}

let viewport: ViewportMedia;
beforeEach(() => {
  viewport = new ViewportMedia();
  vi.stubGlobal(
    'matchMedia',
    vi.fn(() => viewport),
  );
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

it('associates the active panel and prevents disabled-tab activation with arrow keys', async () => {
  function Example() {
    const [selectedTab, onSwitch] = useState(0);
    return (
      <InspectionTabs
        tabs={['Overview', { label: 'Unavailable', disabled: true }, 'Details']}
        selectedTab={selectedTab}
        onSwitch={onSwitch}
        ariaLabel='Inspection'
      >
        {selectedTab === 0 ? 'Overview content' : 'Details content'}
      </InspectionTabs>
    );
  }
  const user = userEvent.setup();
  render(<Example />);
  expect(screen.getByRole('tablist', { name: 'Inspection' })).toBeVisible();
  const first = screen.getByRole('tab', { name: 'Overview' });
  expect(screen.getByRole('tabpanel', { name: 'Overview' })).toHaveTextContent('Overview content');
  first.focus();
  await user.keyboard('{ArrowRight}');
  expect(screen.getByRole('tab', { name: 'Unavailable' })).toHaveFocus();
  await user.keyboard('{Enter}');
  expect(screen.getByRole('tabpanel', { name: 'Overview' })).toBeVisible();
  await user.keyboard('{ArrowRight}');
  expect(screen.getByRole('tab', { name: 'Details' })).toHaveFocus();
  expect(screen.getByRole('tabpanel', { name: 'Details' })).toHaveTextContent('Details content');
  expect(screen.getByRole('tab', { name: 'Unavailable' })).toHaveAttribute('aria-disabled', 'true');
});

it('requests a pointer tab change once while the controlled parent is still on the old value', async () => {
  const onSwitch = vi.fn();
  const user = userEvent.setup();
  render(
    <InspectionTabs tabs={['Overview', 'Details']} selectedTab={0} onSwitch={onSwitch}>
      Content
    </InspectionTabs>,
  );
  const details = screen.getByRole('tab', { name: 'Details' });
  await user.click(details);
  expect(onSwitch).toHaveBeenCalledExactlyOnceWith(1);
  // A parent may decline a change; a later explicit click must still request it.
  await user.click(details);
  expect(onSwitch.mock.calls).toEqual([[1], [1]]);
});

it('requests one change when touch compatibility mouse events focus a tab after pointerup', () => {
  const onSwitch = vi.fn();
  render(
    <InspectionTabs tabs={['Overview', 'Details']} selectedTab={0} onSwitch={onSwitch}>
      Content
    </InspectionTabs>,
  );
  const details = screen.getByRole('tab', { name: 'Details' });
  fireEvent.pointerDown(details, { button: 0, pointerType: 'touch' });
  fireEvent.pointerUp(details, { button: 0, pointerType: 'touch' });
  fireEvent.mouseDown(details, { button: 0 });
  act(() => details.focus());
  fireEvent.mouseUp(details, { button: 0 });
  fireEvent.click(details);
  expect(onSwitch).toHaveBeenCalledExactlyOnceWith(1);
});

it('accepts controlled Back and Forward selections without blocking a later tab gesture', async () => {
  const onSwitch = vi.fn();
  const user = userEvent.setup();
  const content = (selectedTab: number) => (
    <InspectionTabs tabs={['Overview', 'Details']} selectedTab={selectedTab} onSwitch={onSwitch}>
      Content
    </InspectionTabs>
  );
  const { rerender } = render(content(0));
  await user.click(screen.getByRole('tab', { name: 'Details' }));
  expect(onSwitch).toHaveBeenCalledExactlyOnceWith(1);
  rerender(content(1));
  expect(screen.getByRole('tab', { name: 'Details' })).toHaveAttribute('aria-selected', 'true');
  rerender(content(0));
  expect(screen.getByRole('tab', { name: 'Overview' })).toHaveAttribute('aria-selected', 'true');
  rerender(content(1));
  await user.click(screen.getByRole('tab', { name: 'Overview' }));
  expect(onSwitch.mock.calls).toEqual([[1], [0]]);
});

it('preserves falsy fields, structured values, and custom field content', () => {
  render(
    <>
      <InspectionFields
        title='Parameters'
        fields={[
          ['Enabled', false],
          ['Count', 0],
          ['Config', '{"items":[1,2]}'],
        ]}
      />
      <InspectionFields
        title='Artifacts'
        fields={[['Model', { uri: 's3://model' }]]}
        valueComponent={({ value }) =>
          typeof value === 'object' ? <a href={value.uri}>Artifact URI</a> : <span>{value}</span>
        }
      />
    </>,
  );
  const parameters = within(screen.getByRole('region', { name: 'Parameters' }));
  expect(parameters.getByText('false')).toBeVisible();
  expect(parameters.getByText('0')).toBeVisible();
  expect(parameters.getByText(/"items"/).tagName).toBe('PRE');
  expect(screen.getByRole('link', { name: 'Artifact URI' })).toHaveAttribute('href', 's3://model');
});

function PanelExample() {
  const [open, setOpen] = useState(false);
  return (
    <div className='kfp-theme dark'>
      <button onClick={() => setOpen(true)}>Inspect task</button>
      <button>Canvas action</button>
      <InspectionPanel isOpen={open} title='Train model' onClose={() => setOpen(false)}>
        <button>Task action</button>
      </InspectionPanel>
    </div>
  );
}

it('keeps desktop inspector modeless, themed, and restores focus after Escape', async () => {
  const user = userEvent.setup();
  render(<PanelExample />);
  await user.click(screen.getByRole('button', { name: 'Inspect task' }));
  const dialog = await screen.findByRole('dialog', { name: 'Train model' });
  expect(dialog).not.toHaveAttribute('aria-modal', 'true');
  expect(dialog.closest('.kfp-theme')).toHaveClass('dark');
  await waitFor(() => expect(screen.getByRole('button', { name: 'close' })).toHaveFocus());
  await user.keyboard('{Escape}');
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  expect(screen.getByRole('button', { name: 'Inspect task' })).toHaveFocus();
});

it('uses a modal inspector on narrow screens and keeps task content on resize', async () => {
  viewport.setNarrow(true);
  const user = userEvent.setup();
  render(<PanelExample />);
  await user.click(screen.getByRole('button', { name: 'Inspect task' }));
  expect(await screen.findByRole('dialog', { name: 'Train model' })).toHaveAttribute(
    'aria-modal',
    'true',
  );
  await waitFor(() => expect(screen.getByRole('button', { name: 'close' })).toHaveFocus());
  await user.tab({ shift: true });
  await waitFor(() => expect(screen.getByRole('button', { name: 'Task action' })).toHaveFocus());
  act(() => viewport.setNarrow(false));
  expect(screen.getByRole('dialog', { name: 'Train model' })).not.toHaveAttribute(
    'aria-modal',
    'true',
  );
  expect(screen.getByRole('button', { name: 'Task action' })).toBeVisible();
  await user.click(screen.getByRole('button', { name: 'close' }));
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
});

it('resizes the desktop inspector with a named keyboard separator and preserves width on rerender', async () => {
  vi.stubGlobal('innerWidth', 1200);
  const close = vi.fn();
  const content = (title: string) => (
    <InspectionPanel isOpen title={title} onClose={close}>
      <button>Task action</button>
    </InspectionPanel>
  );
  const user = userEvent.setup();
  const view = render(content('Train model'));
  const handle = screen.getByRole('separator', { name: 'Resize node details' });
  const dialog = document.getElementById(handle.getAttribute('aria-controls')!)!;
  expect(dialog).toHaveAttribute('role', 'dialog');
  const panel = dialog.querySelector('.kfp-inspector-panel')!;
  expect(handle).toHaveAttribute('aria-orientation', 'vertical');
  expect(handle).toHaveAttribute('aria-valuenow', '380');
  expect(handle).toHaveAttribute('aria-valuemin', '300');
  expect(handle).toHaveAttribute('aria-valuemax', '1080');
  await waitFor(() => expect(screen.getByRole('button', { name: 'close' })).toHaveFocus());
  handle.focus();
  await user.keyboard('{ArrowLeft}');
  expect(handle).toHaveAttribute('aria-valuenow', '400');
  expect(panel).toHaveStyle({ width: '400px' });
  await user.keyboard('{ArrowRight}');
  expect(handle).toHaveAttribute('aria-valuenow', '380');
  await user.keyboard('{Home}{ArrowRight}');
  expect(handle).toHaveAttribute('aria-valuenow', '300');
  await user.keyboard('{End}{ArrowLeft}');
  expect(handle).toHaveAttribute('aria-valuenow', '1080');
  expect(handle).toHaveAttribute('aria-valuetext', '1080 pixels');
  expect(panel).toHaveStyle({ width: '1080px' });
  view.rerender(content('Updated task'));
  expect(handle).toHaveFocus();
  expect(handle).toHaveAttribute('aria-valuenow', '1080');
  expect(panel).toHaveStyle({ width: '1080px' });
  await user.keyboard('{Escape}');
  expect(close).toHaveBeenCalledOnce();
});

it('bounds inspector width to the viewport without discarding the chosen desktop width', async () => {
  vi.stubGlobal('innerWidth', 1200);
  const user = userEvent.setup();
  render(<PanelExample />);
  await user.click(screen.getByRole('button', { name: 'Inspect task' }));
  const handle = screen.getByRole('separator', { name: 'Resize node details' });
  await waitFor(() => expect(screen.getByRole('button', { name: 'close' })).toHaveFocus());
  handle.focus();
  await user.keyboard('{End}');
  act(() => {
    vi.stubGlobal('innerWidth', 1000);
    window.dispatchEvent(new Event('resize'));
  });
  expect(handle).toHaveAttribute('aria-valuemax', '900');
  expect(handle).toHaveAttribute('aria-valuenow', '900');
  act(() => viewport.setNarrow(true));
  expect(screen.queryByRole('separator')).not.toBeInTheDocument();
  expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true');
  await waitFor(() => expect(screen.getByRole('button', { name: 'close' })).toHaveFocus());
  act(() => {
    vi.stubGlobal('innerWidth', 1200);
    window.dispatchEvent(new Event('resize'));
    viewport.setNarrow(false);
  });
  expect(screen.getByRole('separator', { name: 'Resize node details' })).toHaveAttribute(
    'aria-valuenow',
    '1080',
  );
});

it('keeps error details readable in a themed dismissible dialog', async () => {
  const user = userEvent.setup();
  render(
    <ThemeProvider defaultTheme='dark' storageKey='inspection-notice-test'>
      <InspectionNotice
        mode='warning'
        message='Refresh failed'
        additionalInfo='Service unavailable'
      />
    </ThemeProvider>,
  );
  expect(screen.getByRole('alert')).toHaveTextContent('Refresh failed');
  await user.click(screen.getByRole('button', { name: 'Details' }));
  const dialog = screen.getByRole('dialog', { name: 'Warning' });
  expect(dialog.closest('.kfp-theme')).toHaveClass('dark');
  expect(within(dialog).getByText('Service unavailable')).toBeVisible();
  await user.click(within(dialog).getByRole('button', { name: 'Dismiss' }));
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
});

it('retains pointer deltas when DOM measurements lag and starts each drag from the current width', async () => {
  const user = userEvent.setup();
  const view = render(
    <InspectionPanel isOpen title='Train model' onClose={vi.fn()}>
      Task
    </InspectionPanel>,
  );
  const handle = screen.getByRole('separator', { name: 'Resize node details' });
  const panel = document.querySelector('.kfp-inspector-panel')!;
  let measuredWidth = 380;
  Object.defineProperty(panel, 'offsetWidth', { configurable: true, get: () => measuredWidth });
  await waitFor(() => expect(screen.getByRole('button', { name: 'close' })).toHaveFocus());
  fireEvent.mouseDown(handle, { clientX: 100, clientY: 20 });
  // The real browser can still report the prior width after the final pointer move.
  fireEvent.mouseMove(window, { clientX: 60, clientY: 20 });
  expect(handle).toHaveAttribute('aria-valuenow', '420');
  fireEvent.mouseUp(window, { clientX: 60, clientY: 20 });
  expect(handle).toHaveAttribute('aria-valuenow', '420');
  expect(panel).toHaveStyle({ width: '420px' });
  view.rerender(
    <InspectionPanel isOpen title='Updated task' onClose={vi.fn()}>
      Updated task
    </InspectionPanel>,
  );
  expect(handle).toHaveAttribute('aria-valuenow', '420');
  handle.focus();
  await user.keyboard('{ArrowRight}');
  expect(handle).toHaveAttribute('aria-valuenow', '400');
  measuredWidth = 400;
  fireEvent.mouseDown(handle, { clientX: 100, clientY: 20 });
  fireEvent.mouseMove(window, { clientX: 120, clientY: 20 });
  fireEvent.mouseUp(window, { clientX: 120, clientY: 20 });
  expect(handle).toHaveAttribute('aria-valuenow', '380');
  expect(panel).toHaveStyle({ width: '380px' });
  measuredWidth = 380;
  // re-resizable retains its previous delta when a later gesture does not move.
  fireEvent.mouseDown(handle, { clientX: 100, clientY: 20 });
  fireEvent.mouseUp(window, { clientX: 100, clientY: 20 });
  expect(handle).toHaveAttribute('aria-valuenow', '380');
  expect(panel).toHaveStyle({ width: '380px' });
});
