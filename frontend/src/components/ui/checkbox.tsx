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

import { Checkbox as CheckboxPrimitive } from '@base-ui/react/checkbox';
import { Check, Minus } from 'lucide-react';
import type { ComponentProps } from 'react';
import { cn } from './utils';
import './checkbox.css';

type CheckboxProps = Omit<ComponentProps<typeof CheckboxPrimitive.Root>, 'className'> & {
  className?: string;
};

export function Checkbox({ className, indeterminate, ...props }: CheckboxProps) {
  return (
    <CheckboxPrimitive.Root
      className={cn('kfp-checkbox', className)}
      indeterminate={indeterminate}
      {...props}
    >
      <CheckboxPrimitive.Indicator className='kfp-checkbox-indicator'>
        {indeterminate ? (
          <Minus size={13} aria-hidden='true' />
        ) : (
          <Check size={13} aria-hidden='true' />
        )}
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  );
}
