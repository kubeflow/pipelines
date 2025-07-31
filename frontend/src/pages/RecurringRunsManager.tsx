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
import BusyButton from 'src/atoms/BusyButton';
import CustomTable, { Column, Row, CustomRendererProps } from 'src/components/CustomTable';
import Toolbar, { ToolbarActionMap } from 'src/components/Toolbar';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { Apis, JobSortKeys, ListRequest } from 'src/lib/Apis';
import { DialogProps, RoutePage, RouteParams } from 'src/components/Router';
import { Link } from 'react-router-dom';
import { SnackbarProps } from '@mui/material/Snackbar';
import { commonCss } from 'src/Css';
import { logger, formatDateString, errorToMessage } from 'src/lib/Utils';

export type RecurringRunListProps = {
  experimentId: string;
  updateDialog: (dialogProps: DialogProps) => void;
  updateSnackbar: (snackbarProps: SnackbarProps) => void;
};

const RecurringRunsManager: React.FC<RecurringRunListProps> = props => {
  const [busyIds, setBusyIds] = React.useState<Set<string>>(new Set());
  const [runs, setRuns] = React.useState<V2beta1RecurringRun[]>([]);
  const [selectedIds, setSelectedIds] = React.useState<string[]>([]);
  const [toolbarActionMap, setToolbarActionMap] = React.useState<ToolbarActionMap>({});
  const tableRef = React.useRef<CustomTable>(null);

  const refresh = React.useCallback(async (): Promise<void> => {
    if (tableRef.current) {
      await tableRef.current.reload();
    }
  }, []);

  const nameCustomRenderer: React.FC<CustomRendererProps<string>> = rendererProps => {
    return (
      <Link
        className={commonCss.link}
        to={RoutePage.RECURRING_RUN_DETAILS.replace(
          ':' + RouteParams.recurringRunId,
          rendererProps.id,
        )}
      >
        {rendererProps.value}
      </Link>
    );
  };

  const enabledCustomRenderer: React.FC<CustomRendererProps<
    V2beta1RecurringRunStatus
  >> = rendererProps => {
    const isBusy = busyIds.has(rendererProps.id);
    return (
      <BusyButton
        outlined={rendererProps.value === V2beta1RecurringRunStatus.ENABLED}
        title={rendererProps.value === V2beta1RecurringRunStatus.ENABLED ? 'Enabled' : 'Disabled'}
        busy={isBusy}
        onClick={() => {
          const newBusyIds = new Set(busyIds);
          newBusyIds.add(rendererProps.id);
          setBusyIds(newBusyIds);

          const setEnabledState = async () => {
            rendererProps.value === V2beta1RecurringRunStatus.ENABLED
              ? await setEnabledStateHelper(rendererProps.id, false)
              : await setEnabledStateHelper(rendererProps.id, true);

            const updatedBusyIds = new Set(busyIds);
            updatedBusyIds.delete(rendererProps.id);
            setBusyIds(updatedBusyIds);
            await refresh();
          };

          setEnabledState();
        }}
      />
    );
  };

  const setEnabledStateHelper = React.useCallback(
    async (id: string, enabled: boolean): Promise<void> => {
      try {
        await (enabled
          ? Apis.recurringRunServiceApi.enableRecurringRun(id)
          : Apis.recurringRunServiceApi.disableRecurringRun(id));
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        props.updateDialog({
          buttons: [{ text: 'Dismiss' }],
          content: 'Error changing enabled state of recurring run:\n' + errorMessage,
          title: 'Error',
        });
        logger.error('Error changing enabled state of recurring run', errorMessage);
      }
    },
    [props],
  );

  const loadRuns = React.useCallback(
    async (request: ListRequest): Promise<string> => {
      let runs: V2beta1RecurringRun[] = [];
      let nextPageToken = '';
      try {
        const response = await Apis.recurringRunServiceApi.listRecurringRuns(
          request.pageToken,
          request.pageSize,
          request.sortBy,
          undefined,
          request.filter,
          props.experimentId,
        );
        runs = response.recurringRuns || [];
        nextPageToken = response.next_page_token || '';
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        props.updateDialog({
          buttons: [{ text: 'Dismiss' }],
          content: 'List recurring run configs request failed with:\n' + errorMessage,
          title: 'Error retrieving recurring run configs',
        });
        logger.error('Could not get list of recurring runs', errorMessage);
      }

      setRuns(runs);
      return nextPageToken;
    },
    [props],
  );

  const columns: Column[] = [
    {
      customRenderer: nameCustomRenderer,
      flex: 2,
      label: 'Run name',
      sortKey: JobSortKeys.NAME,
    },
    { label: 'Created at', flex: 2, sortKey: JobSortKeys.CREATED_AT },
    { customRenderer: enabledCustomRenderer, label: '', flex: 1 },
  ];

  const rows: Row[] = runs.map(r => {
    return {
      error: r.error?.toString(),
      id: r.recurring_run_id!,
      otherFields: [r.display_name, formatDateString(r.created_at), r.status],
    };
  });

  return (
    <React.Fragment>
      <Toolbar actions={toolbarActionMap} breadcrumbs={[]} pageTitle='Recurring runs' />
      <CustomTable
        columns={columns}
        rows={rows}
        ref={tableRef}
        selectedIds={selectedIds}
        updateSelection={ids => setSelectedIds(ids)}
        initialSortColumn={JobSortKeys.CREATED_AT}
        reload={loadRuns}
        filterLabel='Filter recurring runs'
        disableSelection={true}
        emptyMessage={'No recurring runs found in this experiment.'}
      />
    </React.Fragment>
  );
};

export default RecurringRunsManager;
