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

import * as React from 'react';
import { useDropzone } from 'react-dropzone';

/**
 * The imperative handle exposed via ref. Class components that cannot use hooks
 * directly hold a ref of this type and call open() to trigger the file picker.
 */
export interface DropzoneAreaHandle {
  open: () => void;
}

interface DropzoneAreaProps {
  onDrop: (files: File[]) => void;
  onDragEnter?: () => void;
  onDragLeave?: () => void;
  /** v14 accept format: MIME type → array of extensions, e.g. { 'application/yaml': ['.yaml'] } */
  accept?: Record<string, string[]>;
  disabled?: boolean;
  id?: string;
  style?: React.CSSProperties;
  /** Extra attributes forwarded to the hidden <input type="file"> element (e.g. tabIndex). */
  inputProps?: React.InputHTMLAttributes<HTMLInputElement>;
  children?: React.ReactNode;
  'data-testid'?: string;
  'aria-label'?: string;
}

/**
 * A thin functional wrapper around react-dropzone v14's useDropzone hook.
 * Exposes an imperative `open()` handle via forwardRef so class components can
 * programmatically open the file picker without needing to call the hook directly.
 */
const DropzoneArea = React.forwardRef<DropzoneAreaHandle, DropzoneAreaProps>((props, ref) => {
  const {
    onDrop,
    onDragEnter,
    onDragLeave,
    accept,
    disabled,
    id,
    style,
    inputProps,
    children,
    'data-testid': dataTestId,
    'aria-label': ariaLabel,
  } = props;

  const { getRootProps, getInputProps, open } = useDropzone({
    onDrop,
    noClick: true,
    accept,
    disabled,
    onDragEnter: (_e) => onDragEnter?.(),
    onDragLeave: (_e) => onDragLeave?.(),
  });

  React.useImperativeHandle(ref, () => ({ open }), [open]);

  return (
    <div {...getRootProps({ id, style, 'data-testid': dataTestId })}>
      <input aria-label={ariaLabel} {...getInputProps(inputProps)} />
      {children}
    </div>
  );
});

DropzoneArea.displayName = 'DropzoneArea';

export default DropzoneArea;
