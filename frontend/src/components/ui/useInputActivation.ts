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

import { useImperativeHandle, useRef } from 'react';
import type { MouseEvent, Ref } from 'react';
import type { BaseUIEvent } from '@base-ui/react/types';

const pointerActivation = new WeakMap<Document, boolean>();

function supportsPointerActivation(document: Document): boolean {
  const cached = pointerActivation.get(document);
  if (cached !== undefined) return cached;
  const input = document.createElement('input');
  input.type = 'checkbox';
  const view = document.defaultView;
  if (view?.PointerEvent) {
    input.dispatchEvent(new view.PointerEvent('click', { bubbles: true, cancelable: true }));
  }
  pointerActivation.set(document, input.checked);
  return input.checked;
}

type ClickEvent = BaseUIEvent<MouseEvent<HTMLElement>>;

type InputActivationProps = {
  inputRef?: Ref<HTMLInputElement>;
  onClick?: (event: ClickEvent) => void;
  disabled?: boolean;
  readOnly?: boolean;
};

// Firefox 128 dispatches a constructed PointerEvent click without performing the
// native checkbox activation Base UI relies on. Keep Base UI's state/validation
// path and use its public event/ref APIs to activate only affected inputs.
export function useInputActivation({
  inputRef,
  onClick,
  disabled,
  readOnly,
}: InputActivationProps) {
  const input = useRef<HTMLInputElement | null>(null);
  // Synchronize the public DOM ref through React, including callback-ref cleanup.
  useImperativeHandle(inputRef, () => input.current!, []);

  function handleClick(event: ClickEvent) {
    onClick?.(event);
    if (event.defaultPrevented || event.baseUIHandlerPrevented) {
      event.preventBaseUIHandler();
      return;
    }
    const target = input.current;
    if (
      disabled ||
      readOnly ||
      !target ||
      target.disabled ||
      supportsPointerActivation(target.ownerDocument)
    )
      return;
    const view = target.ownerDocument.defaultView;
    if (!view) return;
    event.preventBaseUIHandler();
    event.preventDefault();
    target.dispatchEvent(
      new view.MouseEvent('click', {
        bubbles: true,
        cancelable: true,
        composed: true,
        detail: 0,
        shiftKey: event.shiftKey,
        ctrlKey: event.ctrlKey,
        altKey: event.altKey,
        metaKey: event.metaKey,
      }),
    );
  }

  return { inputRef: input, onClick: handleClick };
}
