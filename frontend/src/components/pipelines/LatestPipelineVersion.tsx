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

import { useContext } from 'react';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router';
import { Apis } from 'src/lib/Apis';
import { RoutePageFactory } from '../Router';
import type { CustomRendererProps } from '../CustomTable';
/** Fetch only the newest version for each visible pipeline, sharing cached results. */
export function LatestPipelineVersion({ id }: CustomRendererProps<string>) {
  const namespace = useContext(NamespaceContext);
  const { data, isPending, isError, refetch } = useQuery({
    queryKey: ['latest_pipeline_version', namespace, id],
    queryFn: () =>
      Apis.pipelineServiceApiV2.listPipelineVersions(id, undefined, 1, 'created_at desc'),
    staleTime: 30_000,
  });
  if (isPending) return <span aria-label='Loading latest version'>…</span>;
  if (isError)
    return (
      <button
        type='button'
        onClick={(event) => {
          event.stopPropagation();
          void refetch();
        }}
      >
        Retry version lookup
      </button>
    );
  const version = data?.pipeline_versions?.[0];
  if (!version?.pipeline_version_id) return <span>No versions</span>;
  return (
    <Link
      onClick={(event) => event.stopPropagation()}
      to={RoutePageFactory.pipelineVersionDetails(id, version.pipeline_version_id)}
    >
      {version.display_name || version.name || version.pipeline_version_id}
    </Link>
  );
}
