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

import { createRef, useState } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ComponentProps } from 'react';
import type { Checkbox as CheckboxType } from './checkbox';
import type { Switch as SwitchType } from './switch';

let Checkbox: typeof CheckboxType;
let Switch: typeof SwitchType;

beforeEach(async () => {
  // Each test models one owner document's feature support for its entire lifetime.
  vi.resetModules();
  ({ Checkbox } = await import('./checkbox'));
  ({ Switch } = await import('./switch'));
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

function withoutPointerActivation() {
  const dispatch = HTMLInputElement.prototype.dispatchEvent;
  return vi.spyOn(HTMLInputElement.prototype, 'dispatchEvent').mockImplementation(function (
    this: HTMLInputElement,
    event: Event,
  ) {
    // Match Firefox 128: the click is delivered, but no native input/change or toggle follows.
    if (event.type === 'click' && event instanceof window.PointerEvent) {
      return dispatch.call(this, new Event('click', { bubbles: true, cancelable: true }));
    }
    return dispatch.call(this, event);
  });
}

for (const role of ['checkbox', 'switch'] as const) {
  describe(role, () => {
    function Control(props: ComponentProps<typeof CheckboxType>) {
      return role === 'checkbox' ? <Checkbox {...props} /> : <Switch {...props} />;
    }

    it('falls back for pointer and Space activation once while preserving controlled state', async () => {
      const dispatch = withoutPointerActivation();
      const change = vi.fn();
      function Example() {
        const [checked, setChecked] = useState(false);
        return (
          <Control
            aria-label='Enable'
            checked={checked}
            onCheckedChange={(value, details) => {
              change(value, details);
              setChecked(value);
            }}
          />
        );
      }
      const user = userEvent.setup();
      render(<Example />);
      const control = screen.getByRole(role, { name: 'Enable' });
      await user.click(control);
      expect(control).toBeChecked();
      expect(change).toHaveBeenCalledTimes(1);
      await user.keyboard(' ');
      expect(control).not.toBeChecked();
      expect(change).toHaveBeenCalledTimes(2);
      expect(change.mock.calls.map((call) => call[0])).toEqual([true, false]);
      expect(dispatch.mock.calls.filter(([event]) => event instanceof PointerEvent)).toHaveLength(
        1,
      );
    });

    it('leaves supported native PointerEvent activation on Base UI’s original path', async () => {
      const dispatch = vi.spyOn(HTMLInputElement.prototype, 'dispatchEvent');
      const change = vi.fn();
      render(<Control aria-label='Enable' onCheckedChange={change} />);
      await userEvent.click(screen.getByRole(role, { name: 'Enable' }));
      expect(screen.getByRole(role)).toBeChecked();
      expect(change).toHaveBeenCalledTimes(1);
      expect(change.mock.calls[0][1].event).toBeInstanceOf(PointerEvent);
      expect(dispatch.mock.calls.filter(([event]) => event instanceof PointerEvent)).toHaveLength(
        2,
      );
    });

    it('preserves modifiers, form values, and Base UI change cancellation', () => {
      withoutPointerActivation();
      const change = vi.fn();
      const { rerender } = render(
        <form aria-label='Settings'>
          <Control aria-label='Enable' name='enabled' value='yes' onCheckedChange={change} />
        </form>,
      );
      fireEvent.click(screen.getByRole(role), {
        shiftKey: true,
        ctrlKey: true,
        altKey: true,
        metaKey: true,
      });
      expect(change).toHaveBeenCalledTimes(1);
      expect(change.mock.calls[0][1].event).toMatchObject({
        shiftKey: true,
        ctrlKey: true,
        altKey: true,
        metaKey: true,
      });
      expect(new FormData(screen.getByRole('form') as HTMLFormElement).get('enabled')).toBe('yes');
      rerender(
        <Control
          aria-label='Enable'
          checked={false}
          onCheckedChange={(_value, details) => details.cancel()}
        />,
      );
      fireEvent.click(screen.getByRole(role));
      expect(screen.getByRole(role)).not.toBeChecked();
    });

    it('honors disabled, readOnly and caller cancellation without changes', () => {
      withoutPointerActivation();
      const change = vi.fn();
      const { rerender } = render(
        <Control aria-label='Enable' disabled onCheckedChange={change} />,
      );
      fireEvent.click(screen.getByRole(role));
      rerender(<Control aria-label='Enable' readOnly onCheckedChange={change} />);
      fireEvent.click(screen.getByRole(role));
      const click = vi.fn((event) => event.preventDefault());
      rerender(<Control aria-label='Enable' onClick={click} onCheckedChange={change} />);
      fireEvent.click(screen.getByRole(role));
      expect(click).toHaveBeenCalledTimes(1);
      rerender(
        <Control
          aria-label='Enable'
          onClick={(event) => event.preventBaseUIHandler()}
          onCheckedChange={change}
        />,
      );
      fireEvent.click(screen.getByRole(role));
      expect(change).not.toHaveBeenCalled();
      expect(screen.getByRole(role)).not.toBeChecked();
    });

    it('preserves input object refs and callback-ref cleanup when a ref changes or unmounts', () => {
      withoutPointerActivation();
      const objectRef = createRef<HTMLInputElement>();
      const dispose = vi.fn();
      const callbackRef = vi.fn((_element: HTMLInputElement | null) => dispose);
      const { rerender, unmount } = render(<Control aria-label='Enable' inputRef={objectRef} />);
      expect(objectRef.current?.type).toBe('checkbox');
      rerender(<Control aria-label='Enable' inputRef={callbackRef} />);
      expect(objectRef.current).toBeNull();
      expect(callbackRef).toHaveBeenCalledTimes(1);
      expect(callbackRef.mock.calls[0][0]?.type).toBe('checkbox');
      fireEvent.click(screen.getByRole(role));
      expect(screen.getByRole(role)).toBeChecked();
      unmount();
      expect(dispose).toHaveBeenCalledTimes(1);
      expect(callbackRef).toHaveBeenCalledTimes(1);
    });
  });
}

it('activates a mixed checkbox and retains its Base UI indeterminate semantics', () => {
  withoutPointerActivation();
  const change = vi.fn();
  render(
    <Checkbox aria-label='Select page' indeterminate checked={false} onCheckedChange={change} />,
  );
  expect(screen.getByRole('checkbox')).toBePartiallyChecked();
  fireEvent.click(screen.getByRole('checkbox'));
  expect(change).toHaveBeenCalledTimes(1);
  expect(change.mock.calls[0][0]).toBe(true);
  expect(screen.getByRole('checkbox')).toBePartiallyChecked();
});

it('honors a CheckboxGroup disabled state inherited by its native input', async () => {
  withoutPointerActivation();
  const { CheckboxGroup } = await import('@base-ui/react/checkbox-group');
  const change = vi.fn();
  render(
    <CheckboxGroup disabled>
      <Checkbox value='pipeline' aria-label='Pipeline' onCheckedChange={change} />
    </CheckboxGroup>,
  );
  const checkbox = screen.getByRole('checkbox', { name: 'Pipeline' });
  expect(checkbox).toHaveAttribute('aria-disabled', 'true');
  fireEvent.click(checkbox);
  expect(change).not.toHaveBeenCalled();
  expect(checkbox).not.toBeChecked();
});
