/*
 * Copyright 2018 The Kubeflow Authors
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
import CustomTable, { Column, Row, CustomRendererProps } from 'src/components/CustomTable';
import Metric from 'src/components/Metric';
import { MetricMetadata, ExperimentInfo } from 'src/lib/RunUtils';
import { V2beta1Run, V2beta1RuntimeState, V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { V2beta1ListExperimentsResponse } from 'src/apisv2beta1/experiment';
import { Apis, RunSortKeys, ListRequest } from 'src/lib/Apis';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import { RoutePage, RouteParams, QUERY_PARAMS } from 'src/components/Router';
import { useURLParser } from 'src/lib/URLParser';
import { commonCss, color } from 'src/Css';
import { formatDateString, logger, errorToMessage, getRunDurationV2 } from 'src/lib/Utils';
import { statusToIcon } from './StatusV2';
import Tooltip from '@mui/material/Tooltip';
import { ForwardedLink } from 'src/atoms/ForwardedLink';

interface PipelineVersionInfo {
  displayName?: string;
  versionId?: string;
  runId?: string;
  recurringRunId?: string;
  pipelineId?: string;
  usePlaceholder: boolean;
}

interface RecurringRunInfo {
  displayName?: string;
  id?: string;
}

interface DisplayRun {
  experiment?: ExperimentInfo;
  recurringRun?: RecurringRunInfo;
  run: V2beta1Run;
  pipelineVersion?: PipelineVersionInfo;
  error?: string;
}

interface DisplayMetric {
  metadata?: MetricMetadata;
  // run metric field is currently not supported in v2 API
  // Context: https://github.com/kubeflow/pipelines/issues/8957
}

// Both masks cannot be provided together.
type MaskProps = Exclude<
  { experimentIdMask?: string; namespaceMask?: string },
  { experimentIdMask: string; namespaceMask: string }
>;

export type RunListProps = MaskProps & {
  disablePaging?: boolean;
  disableSelection?: boolean;
  disableSorting?: boolean;
  hideExperimentColumn?: boolean;
  hideMetricMetadata?: boolean;
  noFilterBox?: boolean;
  onError: (message: string, error: Error) => void;
  onSelectionChange?: (selectedRunIds: string[]) => void;
  runIdListMask?: string[];
  selectedIds?: string[];
  storageState?: V2beta1RunStorageState;
};

const RunList: React.FC<RunListProps> = props => {
  const [metrics, setMetrics] = React.useState<MetricMetadata[]>([]);
  const [runs, setRuns] = React.useState<DisplayRun[]>([]);
  const tableRef = React.useRef<CustomTable>(null);
  const location = useLocation();
  const navigate = useNavigate();
  const urlParser = useURLParser();

  const refresh = React.useCallback(async (): Promise<void> => {
    if (tableRef.current) {
      await tableRef.current.reload();
    }
  }, []);

  const nameCustomRenderer: React.FC<CustomRendererProps<string>> = rendererProps => {
    return (
      <Tooltip title={rendererProps.value || ''} enterDelay={300} placement='top-start'>
        <ForwardedLink
          className={commonCss.link}
          onClick={e => e.stopPropagation()}
          to={RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, rendererProps.id)}
        >
          {rendererProps.value}
        </ForwardedLink>
      </Tooltip>
    );
  };

  const pipelineVersionCustomRenderer: React.FC<CustomRendererProps<
    PipelineVersionInfo
  >> = rendererProps => {
    // If the getPipeline call failed or a run has no pipeline, we display a placeholder.
    if (
      !rendererProps.value ||
      (!rendererProps.value.usePlaceholder && !rendererProps.value.pipelineId)
    ) {
      return <div>-</div>;
    }
    const search = rendererProps.value.recurringRunId
      ? urlParser.build({ [QUERY_PARAMS.fromRecurringRunId]: rendererProps.value.recurringRunId })
      : urlParser.build({ [QUERY_PARAMS.fromRunId]: rendererProps.id });
    const url = rendererProps.value.usePlaceholder
      ? RoutePage.PIPELINE_DETAILS_NO_VERSION.replace(':' + RouteParams.pipelineId + '?', '') +
        search
      : !!rendererProps.value.versionId
      ? RoutePage.PIPELINE_DETAILS.replace(
          ':' + RouteParams.pipelineId,
          rendererProps.value.pipelineId || '',
        ).replace(':' + RouteParams.pipelineVersionId, rendererProps.value.versionId || '')
      : RoutePage.PIPELINE_DETAILS_NO_VERSION.replace(
          ':' + RouteParams.pipelineId,
          rendererProps.value.pipelineId || '',
        );
    if (rendererProps.value.usePlaceholder) {
      return (
        <Link className={commonCss.link} onClick={e => e.stopPropagation()} to={url}>
          [View pipeline]
        </Link>
      );
    } else {
      // Display name could be too long, so we show the full content in tooltip on hover.
      return (
        <Tooltip
          title={rendererProps.value.displayName || ''}
          enterDelay={300}
          placement='top-start'
        >
          <ForwardedLink className={commonCss.link} onClick={e => e.stopPropagation()} to={url}>
            {rendererProps.value.displayName}
          </ForwardedLink>
        </Tooltip>
      );
    }
  };

  const recurringRunCustomRenderer: React.FC<CustomRendererProps<
    RecurringRunInfo
  >> = rendererProps => {
    // If the response of listRuns doesn't contain job details, we display a placeholder.
    if (!rendererProps.value || !rendererProps.value.id) {
      return <div>-</div>;
    }
    const url = RoutePage.RECURRING_RUN_DETAILS.replace(
      ':' + RouteParams.recurringRunId,
      rendererProps.value.id || '',
    );
    return (
      <Link className={commonCss.link} onClick={e => e.stopPropagation()} to={url}>
        {rendererProps.value.displayName || '[View config]'}
      </Link>
    );
  };

  const experimentCustomRenderer: React.FC<CustomRendererProps<ExperimentInfo>> = rendererProps => {
    // If the getExperiment call failed or a run has no experiment, we display a placeholder.
    if (!rendererProps.value || !rendererProps.value.id) {
      return <div>-</div>;
    }
    return (
      <Link
        className={commonCss.link}
        onClick={e => e.stopPropagation()}
        to={RoutePage.EXPERIMENT_DETAILS.replace(
          ':' + RouteParams.experimentId,
          rendererProps.value.id,
        )}
      >
        {rendererProps.value.displayName}
      </Link>
    );
  };

  const statusCustomRenderer: React.FC<CustomRendererProps<
    V2beta1RuntimeState
  >> = rendererProps => {
    return statusToIcon(rendererProps.value);
  };

  const metricBufferCustomRenderer: React.FC<CustomRendererProps<{}>> = () => {
    return <div style={{ borderLeft: `1px solid ${color.divider}`, padding: '20px 0' }} />;
  };

  const metricCustomRenderer: React.FC<CustomRendererProps<DisplayMetric>> = rendererProps => {
    const displayMetric = rendererProps.value;
    if (!displayMetric) {
      return <div />;
    }

    return <Metric metadata={displayMetric.metadata} />;
  };

  const setColumns = React.useCallback(
    async (displayRuns: DisplayRun[]): Promise<DisplayRun[]> => {
      let experimentsResponse: V2beta1ListExperimentsResponse;
      let experimentsGetError: string;
      try {
        if (!props.namespaceMask) {
          // Single-user mode.
          experimentsResponse = await Apis.experimentServiceApiV2.listExperiments();
        } else {
          // Multi-user mode.
          experimentsResponse = await Apis.experimentServiceApiV2.listExperiments(
            undefined,
            undefined,
            undefined,
            undefined,
            props.namespaceMask,
          );
        }
      } catch (error) {
        experimentsGetError =
          'Failed to get associated experiment: ' + (await errorToMessage(error));
      }

      return Promise.all(
        displayRuns.map(async displayRun => {
          setRecurringRun(displayRun);

          await getAndSetPipelineVersionNames(displayRun);

          if (!props.hideExperimentColumn) {
            const experimentId = displayRun.run.experiment_id;

            if (experimentId) {
              const experiment = experimentsResponse?.experiments?.find(
                e => e.experiment_id === displayRun.run.experiment_id,
              );
              // If matching experiment id not found (typically because it has been deleted), set display name to "-".
              const displayName = experiment?.display_name || '-';
              if (experimentsGetError) {
                displayRun.error = experimentsGetError;
              } else {
                displayRun.experiment = {
                  displayName: displayName,
                  id: experimentId,
                };
              }
            }
          }
          return displayRun;
        }),
      );
    },
    [props.namespaceMask, props.hideExperimentColumn],
  );

  const setRecurringRun = (displayRun: DisplayRun): void => {
    const recurringRunId = displayRun.run.recurring_run_id;
    // TBD(jlyaoyuli): how to get recurringRun name
    if (recurringRunId) {
      displayRun.recurringRun = { id: recurringRunId };
    }
  };

  /**
   * For each run ID, fetch its corresponding run, and set it in DisplayRuns
   */
  const getAndSetRuns = (displayRuns: DisplayRun[]): Promise<DisplayRun[]> => {
    return Promise.all(
      displayRuns.map(async displayRun => {
        let getRunResponse: V2beta1Run;
        try {
          getRunResponse = await Apis.runServiceApiV2.getRun(displayRun.run!.run_id!);
          displayRun.run = getRunResponse;
        } catch (err) {
          displayRun.error = await errorToMessage(err);
        }
        return displayRun;
      }),
    );
  };

  /**
   * For the given DisplayRun, get its ApiRun and retrieve that ApiRun's Pipeline ID if it has one,
   * then use that Pipeline ID to fetch its associated Pipeline and attach that Pipeline's name to
   * the DisplayRun. If the ApiRun has no Pipeline ID, then the corresponding DisplayRun will show
   * '-'.
   */
  const getAndSetPipelineVersionNames = async (displayRun: DisplayRun): Promise<void> => {
    const pipelineId = displayRun.run.pipeline_version_reference?.pipeline_id;
    const pipelineVersionId = displayRun.run.pipeline_version_reference?.pipeline_version_id;
    if (pipelineId && pipelineVersionId) {
      try {
        const pipelineVersion = await Apis.pipelineServiceApiV2.getPipelineVersion(
          pipelineId,
          pipelineVersionId,
        );
        displayRun.pipelineVersion = {
          displayName: pipelineVersion.display_name,
          pipelineId: pipelineVersion.pipeline_id,
          usePlaceholder: false,
          versionId: pipelineVersion.pipeline_version_id,
        };
      } catch (err) {
        displayRun.error =
          'Failed to get associated pipeline version: ' + (await errorToMessage(err));
        return;
      }
    } else if (displayRun.run.pipeline_spec) {
      // pipeline_spec in v2 can store either workflow_manifest or pipeline_manifest
      displayRun.pipelineVersion = displayRun.recurringRun?.id
        ? { usePlaceholder: true, recurringRunId: displayRun.recurringRun.id }
        : { usePlaceholder: true };
    }
  };

  const loadRuns = React.useCallback(
    async (request: ListRequest): Promise<string> => {
      let displayRuns: DisplayRun[] = [];
      let nextPageToken = '';

      if (Array.isArray(props.runIdListMask)) {
        displayRuns = props.runIdListMask.map((id: string) => ({ run: { run_id: id } }));
        const filter = JSON.parse(
          decodeURIComponent(request.filter || '{"predicates": []}'),
        ) as V2beta1Filter;
        // listRuns doesn't currently support batching by IDs, so in this case we retrieve and filter
        // each run individually.
        await getAndSetRuns(displayRuns);
        const predicates = filter.predicates?.filter(
          p => p.key === 'name' && p.operation === V2beta1PredicateOperation.ISSUBSTRING,
        );
        const substrings = predicates?.map(p => p.string_value?.toLowerCase() || '') || [];
        displayRuns = displayRuns.filter(runDetail => {
          for (const sub of substrings) {
            if (!runDetail?.run?.display_name?.toLowerCase().includes(sub)) {
              return false;
            }
          }
          return true;
        });
      } else {
        // Load all runs
        if (props.storageState) {
          try {
            // Augment the request filter with the storage state predicate
            const filter = JSON.parse(
              decodeURIComponent(request.filter || '{"predicates": []}'),
            ) as V2beta1Filter;
            filter.predicates = (filter.predicates || []).concat([
              {
                key: 'storage_state',
                // Use EQUALS ARCHIVED or NOT EQUALS ARCHIVED to account for cases where the field
                // is missing, in which case it should be counted as available.
                operation:
                  props.storageState === V2beta1RunStorageState.ARCHIVED
                    ? V2beta1PredicateOperation.EQUALS
                    : V2beta1PredicateOperation.NOTEQUALS,
                string_value: V2beta1RunStorageState.ARCHIVED.toString(),
              },
            ]);
            request.filter = encodeURIComponent(JSON.stringify(filter));
          } catch (err) {
            logger.error('Could not parse request filter: ', request.filter);
          }
        }

        try {
          const response = await Apis.runServiceApiV2.listRuns(
            props.namespaceMask,
            props.experimentIdMask,
            request.pageToken,
            request.pageSize,
            request.sortBy,
            request.filter,
          );

          displayRuns = (response.runs || []).map(r => ({ run: r }));
          nextPageToken = response.next_page_token || '';
        } catch (err) {
          const error = new Error(await errorToMessage(err));
          props.onError('Error: failed to fetch runs.', error);
          // No point in continuing if we couldn't retrieve any runs.
          return '';
        }
      }

      await setColumns(displayRuns);

      setRuns(displayRuns);
      return nextPageToken;
    },
    [props, setColumns, getAndSetRuns],
  );

  // Only show the two most prevalent metrics
  const metricMetadata: MetricMetadata[] = metrics.slice(0, 2);
  const columns: Column[] = [
    {
      customRenderer: nameCustomRenderer,
      flex: 1.5,
      label: 'Run name',
      sortKey: RunSortKeys.NAME,
    },
    { customRenderer: statusCustomRenderer, flex: 0.5, label: 'Status' },
    { label: 'Duration', flex: 0.5 },
    { customRenderer: pipelineVersionCustomRenderer, label: 'Pipeline Version', flex: 1 },
    { customRenderer: recurringRunCustomRenderer, label: 'Recurring Run', flex: 0.5 },
    { label: 'Start time', flex: 1, sortKey: RunSortKeys.CREATED_AT },
  ];

  if (!props.hideExperimentColumn) {
    columns.splice(3, 0, {
      customRenderer: experimentCustomRenderer,
      flex: 1,
      label: 'Experiment',
    });
  }

  if (metricMetadata.length && !props.hideMetricMetadata) {
    // This is a column of empty cells with a left border to separate the metrics from the other
    // columns.
    columns.push({
      customRenderer: metricBufferCustomRenderer,
      flex: 0.1,
      label: '',
    });

    columns.push(
      ...metricMetadata.map(metadata => {
        return {
          customRenderer: metricCustomRenderer,
          flex: 0.5,
          label: metadata.name!,
        };
      }),
    );
  }

  const rows: Row[] = runs.map(r => {
    const displayMetrics = metricMetadata.map(metadata => {
      const displayMetric: DisplayMetric = { metadata };
      return displayMetric;
    });
    const row = {
      error: r.error,
      id: r.run.run_id!,
      otherFields: [
        r.run!.display_name,
        r.run.state || '-',
        getRunDurationV2(r.run),
        r.pipelineVersion,
        r.recurringRun,
        formatDateString(r.run.created_at),
      ] as any,
    };
    if (!props.hideExperimentColumn) {
      row.otherFields.splice(3, 0, r.experiment);
    }
    if (displayMetrics.length && !props.hideMetricMetadata) {
      row.otherFields.push(''); // Metric buffer column
      row.otherFields.push(...(displayMetrics as any));
    }
    return row;
  });

  return (
    <div>
      <CustomTable
        columns={columns}
        rows={rows}
        selectedIds={props.selectedIds}
        initialSortColumn={RunSortKeys.CREATED_AT}
        ref={tableRef}
        filterLabel='Filter runs'
        updateSelection={props.onSelectionChange}
        reload={loadRuns}
        disablePaging={props.disablePaging}
        disableSorting={props.disableSorting}
        disableSelection={props.disableSelection}
        noFilterBox={props.noFilterBox}
        emptyMessage={
          `No` +
          `${props.storageState === V2beta1RunStorageState.ARCHIVED ? ' archived' : ' available'}` +
          ` runs found` +
          `${
            props.experimentIdMask
              ? ' for this experiment'
              : props.namespaceMask
              ? ' for this namespace'
              : ''
          }.`
        }
      />
    </div>
  );
};

export default RunList;
