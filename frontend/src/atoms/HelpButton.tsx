/**
 * Copyright 2021 The Kubeflow Authors
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

import type { ReactNode } from 'react';
import { Popover } from '@base-ui/react/popover';
import { CircleHelp, X } from 'lucide-react';
import { Button } from '../components/ui/button';
import { useTheme } from '../components/shell/ThemeProvider';
import './SharedAtoms.css';

interface HelpButtonProps {
  helpText?: ReactNode;
  label?: string;
}

export function HelpButton({ helpText, label = 'Help' }: HelpButtonProps) {
  const { themeClassName } = useTheme();
  return (
    <Popover.Root>
      <Popover.Trigger
        openOnHover
        delay={150}
        closeDelay={400}
        render={<Button variant='ghost' size='icon' aria-label={label} />}
      >
        <CircleHelp aria-hidden />
      </Popover.Trigger>
      <Popover.Portal className={themeClassName}>
        <Popover.Positioner side='top' sideOffset={8} className='kfp-help-positioner'>
          <Popover.Popup className='kfp-help-popup'>
            <div className='kfp-help-heading'>
              <Popover.Title>{label}</Popover.Title>
              <Popover.Close
                render={<Button variant='ghost' size='icon' aria-label='Close help' />}
              >
                <X aria-hidden />
              </Popover.Close>
            </div>
            <Popover.Description render={<div />}>{helpText}</Popover.Description>
          </Popover.Popup>
        </Popover.Positioner>
      </Popover.Portal>
    </Popover.Root>
  );
}
