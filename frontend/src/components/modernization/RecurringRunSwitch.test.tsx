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

it('sends one enable request while pending and waits for the refreshed server value', async () => {
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
