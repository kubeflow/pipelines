/*
 * Copyright 2025 The Kubeflow Authors
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

import * as JsYaml from 'js-yaml';
import { useQuery } from '@tanstack/react-query';
import { Apis } from 'src/lib/Apis';
import { queryKeys } from './queryKeys';

/**
 * Shared hook for fetching a pipeline version's template (pipeline_spec as YAML).
 *
 * This was previously duplicated in RunDetailsRouter and RecurringRunDetailsRouter.
 * The query is only enabled when both pipelineId and pipelineVersionId are present.
 * Pipeline version specs are immutable once published, so staleTime and gcTime
 * are set to Infinity.
 */
export function usePipelineVersionTemplate(
  pipelineId: string | undefined,
  pipelineVersionId: string | undefined,
) {
  return useQuery<string, Error>({
    queryKey: queryKeys.pipelineVersionTemplate(pipelineId, pipelineVersionId),
    queryFn: async () => {
      if (!pipelineId || !pipelineVersionId) {
        return '';
      }
      const pipelineVersion = await Apis.pipelineServiceApiV2.getPipelineVersion(
        pipelineId,
        pipelineVersionId,
      );
      const pipelineSpec = pipelineVersion.pipeline_spec;
      return pipelineSpec ? JsYaml.safeDump(pipelineSpec) : '';
    },
    enabled: !!pipelineId && !!pipelineVersionId,
    staleTime: Infinity,
    gcTime: Infinity,
  });
}
