/*
 * Copyright 2023 The Kubeflow Authors
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

import React, { useEffect, useState } from 'react';
import { useQuery } from 'react-query';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import DetailsTable from 'src/components/DetailsTable';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { Apis } from 'src/lib/Apis';
import { PageProps } from 'src/pages/Page';
import { RoutePage, RouteParams } from 'src/components/Router';
import { Breadcrumb, ToolbarProps } from 'src/components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from 'src/Css';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { formatDateString, enabledDisplayStringV2 } from 'src/lib/Utils';
import { triggerDisplayString } from 'src/lib/TriggerUtils';

function getInitialToolbarState(
  recurringRun: V2beta1RecurringRun,
  props: PageProps,
  Refresh: () => void,
): ToolbarProps {
  const buttons = new Buttons(props, Refresh);
  return {
    actions: buttons
      .cloneRecurringRun(() => (recurringRun ? [recurringRun.recurring_run_id!] : []), true)
      .refresh(Refresh)
      .enableRecurringRun(() => (recurringRun ? recurringRun.recurring_run_id! : ''))
      .disableRecurringRun(() => (recurringRun ? recurringRun.recurring_run_id! : ''))
      // .delete(
      //   () => (recurringRun ? [recurringRun.recurring_run_id!] : []),
      //   'recurring run config',
      //   this._deleteCallback.bind(this),
      //   true /* useCurrentResource */,
      // )
      .getToolbarActionMap(),
    breadcrumbs: [],
    pageTitle: '',
  };
}

export function RecurringRunDetailsV2FC(props: PageProps) {
  const [refresh, setRefresh] = useState(true);
  const [toolbarState, setToolbarState] = useState<ToolbarProps>();
  const recurringRunId = props.match.params[RouteParams.recurringRunId];
  const Refresh = () => setRefresh(refreshed => !refreshed);

  const { data: recurringRun, refetch: refetchRecurringRun } = useQuery<V2beta1RecurringRun, Error>(
    ['recurringRun'],
    async () => {
      if (!recurringRunId) {
        throw new Error('Recurring run ID is missing');
      }
      return await Apis.recurringRunServiceApi.getRecurringRun(recurringRunId);
    },
    { enabled: !!recurringRunId, staleTime: 0, cacheTime: 0 },
  );

  const experimentId = recurringRun?.experiment_id!;
  const { data: experiment } = useQuery<V2beta1Experiment, Error>(
    ['experiment'],
    async () => {
      if (!experimentId) {
        throw new Error('Experiment ID is missing');
      }
      return await Apis.experimentServiceApiV2.getExperiment(experimentId);
    },
    { enabled: !!experimentId, staleTime: 0 },
  );

  useEffect(() => {
    if (recurringRun) {
      setToolbarState(getInitialToolbarState(recurringRun, props, Refresh));
    }
  }, [recurringRun]);

  useEffect(() => {
    if (toolbarState) {
      toolbarState.actions[ButtonKeys.ENABLE_RECURRING_RUN].disabled =
        recurringRun?.status === V2beta1RecurringRunStatus.ENABLED;
      toolbarState.actions[ButtonKeys.DISABLE_RECURRING_RUN].disabled =
        recurringRun?.status !== V2beta1RecurringRunStatus.ENABLED;
      toolbarState.pageTitle = recurringRun?.display_name!;
      toolbarState.breadcrumbs = getBreadcrumbs(experiment);
      props.updateToolbar(toolbarState);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [toolbarState, experiment, recurringRun]);

  useEffect(() => {
    refetchRecurringRun();
  }, [refresh, refetchRecurringRun]);

  const deleteCallback = (selectedIds: string[], success: boolean) => {
    if (success) {
      const breadcrumbs = props.toolbarProps.breadcrumbs;
      const previousPage = breadcrumbs.length
        ? breadcrumbs[breadcrumbs.length - 1].href
        : RoutePage.EXPERIMENTS;
      props.history.push(previousPage);
    }
  };

  return (
    <div className={classes(commonCss.page, padding(20, 'lr'))}>
      {recurringRun && (
        <div className={commonCss.scrollContainer}>
          <div className={padding(20)}>
            <DetailsTable
              title='Recurring run details'
              fields={getRecurringRunDetails(recurringRun)}
            ></DetailsTable>
            <DetailsTable title='Run triggers' fields={getRunTriggers(recurringRun)}></DetailsTable>
            <DetailsTable
              title='Run parameters'
              fields={getRunParameters(recurringRun)}
            ></DetailsTable>
          </div>
        </div>
      )}
    </div>
  );
}

function getBreadcrumbs(experiment?: V2beta1Experiment): Breadcrumb[] {
  const breadcrumbs: Breadcrumb[] = [];
  if (experiment) {
    breadcrumbs.push(
      { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
      {
        displayName: experiment.display_name!,
        href: RoutePage.EXPERIMENT_DETAILS.replace(
          ':' + RouteParams.experimentId,
          experiment.experiment_id!,
        ),
      },
    );
  } else {
    breadcrumbs.push({ displayName: 'All runs', href: RoutePage.RUNS });
  }

  return breadcrumbs;
}

function getRecurringRunDetails(recurringRun: V2beta1RecurringRun): Array<KeyValue<string>> {
  let details: Array<KeyValue<string>> = [];

  details.push(['Description', recurringRun.description!]);
  details.push(['Created at', formatDateString(recurringRun.created_at)]);

  return details;
}

function getRunTriggers(recurringRun: V2beta1RecurringRun): Array<KeyValue<string>> {
  let triggers: Array<KeyValue<string>> = [];

  triggers.push(['Enabled', enabledDisplayStringV2(recurringRun.trigger, recurringRun.status!)]);
  triggers.push(['Trigger', triggerDisplayString(recurringRun.trigger)]);
  triggers.push(['Max. concurrent runs', recurringRun.max_concurrency]);
  triggers.push(['Catchup', `${!recurringRun.no_catchup}`]);
  triggers.push(['Start time', '']);

  return triggers;
}

function getRunParameters(recurringRun: V2beta1RecurringRun): Array<KeyValue<string>> {
  let parameters: Array<KeyValue<string>> = [];

  parameters = Object.entries(recurringRun.runtime_config?.parameters || []).map(param => [
    param[0] || '',
    param[1] || '',
  ]);

  return parameters;
}

function deleteCallback(selectedIds: string[], success: boolean, props: PageProps) {
  if (success) {
    const breadcrumbs = props.toolbarProps.breadcrumbs;
    const previousPage = breadcrumbs.length
      ? breadcrumbs[breadcrumbs.length - 1].href
      : RoutePage.EXPERIMENTS;
    props.history.push(previousPage);
  }
}
