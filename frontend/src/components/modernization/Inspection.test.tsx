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
