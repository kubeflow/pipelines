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

import { createRef } from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TextField } from './text-field';
import { Button } from './button';

it('associates labels, required state, hints and recovering validation with the native input', async () => {
  const ref = createRef<HTMLInputElement>();
  const { rerender } = render(
    <TextField
      label='Run name'
      required
      hint='Choose a unique name.'
      error='A name is required.'
      ref={ref}
    />,
  );
  const input = screen.getByRole('textbox', { name: 'Run name' });
  expect(input).toBeRequired();
  expect(input).toHaveAccessibleDescription('Choose a unique name. A name is required.');
  expect(input).toBeInvalid();
  expect(ref.current).toBe(input);
  await userEvent.type(input, 'training');
  expect(input).toHaveValue('training');
  rerender(<TextField label='Run name' required hint='Choose a unique name.' ref={ref} />);
  expect(input).toBeValid();
  expect(input).toHaveAccessibleDescription('Choose a unique name.');
  expect(screen.queryByRole('alert')).not.toBeInTheDocument();
});

it('keeps a resource picker action reachable beside a read-only value', async () => {
  const choose = vi.fn();
  render(
    <TextField
      label='Pipeline'
      value='Training'
      readOnly
      trailingContent={<Button onClick={choose}>Choose pipeline</Button>}
    />,
  );
  await userEvent.tab();
  expect(screen.getByRole('textbox', { name: 'Pipeline' })).toHaveFocus();
  await userEvent.keyboard('x');
  expect(screen.getByRole('textbox', { name: 'Pipeline' })).toHaveValue('Training');
  await userEvent.tab();
  await userEvent.keyboard('{Enter}');
  expect(choose).toHaveBeenCalledOnce();
});

it('preserves textarea edits, disabled state, custom descriptions and explicit ids', async () => {
  const ref = createRef<HTMLTextAreaElement>();
  const { rerender } = render(
    <>
      <p id='extra'>Optional context.</p>
      <TextField
        multiline
        id='description'
        label='Description'
        aria-describedby='extra'
        defaultValue='First line'
        ref={ref}
      />
    </>,
  );
  const textarea = screen.getByRole('textbox', { name: 'Description' });
  expect(textarea).toHaveAttribute('id', 'description');
  expect(textarea).toHaveAccessibleDescription('Optional context.');
  expect(ref.current).toBe(textarea);
  await userEvent.type(textarea, '{Enter}Second line');
  expect(textarea).toHaveValue('First line\nSecond line');
  rerender(<TextField multiline id='description' label='Description' disabled value='Saved' />);
  expect(screen.getByRole('textbox', { name: 'Description' })).toBeDisabled();
});
