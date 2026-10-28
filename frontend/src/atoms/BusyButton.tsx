/*
 * Copyright 2018 The Kubeflow Authors
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

import type { ComponentType } from 'react';
import { LoaderCircle } from 'lucide-react';
import { Button, type ButtonProps } from '../components/ui/button';
import { cn } from '../components/ui/utils';
import './SharedAtoms.css';

type BusyButtonProps = Omit<ButtonProps, 'title' | 'color'> & {
  title: string;
  icon?: ComponentType<{ className?: string; 'aria-hidden'?: boolean }>;
  busy?: boolean;
  outlined?: boolean;
};

export default function BusyButton({
  title,
  icon: Icon,
  busy,
  outlined,
  disabled,
  className,
  variant,
  ...props
}: BusyButtonProps) {
  return (
    <Button
      {...props}
      className={cn('kfp-busy-button', className)}
      variant={outlined ? 'secondary' : variant}
      disabled={busy || disabled}
      aria-busy={busy || undefined}
    >
      {Icon && <Icon aria-hidden />}
      <span>{title}</span>
      {busy && <LoaderCircle className='kfp-atom-spinner' aria-hidden />}
    </Button>
  );
}
