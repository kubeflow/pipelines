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

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { HelpButton } from './HelpButton';
import { ExternalLink } from './ExternalLink';
import { ThemeProvider } from '../components/modernization/ThemeProvider';

beforeEach(() => {
  localStorage.clear();
  vi.stubGlobal('matchMedia', (media: string) => ({
    media,
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }));
});
afterEach(() => vi.unstubAllGlobals());

function example() {
  render(
    <ThemeProvider defaultTheme='dark'>
      <HelpButton
        label='About service accounts'
        helpText={
          <p>
            Read the{' '}
            <ExternalLink href='https://example.test/permissions'>
              required permissions
            </ExternalLink>
            .
          </p>
        }
      />
    </ThemeProvider>,
  );
}

it('opens rich help by keyboard, exposes its link, and restores focus on Escape', async () => {
  const user = userEvent.setup();
  example();
  const trigger = screen.getByRole('button', { name: 'About service accounts' });
  await user.tab();
  expect(trigger).toHaveFocus();
  await user.keyboard('{Enter}');
  const popup = await screen.findByRole('dialog', { name: 'About service accounts' });
  expect(popup.closest('.kfp-theme')).toHaveClass('dark');
  await waitFor(() =>
    expect(within(popup).getByRole('button', { name: 'Close help' })).toHaveFocus(),
  );
  await user.tab();
  const link = within(popup).getByRole('link', { name: 'required permissions' });
  expect(link).toHaveFocus();
  expect(link).toHaveAttribute('href', 'https://example.test/permissions');
  await user.keyboard('{Escape}');
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  await waitFor(() => expect(trigger).toHaveFocus());
});

it('retains pointer help and an explicit close control', async () => {
  const user = userEvent.setup();
  example();
  await user.hover(screen.getByRole('button', { name: 'About service accounts' }));
  const popup = await screen.findByRole('dialog', { name: 'About service accounts' });
  await user.click(within(popup).getByRole('button', { name: 'Close help' }));
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
});
