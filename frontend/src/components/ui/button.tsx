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
import { Button as ButtonPrimitive } from '@base-ui/react/button';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from './utils';

export const buttonVariants = cva(
  'inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-control border font-kfp-sans text-[13px] font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50 aria-busy:cursor-wait [&_svg]:size-[18px] [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        default: 'border-transparent bg-primary text-primary-foreground hover:brightness-95',
        secondary: 'border-border-strong bg-card text-foreground hover:bg-muted',
        ghost:
          'border-transparent bg-transparent text-foreground-2 hover:bg-muted hover:text-foreground',
        destructive:
          'border-transparent bg-destructive text-destructive-foreground hover:brightness-95',
      },
      size: {
        default: 'h-[34px] px-3',
        sm: 'h-8 px-2.5',
        icon: 'size-[34px] p-0',
      },
    },
    defaultVariants: { variant: 'default', size: 'default' },
  },
);

export type ButtonProps = Omit<React.ComponentProps<typeof ButtonPrimitive>, 'className'> &
  VariantProps<typeof buttonVariants> & { className?: string };

export function Button({ className, variant, size, ...props }: ButtonProps) {
  return (
    <ButtonPrimitive
      data-slot='button'
      className={cn(buttonVariants({ variant, size }), className)}
      {...props}
    />
  );
}
