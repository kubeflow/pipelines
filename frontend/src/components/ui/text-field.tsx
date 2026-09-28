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

import { useId } from 'react';
import type { ComponentProps, ReactNode } from 'react';
import { Input } from './input';
import { cn } from './utils';
import './text-field.css';

type FieldDescription = {
  label: ReactNode;
  hint?: ReactNode;
  error?: ReactNode;
  trailingContent?: ReactNode;
  containerClassName?: string;
};

export type TextFieldProps = FieldDescription &
  (
    | ({ multiline?: false } & ComponentProps<typeof Input>)
    | ({ multiline: true } & ComponentProps<'textarea'>)
  );

export function TextField(props: TextFieldProps) {
  const generatedId = useId();
  const id = props.id || generatedId;
  const { label, hint, error, trailingContent, containerClassName } = props;
  const description =
    [props['aria-describedby'], hint && `${id}-hint`, error && `${id}-error`]
      .filter(Boolean)
      .join(' ') || undefined;

  let control: ReactNode;
  if (props.multiline) {
    const {
      multiline,
      label,
      hint,
      error,
      trailingContent,
      containerClassName,
      className,
      ...rest
    } = props;
    control = (
      <textarea
        {...rest}
        id={id}
        className={cn('kfp-text-field-textarea', className)}
        aria-describedby={description}
        aria-invalid={!!error || props['aria-invalid']}
      />
    );
  } else {
    const { multiline, label, hint, error, trailingContent, containerClassName, ...rest } = props;
    control = (
      <Input
        {...rest}
        id={id}
        aria-describedby={description}
        aria-invalid={!!error || props['aria-invalid']}
      />
    );
  }

  return (
    <div className={cn('kfp-text-field', containerClassName)}>
      <label htmlFor={id}>
        {label}
        {props.required && <span aria-hidden='true'> *</span>}
      </label>
      <div className='kfp-text-field-control'>
        {control}
        {trailingContent}
      </div>
      {hint && (
        <div className='kfp-text-field-hint' id={`${id}-hint`}>
          {hint}
        </div>
      )}
      {error && (
        <div className='kfp-text-field-error' id={`${id}-error`} role='alert'>
          {error}
        </div>
      )}
    </div>
  );
}
