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
import { render, screen, fireEvent, act } from '@testing-library/react';
import DropzoneArea, { DropzoneAreaHandle } from './DropzoneArea';

function createFile(name: string, type: string): File {
  return new File(['contents'], name, { type });
}

describe('DropzoneArea', () => {
  it('renders children inside the root element', () => {
    render(
      <DropzoneArea onDrop={vi.fn()}>
        <span data-testid='child'>Upload here</span>
      </DropzoneArea>,
    );
    expect(screen.getByTestId('child')).toBeInTheDocument();
  });

  it('forwards data-testid to the root div', () => {
    render(<DropzoneArea onDrop={vi.fn()} data-testid='dz-root' />);
    expect(screen.getByTestId('dz-root')).toBeInTheDocument();
  });

  it('forwards aria-label to the hidden input', () => {
    render(<DropzoneArea onDrop={vi.fn()} aria-label='Upload file' />);
    expect(screen.getByLabelText('Upload file')).toBeInTheDocument();
  });

  it('exposes an imperative open() handle via ref', () => {
    const ref = React.createRef<DropzoneAreaHandle>();
    render(<DropzoneArea onDrop={vi.fn()} ref={ref} />);
    expect(ref.current).toBeDefined();
    expect(typeof ref.current!.open).toBe('function');
  });

  it('calls onDrop when a file is dropped', async () => {
    const handleDrop = vi.fn();
    render(<DropzoneArea onDrop={handleDrop} data-testid='dz' />);

    const file = createFile('pipeline.yaml', 'application/yaml');
    await act(async () => {
      fireEvent.drop(screen.getByTestId('dz'), {
        dataTransfer: { files: [file], types: ['Files'] },
      });
    });

    expect(handleDrop).toHaveBeenCalledTimes(1);
    expect(handleDrop.mock.calls[0][0]).toEqual(
      expect.arrayContaining([expect.objectContaining({ name: 'pipeline.yaml' })]),
    );
  });

  it('calls onDropRejected when a file is rejected by the accept filter', async () => {
    const handleDrop = vi.fn();
    const handleReject = vi.fn();
    render(
      <DropzoneArea
        onDrop={handleDrop}
        onDropRejected={handleReject}
        accept={{ 'application/yaml': ['.yaml'] }}
        data-testid='dz'
      />,
    );

    const file = createFile('readme.txt', 'text/plain');
    await act(async () => {
      fireEvent.drop(screen.getByTestId('dz'), {
        dataTransfer: { files: [file], types: ['Files'] },
      });
    });

    expect(handleDrop).not.toHaveBeenCalled();
    expect(handleReject).toHaveBeenCalled();
  });

  it('passes extra inputProps to the hidden input', () => {
    render(<DropzoneArea onDrop={vi.fn()} inputProps={{ tabIndex: -1 }} aria-label='file input' />);
    const input = screen.getByLabelText('file input');
    expect(input).toHaveAttribute('tabindex', '-1');
  });
});
