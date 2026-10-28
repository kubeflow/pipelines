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

import * as React from 'react';
import { Link } from 'react-router';
import { V2Run, V2RunStorageState } from 'src/apisv2/run';
import { V2PredicateOperation } from 'src/apisv2/filter';
import { Apis, RunSortKeys } from 'src/lib/Apis';
import { formatDateString } from 'src/lib/Utils';
import { RoutePageFactory } from '../Router';
import { RunStatus } from './RunStatus';
import '../tables/CollectionPreview.css';
import './ExperimentRunPreview.css';

interface Props {
  experimentId: string;
  archived?: boolean;
  runs?: V2Run[];
  error?: string;
}

export function ExperimentRunPreview({ experimentId, archived = false, runs, error }: Props) {
  const [loaded, setLoaded] = React.useState<V2Run[]>();
  const [loadError, setLoadError] = React.useState(false);
  // External synchronization: archived lists fetch their preview when expanded.
  // Active lists already fetched the same five runs for their health summary.
  React.useEffect(() => {
    if (runs !== undefined || error) return;
    let active = true;
    Apis.runServiceApiV2
      .listRuns(
        undefined,
        experimentId,
        undefined,
        5,
        RunSortKeys.CREATED_AT + ' desc',
        encodeURIComponent(
          JSON.stringify({
            predicates: [
              {
                key: 'storage_state',
                operation: archived ? V2PredicateOperation.EQUALS : V2PredicateOperation.NOT_EQUALS,
                string_value: V2RunStorageState.ARCHIVED,
              },
            ],
          }),
        ),
        true,
      )
      .then(
        (response) => {
          if (active) setLoaded(response.runs || []);
        },
        () => {
          if (active) setLoadError(true);
        },
      );
    return () => {
      active = false;
    };
  }, [experimentId, archived, runs, error]);
  const displayedRuns = runs ?? loaded;
  return (
    <section
      className='kfp-collection-preview kfp-experiment-run-preview'
      aria-label='Recent runs preview'
    >
      <div className='kfp-collection-preview-heading'>
        <span>{archived ? 'Recent archived runs' : 'Recent runs'}</span>
        <Link to={RoutePageFactory.experimentDetails(experimentId)}>View all runs</Link>
      </div>
      {error || loadError ? (
        <p role='alert'>Could not load recent runs. Open View all runs to retry.</p>
      ) : displayedRuns === undefined ? (
        <p role='status'>Loading recent runs…</p>
      ) : displayedRuns.length === 0 ? (
        <p>No {archived ? 'archived ' : ''}runs yet.</p>
      ) : (
        <ul>
          {displayedRuns.slice(0, 5).map((run) => (
            <li key={run.run_id}>
              <Link
                title={run.display_name || run.run_id}
                to={RoutePageFactory.runDetails(run.run_id!)}
                state={{ fromExperimentId: experimentId }}
              >
                {run.display_name || run.run_id}
              </Link>
              <RunStatus state={run.state} />
              <span className='kfp-experiment-run-preview-date'>
                {formatDateString(run.created_at)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
