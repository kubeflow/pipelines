// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MetadataTransferForm } from './MetadataTransfer';
import * as transfer from 'src/lib/MetadataTransfer';

vi.mock('src/lib/MetadataTransfer', () => ({
  MAX_TRANSFER_BYTES: 256 * 1024 * 1024,
  exportMetadata: vi.fn(),
  importMetadata: vi.fn(),
  downloadMetadata: vi.fn(),
}));
const summary: transfer.TransferResult = {
  counts: { experiments: 2, pipelines: 3, pipeline_versions: 4, runs: 5, schedules: 1 },
  imported: 5,
  skipped: 0,
  dry_run: true,
  warnings: [],
};
const file = new File(['{"integer":9007199254740993}'], 'metadata.json', {
  type: 'application/json',
});
function selectFile(): void {
  fireEvent.change(screen.getByLabelText('Metadata archive (JSON, up to 256 MiB)'), {
    target: { files: [file] },
  });
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(transfer.importMetadata).mockResolvedValue(summary);
});

describe('MetadataTransferForm', () => {
  it('requires successful validation before import and invalidates review when options change', async () => {
    const user = userEvent.setup();
    render(<MetadataTransferForm namespace='team' />);
    const apply = screen.getByRole('button', { name: 'Import metadata' });
    expect(apply).toBeDisabled();
    selectFile();
    expect(apply).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'Validate archive' }));
    await screen.findByText('Archive validated. Ready to import.');
    expect(transfer.importMetadata).toHaveBeenCalledWith(
      'team',
      file,
      'imported-',
      true,
      expect.any(AbortSignal),
    );
    expect(apply).toBeEnabled();
    await user.type(screen.getByLabelText('Imported name prefix'), 'copy');
    expect(apply).toBeDisabled();
    expect(screen.queryByRole('status', { name: 'Import result' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Validate archive' }));
    await screen.findByText('Archive validated. Ready to import.');
    await user.click(apply);
    await screen.findByText('Import complete');
    expect(transfer.importMetadata).toHaveBeenLastCalledWith(
      'team',
      file,
      'imported-copy',
      false,
      expect.any(AbortSignal),
    );
    expect(apply).toBeDisabled();
  });
  it('clears stale errors after a successful retry and file changes invalidate validation', async () => {
    const user = userEvent.setup();
    vi.mocked(transfer.importMetadata).mockRejectedValueOnce(new Error('Conflicting experiment'));
    render(<MetadataTransferForm namespace='team' />);
    selectFile();
    await user.click(screen.getByRole('button', { name: 'Validate archive' }));
    await screen.findByText('Conflicting experiment');
    await user.click(screen.getByRole('button', { name: 'Validate archive' }));
    await screen.findByText('Archive validated. Ready to import.');
    expect(screen.queryByText('Conflicting experiment')).not.toBeInTheDocument();
    selectFile();
    expect(screen.getByRole('button', { name: 'Import metadata' })).toBeDisabled();
  });
  it('disables submissions while validation is pending', async () => {
    const user = userEvent.setup();
    let resolve!: (value: transfer.TransferResult) => void;
    vi.mocked(transfer.importMetadata).mockReturnValueOnce(
      new Promise((done) => {
        resolve = done;
      }),
    );
    render(<MetadataTransferForm namespace='team' />);
    selectFile();
    await user.click(screen.getByRole('button', { name: 'Validate archive' }));
    expect(screen.getByRole('button', { name: 'Validating…' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Download archive' })).toBeDisabled();
    resolve(summary);
    await screen.findByText('Archive validated. Ready to import.');
    expect(transfer.importMetadata).toHaveBeenCalledTimes(1);
  });
  it('downloads history within second-level UTC bounds independent of local timezone', async () => {
    const user = userEvent.setup();
    const blob = new Blob(['{}']);
    vi.mocked(transfer.exportMetadata).mockResolvedValue(blob);
    render(<MetadataTransferForm namespace='team' />);
    expect(screen.getByLabelText('Completed from (UTC)')).toHaveAttribute('step', '1');
    fireEvent.change(screen.getByLabelText('Completed from (UTC)'), {
      target: { value: '2026-10-01T12:30:15' },
    });
    fireEvent.change(screen.getByLabelText('Completed before (UTC)'), {
      target: { value: '2026-10-01T12:31:45' },
    });
    await user.click(screen.getByRole('button', { name: 'Download archive' }));
    await waitFor(() => expect(transfer.downloadMetadata).toHaveBeenCalledWith(blob));
    expect(transfer.exportMetadata).toHaveBeenCalledWith(
      'team',
      {
        completed_after: Date.parse('2026-10-01T12:30:15Z') / 1000,
        completed_before: Date.parse('2026-10-01T12:31:45Z') / 1000,
      },
      expect.any(AbortSignal),
    );
  });
  it('rejects oversized files without validation', () => {
    render(<MetadataTransferForm namespace='team' />);
    const large = new File([], 'large.json');
    Object.defineProperty(large, 'size', { value: transfer.MAX_TRANSFER_BYTES + 1 });
    fireEvent.change(screen.getByLabelText('Metadata archive (JSON, up to 256 MiB)'), {
      target: { files: [large] },
    });
    expect(screen.getByText('Choose an archive no larger than 256 MiB.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Validate archive' })).toBeDisabled();
  });
  it('discards the validated preview when namespace changes', async () => {
    const user = userEvent.setup();
    const view = render(<MetadataTransferForm key='one' namespace='one' />);
    selectFile();
    await user.click(screen.getByRole('button', { name: 'Validate archive' }));
    await screen.findByText('Archive validated. Ready to import.');
    view.rerender(<MetadataTransferForm key='two' namespace='two' />);
    expect(screen.queryByRole('status', { name: 'Import result' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Import metadata' })).toBeDisabled();
  });
  it('aborts an in-flight request when leaving the namespace', async () => {
    const user = userEvent.setup();
    vi.mocked(transfer.importMetadata).mockReturnValueOnce(new Promise(() => {}));
    const view = render(<MetadataTransferForm key='one' namespace='one' />);
    selectFile();
    await user.click(screen.getByRole('button', { name: 'Validate archive' }));
    const signal = vi.mocked(transfer.importMetadata).mock.calls[0][4];
    view.rerender(<MetadataTransferForm key='two' namespace='two' />);
    expect(signal?.aborted).toBe(true);
    expect(screen.getByRole('button', { name: 'Validate archive' })).toBeDisabled();
  });
  it('rejects a reversed date range before starting export', async () => {
    const user = userEvent.setup();
    render(<MetadataTransferForm namespace='team' />);
    fireEvent.change(screen.getByLabelText('Completed from (UTC)'), {
      target: { value: '2026-10-01T12:31:45' },
    });
    fireEvent.change(screen.getByLabelText('Completed before (UTC)'), {
      target: { value: '2026-10-01T12:30:15' },
    });
    await user.click(screen.getByRole('button', { name: 'Download archive' }));
    await screen.findByText('Choose an end time later than the start time.');
    expect(transfer.exportMetadata).not.toHaveBeenCalled();
  });
  it('keeps empty completion bounds unbounded', async () => {
    const user = userEvent.setup();
    vi.mocked(transfer.exportMetadata).mockResolvedValue(new Blob(['{}']));
    render(<MetadataTransferForm namespace='team' />);
    await user.click(screen.getByRole('button', { name: 'Download archive' }));
    await waitFor(() =>
      expect(transfer.exportMetadata).toHaveBeenCalledWith(
        'team',
        {
          completed_after: undefined,
          completed_before: undefined,
        },
        expect.any(AbortSignal),
      ),
    );
  });
});
