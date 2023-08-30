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
  const [refresh, setRefresh] = useState(true);
  const [getRecurringRunErrMsg, setGetRecurringRunErrMsg] = useState<string>('');
  const [getExperimentErrMsg, setGetExperimentErrMsg] = useState<string>('');

  // Related to Api Response
  const [experimentName, setExperimentName] = useState<string>();
  const [experimentIdFromApi, setExperimentIdFromApi] = useState<string>();
  const [recurringRunName, setRecurringRunName] = useState<string>();
  const [recurringRunIdFromApi, setRecurringRunIdFromApi] = useState<string>();
  const [recurringRunStatus, setRecurringRunStatus] = useState<V2beta1RecurringRunStatus>();

  const recurringRunId = props.match.params[RouteParams.recurringRunId];
  const Refresh = () => setRefresh(refreshed => !refreshed);

  const {
    data: recurringRun,
    error: getRecurringRunError,
    refetch: refetchRecurringRun,
  } = useQuery<V2beta1RecurringRun, Error>(
    ['recurringRun', recurringRunId],
    async () => {
      return await Apis.recurringRunServiceApi.getRecurringRun(recurringRunId);
    },
    { enabled: !!recurringRunId, staleTime: 0, cacheTime: 0 },
  );

  const experimentId = recurringRun?.experiment_id!;
  const { data: experiment, error: getExperimentError } = useQuery<V2beta1Experiment, Error>(
    ['experiment'],
    async () => {
      return await Apis.experimentServiceApiV2.getExperiment(experimentId);
    },
    { enabled: !!experimentId, staleTime: 0 },
  );

  useEffect(() => {
    if (recurringRun) {
      setRecurringRunName(recurringRun.display_name);
      setRecurringRunStatus(recurringRun.status);
      setRecurringRunIdFromApi(recurringRun.recurring_run_id);
    }
  }, [recurringRun]);

  useEffect(() => {
    if (experiment) {
      setExperimentName(experiment.display_name);
      setExperimentIdFromApi(experiment.experiment_id);
    }
  }, [experiment]);

  useEffect(() => {
    const toolbarState = getInitialToolbarState();

    toolbarState.actions[ButtonKeys.ENABLE_RECURRING_RUN].disabled =
      recurringRunStatus === V2beta1RecurringRunStatus.ENABLED;
    toolbarState.actions[ButtonKeys.DISABLE_RECURRING_RUN].disabled =
      recurringRunStatus !== V2beta1RecurringRunStatus.ENABLED;
    toolbarState.pageTitle = recurringRunName || recurringRunIdFromApi || 'Unknown recurring run';
    toolbarState.breadcrumbs = getBreadcrumbs(experimentIdFromApi, experimentName);
    updateToolbar(toolbarState);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    recurringRunIdFromApi,
    recurringRunName,
    recurringRunStatus,
    experimentIdFromApi,
    experimentName,
  ]);

  useEffect(() => {
    if (getRecurringRunError) {
      (async () => {
        const errorMessage = await errorToMessage(getRecurringRunError);
        setGetRecurringRunErrMsg(errorMessage);
      })();
    }

    // getExperimentError is from the getExperiment useQuery which is enabled by the
    // experiment ID in recurringRun object. => when getExperimentError changed,
    // getRecurringRun useQuery must be successful (getRecurringRunError is null)
    if (getExperimentError) {
      (async () => {
        const errorMessage = await errorToMessage(getExperimentError);
        setGetExperimentErrMsg(errorMessage);
      })();
    }
  }, [getRecurringRunError, getExperimentError]);

  useEffect(() => {
    if (getRecurringRunErrMsg) {
      updateBanner({
        additionalInfo: getRecurringRunErrMsg ? getRecurringRunErrMsg : undefined,
        message:
          `Error: failed to retrieve recurring run: ${recurringRunId}.` +
          (getRecurringRunErrMsg ? ' Click Details for more information.' : ''),
        mode: 'error',
      });
    }

    if (getExperimentErrMsg) {
      updateBanner({
        additionalInfo: getExperimentErrMsg ? getExperimentErrMsg : undefined,
        message:
          `Error: failed to retrieve this recurring run's experiment.` +
          (getExperimentErrMsg ? ' Click Details for more information.' : ''),
        mode: 'warning',
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [getRecurringRunErrMsg, getExperimentErrMsg]);

  useEffect(() => {
    refetchRecurringRun();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refresh]);

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
    const buttons = new Buttons(props, Refresh);
    return {
      actions: buttons
        .cloneRecurringRun(() => (recurringRun ? [recurringRun.recurring_run_id!] : []), true)
        .refresh(Refresh)
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

  parameters = Object.entries(recurringRun.runtime_config?.parameters || []).map(param => [
    param[0] || '',
    param[1] || '',
  ]);

  return parameters;
}
