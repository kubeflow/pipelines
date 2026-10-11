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
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Switch } from './switch';

afterEach(cleanup);

it('supports an accessible controlled value and keyboard changes', async () => {
  function Example() {
    const [checked, setChecked] = useState(false);
    return <Switch aria-label='Enable schedule' checked={checked} onCheckedChange={setChecked} />;
  }
  const user = userEvent.setup();
  render(<Example />);
  const control = screen.getByRole('switch', { name: 'Enable schedule' });
  expect(control).not.toBeChecked();
  await user.tab();
  expect(control).toHaveFocus();
  await user.keyboard(' ');
  expect(control).toBeChecked();
  await user.click(control);
  expect(control).not.toBeChecked();
});

it('honors disabled state without changing a controlled value', async () => {
  const change = vi.fn();
  const user = userEvent.setup();
  render(<Switch aria-label='Enable schedule' checked disabled onCheckedChange={change} />);
  const control = screen.getByRole('switch', { name: 'Enable schedule' });
  expect(control).toHaveAttribute('aria-disabled', 'true');
  await user.click(control);
  expect(change).not.toHaveBeenCalled();
  expect(control).toBeChecked();
});
