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
import { useRef, useSyncExternalStore } from 'react';
import type { ReactNode } from 'react';
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

export interface InspectionPanelProps {
  isOpen: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
}

export function InspectionPanel({ isOpen, title, onClose, children }: InspectionPanelProps) {
  const portalContainer = useRef<HTMLDivElement>(null);
  const narrow = useSyncExternalStore(subscribeToNarrowViewport, isNarrowViewport, () => false);
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
          <Dialog.Popup className='kfp-inspector-shell' aria-modal={narrow || undefined}>
            <Resizable
              className='kfp-inspector-panel'
              defaultSize={{ width: 380, height: '100%' }}
              minWidth={300}
              maxWidth='90vw'
              enable={{ left: !narrow }}
            >
              <header className='kfp-inspector-header'>
                <div>
                  <span className='kfp-inspector-eyebrow'>Node details</span>
                  <Dialog.Title>{title}</Dialog.Title>
                </div>
                <Dialog.Close render={<Button variant='ghost' size='icon' aria-label='close' />}>
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
