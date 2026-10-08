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

import { act, cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Apis } from 'src/lib/Apis';
import { V2beta1RecurringRunStatus as Status } from 'src/apisv2beta1/recurringrun';
import { RecurringRunSwitch } from './RecurringRunSwitch';

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

it('sends one enable request while pending and accepts the refreshed server value', async () => {
  let finish!: () => void;
  const enable = vi.spyOn(Apis.recurringRunServiceApi, 'enableRecurringRun').mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = () => resolve({});
      }),
  );
  const onUpdated = vi.fn().mockResolvedValue(undefined);
  const user = userEvent.setup();
  const view = render(
    <RecurringRunSwitch
      id='schedule/a'
      name='Nightly training'
      status={Status.DISABLED}
      onUpdated={onUpdated}
    />,
  );
  const control = screen.getByRole('switch', { name: 'Enable schedule Nightly training' });
  await user.click(control);
  expect(control).toHaveAttribute('aria-busy', 'true');
  expect(control).toHaveAttribute('aria-disabled', 'true');
  expect(control).not.toBeChecked();
  await user.click(control);
  expect(enable).toHaveBeenCalledTimes(1);
  expect(enable).toHaveBeenCalledWith('schedule/a');
  expect(onUpdated).not.toHaveBeenCalled();
  await act(async () => finish());
  expect(onUpdated).toHaveBeenCalledTimes(1);
  view.rerender(
    <RecurringRunSwitch
      id='schedule/a'
      name='Nightly training'
      status={Status.ENABLED}
      onUpdated={onUpdated}
    />,
  );
  expect(control).toBeChecked();
  expect(control).toHaveAttribute('aria-busy', 'false');
});

it('preserves enabled state on failure and clears the row error after a successful retry', async () => {
  const disable = vi
    .spyOn(Apis.recurringRunServiceApi, 'disableRecurringRun')
    .mockRejectedValueOnce(new Error('Permission denied'))
    .mockResolvedValue({});
  const onUpdated = vi.fn().mockResolvedValue(undefined);
  const user = userEvent.setup();
  render(
    <RecurringRunSwitch
      id='nightly'
      name='Nightly'
      status={Status.ENABLED}
      onUpdated={onUpdated}
    />,
  );
  const control = screen.getByRole('switch', { name: 'Enable schedule Nightly' });
  await user.click(control);
  expect(await screen.findByRole('alert')).toHaveTextContent('Permission denied');
  expect(control).toBeChecked();
  expect(onUpdated).not.toHaveBeenCalled();
  await user.click(control);
  await waitFor(() => expect(onUpdated).toHaveBeenCalledTimes(1));
  expect(disable).toHaveBeenCalledTimes(2);
  expect(disable).toHaveBeenLastCalledWith('nightly');
  expect(screen.queryByRole('alert')).not.toBeInTheDocument();
});

it('does not offer a mutation when the server status is unknown', async () => {
  const enable = vi.spyOn(Apis.recurringRunServiceApi, 'enableRecurringRun');
  const user = userEvent.setup();
  render(<RecurringRunSwitch id='unknown' name='Unknown schedule' onUpdated={vi.fn()} />);
  const control = screen.getByRole('switch', { name: 'Enable schedule Unknown schedule' });
  expect(control).toHaveAttribute('aria-disabled', 'true');
  await user.click(control);
  expect(enable).not.toHaveBeenCalled();
});

it('keeps a confirmed mutation when refresh fails and permits the opposite mutation on recovery', async () => {
  const enable = vi.spyOn(Apis.recurringRunServiceApi, 'enableRecurringRun').mockResolvedValue({});
  const disable = vi
    .spyOn(Apis.recurringRunServiceApi, 'disableRecurringRun')
    .mockResolvedValue({});
  const onUpdated = vi
    .fn()
    .mockRejectedValueOnce(new Error('List unavailable'))
    .mockResolvedValue(undefined);
  const user = userEvent.setup();
  const view = render(
    <RecurringRunSwitch
      id='nightly'
      name='Nightly'
      status={Status.DISABLED}
      onUpdated={onUpdated}
    />,
  );
  const control = screen.getByRole('switch', { name: 'Enable schedule Nightly' });
  await user.click(control);
  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Schedule updated, but the list could not refresh: List unavailable',
  );
  expect(screen.getByRole('alert')).not.toHaveTextContent('Unable to update schedule');
  expect(control).toBeChecked();
  expect(screen.getByText(Status.ENABLED)).toBeInTheDocument();
  // An unrelated parent render still carries the old status after the failed refresh.
  view.rerender(
    <RecurringRunSwitch
      id='nightly'
      name='Nightly'
      status={Status.DISABLED}
      onUpdated={onUpdated}
    />,
  );
  expect(control).toBeChecked();
  await user.click(control);
  await waitFor(() => expect(control).toHaveAttribute('aria-busy', 'false'));
  expect(control).not.toBeChecked();
  expect(enable).toHaveBeenCalledTimes(1);
  expect(disable).toHaveBeenCalledTimes(1);
  expect(onUpdated).toHaveBeenCalledTimes(2);
  expect(screen.queryByRole('alert')).not.toBeInTheDocument();
});

it('shows the confirmed status but serializes clicks while the list refresh is pending', async () => {
  const enable = vi.spyOn(Apis.recurringRunServiceApi, 'enableRecurringRun').mockResolvedValue({});
  const disable = vi
    .spyOn(Apis.recurringRunServiceApi, 'disableRecurringRun')
    .mockResolvedValue({});
  let finishRefresh!: () => void;
  const onUpdated = vi.fn(() => new Promise<void>((resolve) => (finishRefresh = resolve)));
  const user = userEvent.setup();
  render(
    <RecurringRunSwitch
      id='nightly'
      name='Nightly'
      status={Status.DISABLED}
      onUpdated={onUpdated}
    />,
  );
  const control = screen.getByRole('switch', { name: 'Enable schedule Nightly' });
  await user.click(control);
  expect(control).toBeChecked();
  expect(control).toHaveAttribute('aria-busy', 'true');
  await user.click(control);
  expect(enable).toHaveBeenCalledTimes(1);
  expect(disable).not.toHaveBeenCalled();
  await act(async () => finishRefresh());
  expect(control).toHaveAttribute('aria-busy', 'false');
  expect(control).toBeChecked();
});

it('accepts subsequent server status changes after a confirmed update without resurrecting it', async () => {
  vi.spyOn(Apis.recurringRunServiceApi, 'enableRecurringRun').mockResolvedValue({});
  const onUpdated = vi.fn().mockResolvedValue(undefined);
  const user = userEvent.setup();
  const view = render(
    <RecurringRunSwitch
      id='nightly'
      name='Nightly'
      status={Status.DISABLED}
      onUpdated={onUpdated}
    />,
  );
  const control = screen.getByRole('switch', { name: 'Enable schedule Nightly' });
  await user.click(control);
  expect(control).toBeChecked();
  view.rerender(
    <RecurringRunSwitch
      id='nightly'
      name='Nightly'
      status={Status.ENABLED}
      onUpdated={onUpdated}
    />,
  );
  expect(control).toBeChecked();
  view.rerender(
    <RecurringRunSwitch
      id='nightly'
      name='Nightly'
      status={Status.DISABLED}
      onUpdated={onUpdated}
    />,
  );
  expect(control).not.toBeChecked();
});

it('does not apply a late mutation or refresh error to a different schedule', async () => {
  let finishEnable!: () => void;
  vi.spyOn(Apis.recurringRunServiceApi, 'enableRecurringRun').mockImplementation(
    () => new Promise((resolve) => (finishEnable = () => resolve({}))),
  );
  const onUpdated = vi.fn().mockRejectedValue(new Error('Old refresh failed'));
  const user = userEvent.setup();
  const view = render(
    <RecurringRunSwitch id='old' name='Old' status={Status.DISABLED} onUpdated={onUpdated} />,
  );
  await user.click(screen.getByRole('switch', { name: 'Enable schedule Old' }));
  view.rerender(
    <RecurringRunSwitch id='new' name='New' status={Status.DISABLED} onUpdated={onUpdated} />,
  );
  await act(async () => finishEnable());
  expect(screen.getByRole('switch', { name: 'Enable schedule New' })).not.toBeChecked();
  expect(screen.queryByRole('alert')).not.toBeInTheDocument();
});
