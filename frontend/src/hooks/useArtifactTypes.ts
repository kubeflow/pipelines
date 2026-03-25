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
import { queryKeys } from './queryKeys';

/**
 * Shared hook for fetching MLMD artifact types.
 *
 * getArtifactTypes() is a no-arg function that returns a global list of
 * artifact types from the Metadata store. This hook ensures a single canonical
 * query key so the result is cached and deduplicated across all consumers
 * (CompareV2, InputOutputTab, RuntimeNodeDetailsV2, MetricsTab).
 *
 * Artifact types are static reference data that does not change during a
 * session, so staleTime: Infinity is appropriate here. Note: InputOutputTab
 * and RuntimeNodeDetailsV2 previously omitted staleTime (defaulting to 0),
 * which caused unnecessary refetches on window focus for immutable data.
 */
export function useArtifactTypes(
  options?: Pick<UseQueryOptions<ArtifactType[], Error>, 'enabled'>,
) {
  return useQuery<ArtifactType[], Error>({
    queryKey: queryKeys.artifactTypes(),
    queryFn: () => getArtifactTypes(),
    staleTime: Infinity,
    ...options,
  });
}
