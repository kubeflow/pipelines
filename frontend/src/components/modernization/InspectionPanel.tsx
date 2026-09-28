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

import { Dialog } from '@base-ui/react/dialog';
import { X } from 'lucide-react';
import { useCallback, useId, useRef, useState, useSyncExternalStore } from 'react';
import type { KeyboardEvent, ReactNode } from 'react';
import { Resizable } from 're-resizable';
import { Button } from '../ui/button';
import './RunInspection.css';

const narrowQuery = '(max-width: 899px)';
function subscribeToNarrowViewport(onChange: () => void) {
  const media = window.matchMedia?.(narrowQuery);
  media?.addEventListener('change', onChange);
  return () => media?.removeEventListener('change', onChange);
}
function isNarrowViewport() {
  return window.matchMedia?.(narrowQuery).matches || false;
}

function subscribeToViewportWidth(onChange: () => void) {
  window.addEventListener('resize', onChange);
  return () => window.removeEventListener('resize', onChange);
}
function getViewportWidth() {
  return window.innerWidth;
}

export interface InspectionPanelProps {
  isOpen: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
}

export function InspectionPanel({ isOpen, title, onClose, children }: InspectionPanelProps) {
  const portalContainer = useRef<HTMLDivElement>(null);
  const closeButton = useRef<HTMLButtonElement>(null);
  const resizeHandleRef = useCallback((element: HTMLDivElement | null) => {
    if (!element) return;
    return () => {
      // A viewport change removes this handle when the inspector becomes modal.
      if (document.activeElement === element) closeButton.current?.focus();
    };
  }, []);
  const narrow = useSyncExternalStore(subscribeToNarrowViewport, isNarrowViewport, () => false);
  const viewportWidth = useSyncExternalStore(
    subscribeToViewportWidth,
    getViewportWidth,
    () => 1024,
  );
  const panelId = useId();
  const [preferredWidth, setPreferredWidth] = useState(380);
  const minWidth = 300;
  const maxWidth = Math.max(minWidth, Math.floor(viewportWidth * 0.9));
  const width = Math.min(preferredWidth, maxWidth);

  function resizeFromKeyboard(event: KeyboardEvent<HTMLDivElement>) {
    let nextWidth: number;
    switch (event.key) {
      // This handle is on the right pane's left edge: moving left widens that pane.
      case 'ArrowLeft':
        nextWidth = width + 20;
        break;
      case 'ArrowRight':
        nextWidth = width - 20;
        break;
      case 'Home':
        nextWidth = minWidth;
        break;
      case 'End':
        nextWidth = maxWidth;
        break;
      default:
        return;
    }
    event.preventDefault();
    event.stopPropagation();
    setPreferredWidth(Math.max(minWidth, Math.min(nextWidth, maxWidth)));
  }

  return (
    <Dialog.Root
      open={isOpen}
      modal={narrow}
      disablePointerDismissal
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <div ref={portalContainer}>
        <Dialog.Portal container={portalContainer}>
          {narrow && <Dialog.Backdrop className='kfp-inspector-backdrop' />}
          <Dialog.Popup
            id={panelId}
            className='kfp-inspector-shell'
            aria-modal={narrow || undefined}
          >
            <Resizable
              className='kfp-inspector-panel'
              size={{ width, height: '100%' }}
              minWidth={minWidth}
              maxWidth={maxWidth}
              enable={{ left: !narrow }}
              onResize={(_event, _direction, element) => setPreferredWidth(element.offsetWidth)}
              onResizeStop={(_event, _direction, element) => setPreferredWidth(element.offsetWidth)}
              handleComponent={{
                left: (
                  <div
                    ref={resizeHandleRef}
                    className='kfp-inspector-resize-handle'
                    role='separator'
                    tabIndex={0}
                    aria-label='Resize node details'
                    aria-orientation='vertical'
                    aria-controls={panelId}
                    aria-valuemin={minWidth}
                    aria-valuemax={maxWidth}
                    aria-valuenow={width}
                    aria-valuetext={`${width} pixels`}
                    title='Resize node details with Left/Right arrows, Home, or End'
                    onKeyDown={resizeFromKeyboard}
                  />
                ),
              }}
            >
              <header className='kfp-inspector-header'>
                <div>
                  <span className='kfp-inspector-eyebrow'>Node details</span>
                  <Dialog.Title>{title}</Dialog.Title>
                </div>
                <Dialog.Close
                  render={
                    <Button ref={closeButton} variant='ghost' size='icon' aria-label='close' />
                  }
                >
                  <X aria-hidden='true' />
                </Dialog.Close>
              </header>
              <div className='kfp-inspector-content'>{children}</div>
            </Resizable>
          </Dialog.Popup>
        </Dialog.Portal>
      </div>
    </Dialog.Root>
  );
}
