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

export const MAX_TRANSFER_BYTES = 256 * 1024 * 1024;

export interface TransferResult {
  counts: {
    experiments: number;
    pipelines: number;
    pipeline_versions: number;
    runs: number;
    schedules: number;
  };
  imported: number;
  skipped: number;
  dry_run: boolean;
  warnings: string[];
}

export interface ExportRange {
  completed_after?: number;
  completed_before?: number;
}

async function checkedResponse(response: Response): Promise<Response> {
  if (response.ok) return response;
  const text = await response.text();
  let message = text || `Request failed (${response.status}).`;
  try {
    const body: unknown = JSON.parse(text);
    if (
      typeof body === 'object' &&
      body !== null &&
      'error' in body &&
      typeof body.error === 'string'
    ) {
      message = body.error;
    }
  } catch {
    // Proxies can return plain text errors.
  }
  throw new Error(message);
}

export async function exportMetadata(
  namespace: string,
  range: ExportRange,
  signal?: AbortSignal,
): Promise<Blob> {
  const query = new URLSearchParams({ namespace });
  const response = await checkedResponse(
    await fetch(`apis/v2/transfer/export?${query}`, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(range),
      signal,
    }),
  );
  const archive = await response.blob();
  if (archive.size > MAX_TRANSFER_BYTES)
    throw new Error('The archive exceeds 256 MiB. Export a smaller completed-run date range.');
  return archive;
}

export async function importMetadata(
  namespace: string,
  file: File,
  namePrefix: string,
  dryRun: boolean,
  signal?: AbortSignal,
): Promise<TransferResult> {
  if (file.size > MAX_TRANSFER_BYTES) throw new Error('Choose an archive no larger than 256 MiB.');
  const query = new URLSearchParams({
    namespace,
    dry_run: String(dryRun),
    name_prefix: namePrefix,
  });
  const response = await checkedResponse(
    await fetch(`apis/v2/transfer/import?${query}`, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      // Keep archive bytes intact, including integers beyond JavaScript's safe range.
      body: file,
      signal,
    }),
  );
  return response.json();
}

export function downloadMetadata(archive: Blob): void {
  const url = URL.createObjectURL(archive);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = 'pipelines-metadata.json';
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  // Revoke on the next task, after the browser has started the download.
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}
