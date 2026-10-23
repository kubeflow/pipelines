/*
 * Copyright 2018 The Kubeflow Authors
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

import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FeatureKey, getFeatureList, initFeatures } from 'src/features';
import FrontendFeatures from './FrontendFeatures';

beforeEach(() => {
  localStorage.setItem(
    'flags',
    JSON.stringify([{ name: FeatureKey.FUNCTIONAL_COMPONENT, active: false }]),
  );
  initFeatures();
});
afterEach(() => vi.restoreAllMocks());

it('edits an unsaved keyboard-accessible draft and resets without persisting it', async () => {
  const writes = vi.spyOn(localStorage, 'setItem');
  const user = userEvent.setup();
  render(<FrontendFeatures />);
  const control = screen.getByRole('switch', { name: 'Enable functional_component' });
  expect(control).not.toBeChecked();
  control.focus();
  await user.keyboard(' ');
  expect(control).toBeChecked();
  expect(getFeatureList()[0].active).toBe(false);
  expect(writes).not.toHaveBeenCalled();
  await user.click(screen.getByRole('button', { name: 'Reset' }));
  expect(control).not.toBeChecked();
  expect(writes).not.toHaveBeenCalled();
});

it('saves exactly once, then resets later edits to the saved state', () => {
  const writes = vi.spyOn(localStorage, 'setItem');
  render(<FrontendFeatures />);
  const control = screen.getByRole('switch', { name: 'Enable functional_component' });
  fireEvent.click(control);
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }));
  expect(getFeatureList()[0].active).toBe(true);
  // saveFeatures checks storage availability before the one actual preference write.
  expect(writes.mock.calls.filter(([key]) => key === 'flags')).toHaveLength(1);
  fireEvent.click(control);
  expect(control).not.toBeChecked();
  fireEvent.click(screen.getByRole('button', { name: 'Reset' }));
  expect(control).toBeChecked();
  expect(getFeatureList()[0].active).toBe(true);
});
