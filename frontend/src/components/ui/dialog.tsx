/*
 * Copyright 2026 The Kubeflow Authors
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

import type { ReactNode } from 'react';
import { Dialog } from '@base-ui/react/dialog';
import { useTheme } from '../modernization/ThemeProvider';
import './dialog.css';

export function ModalDialog({
  open,
  title,
  children,
  actions,
  onClose,
  size = 'sm',
}: {
  open: boolean;
  title: ReactNode;
  children?: ReactNode;
  actions?: ReactNode;
  onClose: () => void;
  size?: 'sm' | 'lg' | 'full';
}) {
  const { resolvedTheme } = useTheme();
  return (
    <Dialog.Root
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onClose();
      }}
    >
      {/* Portals are outside the application theme element, so inherit the resolved palette explicitly. */}
      <Dialog.Portal className={`kfp-theme ${resolvedTheme === 'dark' ? 'dark' : ''}`}>
        <Dialog.Backdrop className='kfp-page-dialog-backdrop' />
        <Dialog.Viewport className='kfp-page-dialog-viewport'>
          <Dialog.Popup className={`kfp-page-dialog kfp-dialog-${size}`}>
            <Dialog.Title className='kfp-page-dialog-title'>{title}</Dialog.Title>
            {children && (
              <Dialog.Description render={<div />} className='kfp-page-dialog-content'>
                {children}
              </Dialog.Description>
            )}
            {actions && <div className='kfp-page-dialog-actions'>{actions}</div>}
          </Dialog.Popup>
        </Dialog.Viewport>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
