/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import {
  V2beta1ListPipelineVersionsResponse,
  V2beta1PipelineVersion,
} from '../apisv2beta1/pipeline';

export const VERSION_DISCOVERY_TIMEOUT_MS = 10_000;
export const VERSION_DISCOVERY_INTERVAL_MS = 500;

const abortError = () => new DOMException('Version discovery cancelled', 'AbortError');

function abortable<T>(operation: Promise<T>, signal: AbortSignal): Promise<T> {
  return new Promise((resolve, reject) => {
    const onAbort = () => reject(abortError());
    if (signal.aborted) onAbort();
    else signal.addEventListener('abort', onAbort, { once: true });
    operation.then(
      (value) => {
        signal.removeEventListener('abort', onAbort);
        resolve(value);
      },
      (error) => {
        signal.removeEventListener('abort', onAbort);
        reject(error);
      },
    );
  });
}

function pause(signal: AbortSignal): Promise<void> {
  let timer: ReturnType<typeof setTimeout>;
  const waiting = new Promise<void>((resolve) => {
    timer = setTimeout(resolve, VERSION_DISCOVERY_INTERVAL_MS);
  });
  return abortable(waiting, signal).finally(() => clearTimeout(timer));
}

/** Poll only empty successful reads; uploads and API errors are never retried here. */
export async function discoverUploadedPipelineVersion(
  pipelineId: string,
  readVersions: (signal: AbortSignal) => Promise<V2beta1ListPipelineVersionsResponse>,
  signal: AbortSignal,
): Promise<V2beta1PipelineVersion> {
  const controller = new AbortController();
  const cancel = () => controller.abort();
  signal.addEventListener('abort', cancel, { once: true });
  if (signal.aborted) cancel();
  let timedOut = false;
  const deadline = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, VERSION_DISCOVERY_TIMEOUT_MS);
  try {
    while (!controller.signal.aborted) {
      const response = await abortable(readVersions(controller.signal), controller.signal);
      if (response.pipeline_versions?.length) {
        const version = response.pipeline_versions[0];
        if (
          !version?.pipeline_version_id ||
          (version.pipeline_id && version.pipeline_id !== pipelineId)
        ) {
          throw new Error('Version discovery returned an invalid pipeline version.');
        }
        return { ...version, pipeline_id: pipelineId };
      }
      await pause(controller.signal);
    }
    throw abortError();
  } catch (error) {
    if (timedOut) {
      throw new Error('The pipeline version is not available yet.', { cause: error });
    }
    throw error;
  } finally {
    clearTimeout(deadline);
    signal.removeEventListener('abort', cancel);
  }
}
