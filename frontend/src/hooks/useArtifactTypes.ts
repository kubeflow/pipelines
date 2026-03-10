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

import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getArtifactTypes } from 'src/mlmd/MlmdUtils';
import { ArtifactType } from 'src/third_party/mlmd';
import { queryKeys, STALE_TIME_STATIC } from './queryKeys';

type UseArtifactTypesOptions = Omit<UseQueryOptions<ArtifactType[], Error>, 'queryKey' | 'queryFn'>;

/**
 * Shared hook to fetch artifact types from MLMD.
 * Uses a single canonical query key so the cache is shared across all consumers.
 * Replaces inline useQuery usages in CompareV2, InputOutputTab, RuntimeNodeDetailsV2, MetricsTab.
 */
export function useArtifactTypes(options?: UseArtifactTypesOptions) {
  return useQuery<ArtifactType[], Error>({
    queryKey: queryKeys.artifactTypes(),
    queryFn: () => getArtifactTypes(),
    staleTime: STALE_TIME_STATIC,
    ...options,
  });
}
