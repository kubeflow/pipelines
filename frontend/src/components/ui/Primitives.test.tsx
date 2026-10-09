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
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from './button';
import { Input } from './input';

describe('Modernization primitives', () => {
  it('keeps ordinary actions from submitting a surrounding form', async () => {
    const submit = vi.fn((event) => event.preventDefault());
    const action = vi.fn();
    render(
      <form onSubmit={submit}>
        <Button onClick={action}>Preview</Button>
      </form>,
    );
    await userEvent.click(screen.getByRole('button', { name: 'Preview' }));
    expect(action).toHaveBeenCalledOnce();
    expect(submit).not.toHaveBeenCalled();
  });

  it('supports explicit keyboard form submission and prevents disabled actions', async () => {
    const submit = vi.fn((event) => event.preventDefault());
    const disabledAction = vi.fn();
    render(
      <form onSubmit={submit}>
        <Button type='submit'>Save</Button>
        <Button disabled onClick={disabledAction}>
          Unavailable
        </Button>
      </form>,
    );
    screen.getByRole('button', { name: 'Save' }).focus();
    await userEvent.keyboard('{Enter}');
    expect(submit).toHaveBeenCalledOnce();
    await userEvent.click(screen.getByRole('button', { name: 'Unavailable' }));
    expect(disabledAction).not.toHaveBeenCalled();
  });

  it('forwards focus and caller styling to the real button', () => {
    const ref = createRef<HTMLButtonElement>();
    render(
      <Button ref={ref} className='h-12'>
        Focus
      </Button>,
    );
    ref.current?.focus();
    expect(screen.getByRole('button')).toHaveFocus();
    expect(screen.getByRole('button')).toHaveClass('h-12');
    expect(screen.getByRole('button')).not.toHaveClass('h-[34px]');
  });

  it('retains input labeling, validation description and controlled changes', () => {
    const change = vi.fn();
    const { rerender } = render(
      <>
        <label htmlFor='run-name'>Run name</label>
        <Input
          id='run-name'
          value=''
          onChange={change}
          aria-invalid='true'
          aria-describedby='name-error'
        />
        <p id='name-error'>Enter a run name.</p>
      </>,
    );
    const input = screen.getByRole('textbox', { name: 'Run name' });
    expect(input).toHaveAccessibleDescription('Enter a run name.');
    expect(input).toHaveAttribute('aria-invalid', 'true');
    fireEvent.change(input, { target: { value: 'Training' } });
    expect(change).toHaveBeenCalledOnce();
    rerender(<Input aria-label='Run name' value='Training' readOnly />);
    expect(screen.getByRole('textbox', { name: 'Run name' })).toHaveValue('Training');
  });
});
