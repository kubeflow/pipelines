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

import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import { queryKeys } from 'src/hooks/queryKeys';
import DetailsTable from 'src/components/DetailsTable';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';
import { Apis } from 'src/lib/Apis';
import { PageProps } from 'src/pages/Page';
import { RoutePage, RouteParams } from 'src/components/Router';
import { Breadcrumb, ToolbarProps } from 'src/components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from 'src/Css';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { formatDateString, enabledDisplayStringV2, errorToMessage } from 'src/lib/Utils';
import { triggerDisplayString } from 'src/lib/TriggerUtils';

export function RecurringRunDetailsV2FC(props: PageProps) {
  const { updateBanner, updateToolbar } = props;

  const recurringRunId = props.match.params[RouteParams.recurringRunId];

  const {
    data: recurringRun,
    error: getRecurringRunError,
    refetch: refetchRecurringRun,
  } = useQuery<V2beta1RecurringRun, Error>({
    queryKey: queryKeys.recurringRun(recurringRunId),
    queryFn: async () => {
      return await Apis.recurringRunServiceApi.getRecurringRun(recurringRunId);
    },

    enabled: !!recurringRunId,
    staleTime: 0,
    gcTime: 0,
  });

  const experimentId = recurringRun?.experiment_id;
  const {
    data: experiment,
    error: getExperimentError,
    refetch: refetchExperiment,
  } = useQuery<V2beta1Experiment, Error>({
    queryKey: queryKeys.experiment(experimentId),
    queryFn: async () => {
      if (!experimentId) {
        throw new Error('Experiment ID is missing');
      }
      return await Apis.experimentServiceApiV2.getExperiment(experimentId);
    },

    enabled: !!experimentId,
    staleTime: 0,
  });

  const refreshRecurringRun = async () => {
    await refetchRecurringRun();
    if (experimentId) {
      await refetchExperiment();
    }
  };

  useEffect(() => {
    const toolbarState = getInitialToolbarState();

    toolbarState.actions[ButtonKeys.ENABLE_RECURRING_RUN].disabled =
      recurringRun?.status === V2beta1RecurringRunStatus.ENABLED;
    toolbarState.actions[ButtonKeys.DISABLE_RECURRING_RUN].disabled =
      recurringRun?.status !== V2beta1RecurringRunStatus.ENABLED;
    toolbarState.pageTitle =
      recurringRun?.display_name || recurringRun?.recurring_run_id || 'Unknown recurring run';
    toolbarState.breadcrumbs = getBreadcrumbs(experiment?.experiment_id, experiment?.display_name);
    updateToolbar(toolbarState);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    recurringRun?.recurring_run_id,
    recurringRun?.display_name,
    recurringRun?.status,
    experiment?.experiment_id,
    experiment?.display_name,
  ]);

  useEffect(() => {
    let cancelled = false;

    const syncBanner = async () => {
      if (getRecurringRunError) {
        const errorMessage = await errorToMessage(getRecurringRunError);
        if (!cancelled) {
          updateBanner({
            additionalInfo: errorMessage ? errorMessage : undefined,
            message:
              `Error: failed to retrieve recurring run: ${recurringRunId}.` +
              (errorMessage ? ' Click Details for more information.' : ''),
            mode: 'error',
          });
        }
        return;
      }

      if (getExperimentError) {
        const errorMessage = await errorToMessage(getExperimentError);
        if (!cancelled) {
          updateBanner({
            additionalInfo: errorMessage ? errorMessage : undefined,
            message:
              `Error: failed to retrieve this recurring run's experiment.` +
              (errorMessage ? ' Click Details for more information.' : ''),
            mode: 'warning',
          });
        }
        return;
      }

      updateBanner({});
    };

    syncBanner();
    return () => {
      cancelled = true;
    };
  }, [getRecurringRunError, getExperimentError, recurringRunId, updateBanner]);

  const deleteCallback = (_: string[], success: boolean) => {
    if (success) {
      const breadcrumbs = props.toolbarProps.breadcrumbs;
      const previousPage = breadcrumbs.length
        ? breadcrumbs[breadcrumbs.length - 1].href
        : RoutePage.EXPERIMENTS;
      props.history.push(previousPage);
    }
  };

  const getInitialToolbarState = (): ToolbarProps => {
    const buttons = new Buttons(props, refreshRecurringRun);
    return {
      actions: buttons
        .cloneRecurringRun(() => (recurringRun ? [recurringRun.recurring_run_id!] : []), true)
        .refresh(refreshRecurringRun)
        .enableRecurringRun(() => (recurringRun ? recurringRun.recurring_run_id! : ''))
        .disableRecurringRun(() => (recurringRun ? recurringRun.recurring_run_id! : ''))
        .delete(
          () => (recurringRun ? [recurringRun.recurring_run_id!] : []),
          'recurring run config',
          deleteCallback,
          true /* useCurrentResource */,
        )
        .getToolbarActionMap(),
      breadcrumbs: [],
      pageTitle: '',
    };
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

function getBreadcrumbs(experimentId?: string, experimentName?: string): Breadcrumb[] {
  const breadcrumbs: Breadcrumb[] = [];
  if (experimentId) {
    breadcrumbs.push(
      { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
      {
        displayName: experimentName || 'Unknown experiment name',
        href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experimentId),
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

  parameters = Object.entries(recurringRun.runtime_config?.parameters || []).map(([key, value]) => {
    const displayValue =
      value == null ? '' : typeof value === 'string' ? value : JSON.stringify(value);
    return [key || '', displayValue];
  });

  return parameters;
}
