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
import { FileError, FileRejection, useDropzone } from 'react-dropzone';

/**
 * The imperative handle exposed via ref. Class components that cannot use hooks
 * directly hold a ref of this type and call open() to trigger the file picker.
 */
export interface DropzoneAreaHandle {
  open: () => void;
}

interface DropzoneAreaProps {
  onDrop: (files: File[]) => void;
  onDropRejected?: (rejections: FileRejection[]) => void;
  onDragEnter?: () => void;
  onDragLeave?: () => void;
  /** v14 accept format: MIME type → array of extensions, e.g. { 'application/yaml': ['.yaml'] } */
  accept?: Record<string, string[]>;
  disabled?: boolean;
  id?: string;
  style?: React.CSSProperties;
  /** Custom file validator run after accept/MIME filtering but before onDropAccepted. */
  validator?: (file: File) => FileError | FileError[] | null;
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
 *
 * Uses onDropAccepted so the onDrop prop is only called with non-empty accepted
 * files, eliminating the need for empty-array guards in consumers.
 */
const DropzoneArea = React.forwardRef<DropzoneAreaHandle, DropzoneAreaProps>((props, ref) => {
  const {
    onDrop,
    onDropRejected,
    onDragEnter,
    onDragLeave,
    accept,
    disabled,
    validator,
    id,
    style,
    inputProps,
    children,
    'data-testid': dataTestId,
    'aria-label': ariaLabel,
  } = props;

  const { getRootProps, getInputProps, open } = useDropzone({
    onDropAccepted: onDrop,
    onDropRejected,
    noClick: true,
    accept,
    disabled,
    validator,
    onDragEnter: (_e) => onDragEnter?.(),
    onDragLeave: (_e) => onDragLeave?.(),
  });

  React.useImperativeHandle(ref, () => ({ open }), [open]);

  return (
    <div {...getRootProps({ id, style, 'data-testid': dataTestId })}>
      <input {...getInputProps({ ...inputProps, 'aria-label': ariaLabel })} />
      {children}
    </div>
  );
});

DropzoneArea.displayName = 'DropzoneArea';

export default DropzoneArea;
