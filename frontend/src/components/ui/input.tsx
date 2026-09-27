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

import * as React from 'react';
import { Input as InputPrimitive } from '@base-ui/react/input';
import { cn } from './utils';

export type InputProps = Omit<React.ComponentProps<typeof InputPrimitive>, 'className'> & {
  className?: string;
};

export function Input({ className, ...props }: InputProps) {
  return (
    <InputPrimitive
      data-slot='input'
      className={cn(
        'h-[34px] w-full min-w-0 rounded-control border border-muted-foreground bg-background px-3 font-kfp-sans text-[13px] text-foreground placeholder:text-muted-foreground focus:border-primary disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-destructive',
        className,
      )}
      {...props}
    />
  );
}
