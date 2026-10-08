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

import { useState, type ReactNode } from 'react';
import { Button } from './button';
import { ModalDialog } from './dialog';

/** Keep inline disclosures and modal details distinct while sharing their controls. */
export function NoticeDetails({
  children,
  variant = 'error',
  inline = false,
  unmountWhenClosed = false,
}: {
  children: ReactNode;
  variant?: 'error' | 'warning' | 'info';
  inline?: boolean;
  unmountWhenClosed?: boolean;
}) {
  const [open, setOpen] = useState(false);
  if (!children) return null;
  if (inline)
    return (
      <details>
        <summary>Details</summary>
        {children}
      </details>
    );
  return (
    <>
      <Button variant='secondary' size='sm' onClick={() => setOpen(true)}>
        Details
      </Button>
      {(!unmountWhenClosed || open) && (
        <ModalDialog
          open={open}
          title={
            variant === 'error' ? 'An error occurred' : variant === 'warning' ? 'Warning' : 'Info'
          }
          onClose={() => setOpen(false)}
          actions={<Button onClick={() => setOpen(false)}>Dismiss</Button>}
        >
          {children}
        </ModalDialog>
      )}
    </>
  );
}
