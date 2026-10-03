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
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ModalDialog } from './dialog';
import { Button } from './button';
import { TextField } from './text-field';
import { ThemeProvider } from '../modernization/ThemeProvider';

beforeEach(() => {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }));
});

afterEach(() => vi.unstubAllGlobals());

it('keeps a simple confirmation message associated with the dialog', () => {
  render(
    <ThemeProvider defaultTheme='light'>
      <ModalDialog open title='Archive run?' onClose={vi.fn()}>
        The run will move to the archive.
      </ModalDialog>
    </ThemeProvider>,
  );
  expect(screen.getByRole('dialog', { name: 'Archive run?' })).toHaveAccessibleDescription(
    'The run will move to the archive.',
  );
});

it('keeps rich content separately navigable and preserves keyboard trapping and focus return', async () => {
  function Example() {
    const [open, setOpen] = useState(false);
    return (
      <ThemeProvider defaultTheme='dark'>
        <Button onClick={() => setOpen(true)}>Choose resource</Button>
        <ModalDialog
          open={open}
          title='Choose a resource'
          onClose={() => setOpen(false)}
          actions={<Button onClick={() => setOpen(false)}>Cancel</Button>}
        >
          <TextField label='Resource name' hint='Filter the available resources.' />
          <table>
            <caption>Available resources</caption>
            <thead>
              <tr>
                <th>Name</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>Training pipeline</td>
              </tr>
            </tbody>
          </table>
        </ModalDialog>
      </ThemeProvider>
    );
  }
  const user = userEvent.setup();
  render(<Example />);
  const trigger = screen.getByRole('button', { name: 'Choose resource' });
  await user.click(trigger);
  const dialog = screen.getByRole('dialog', { name: 'Choose a resource' });
  expect(dialog).not.toHaveAttribute('aria-describedby');
  expect(dialog).toHaveAccessibleDescription('');
  const field = within(dialog).getByRole('textbox', { name: 'Resource name' });
  expect(field).toHaveAccessibleDescription('Filter the available resources.');
  expect(within(dialog).getByRole('table', { name: 'Available resources' })).toBeVisible();
  await waitFor(() => expect(field).toHaveFocus());
  await user.tab({ shift: true });
  await waitFor(() => expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus());
  await user.tab();
  await waitFor(() => expect(field).toHaveFocus());
  await user.keyboard('{Escape}');
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  expect(trigger).toHaveFocus();
});
