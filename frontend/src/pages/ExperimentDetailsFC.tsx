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
import Button from '@material-ui/core/Button';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import Paper from '@material-ui/core/Paper';
import PopOutIcon from '@material-ui/icons/Launch';
import RecurringRunsManager from './RecurringRunsManager';
import RunListsRouter, { RunListsGroupTab } from './RunListsRouter';
import Toolbar, { ToolbarProps } from 'src/components/Toolbar';
import Tooltip from '@material-ui/core/Tooltip';
import { V2beta1Experiment, V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { Apis } from 'src/lib/Apis';
import { Page, PageProps } from './Page';
import { RoutePage, RouteParams } from 'src/components/Router';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, padding } from 'src/Css';
import { logger } from 'src/lib/Utils';
import { useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { Redirect } from 'react-router-dom';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';

const css = stylesheet({
  card: {
    border: '1px solid #ddd',
    borderRadius: 8,
    height: 70,
    marginRight: 24,
    padding: '12px 20px 13px 24px',
  },
  cardActive: {
    border: `1px solid ${color.success}`,
  },
  cardBtn: {
    $nest: {
      '&:hover': {
        backgroundColor: 'initial',
      },
    },
    color: color.theme,
    fontSize: 12,
    minHeight: 16,
    paddingLeft: 0,
    marginRight: 0,
  },
  cardContent: {
    color: color.secondaryText,
    fontSize: 16,
    lineHeight: '28px',
  },
  cardRow: {
    borderBottom: `1px solid ${color.lightGrey}`,
    display: 'flex',
    flexFlow: 'row',
    minHeight: 120,
  },
  cardTitle: {
    color: color.strong,
    display: 'flex',
    fontSize: 12,
    fontWeight: 'bold',
    height: 20,
    justifyContent: 'space-between',
    marginBottom: 6,
  },
  popOutIcon: {
    color: color.secondaryText,
    margin: -2,
    minWidth: 0,
    padding: 3,
  },
  recurringRunsActive: {
    color: '#0d652d',
  },
  recurringRunsCard: {
    width: 270,
  },
  recurringRunsDialog: {
    minWidth: 600,
  },
  runStatsCard: {
    width: 270,
  },
});

export function ExperimentDetailsFC(props: PageProps) {
  const [refresh, setRefresh] = useState(true);
  const [toolbarState, setToolbarState] = useState<ToolbarProps>();
  const Refresh = () => setRefresh(refreshed => !refreshed);
  const experimentId = props.match.params[RouteParams.experimentId];

  const { data: experiment } = useQuery<V2beta1Experiment, Error>(
    ['experiment', experimentId],
    async () => {
      if (!experimentId) {
        throw new Error('Experiment ID is missing');
      }
      return Apis.experimentServiceApiV2.getExperiment(experimentId);
    },
    { enabled: !!experimentId, staleTime: Infinity },
  );

  useEffect(() => {
    if (experiment) {
      setToolbarState(getInitialToolbarState(experiment, props, Refresh));
    }
  }, [experiment]);

  useEffect(() => {
    if (toolbarState) {
      toolbarState.pageTitle = experiment?.display_name!;
      props.updateToolbar(toolbarState);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [toolbarState]);

  return <div>experiment details test</div>;
}

function getInitialToolbarState(
  experiment: V2beta1Experiment,
  props: PageProps,
  refresh: () => void,
) {
  const buttons = new Buttons(props, refresh);
  return {
    actions: buttons.refresh(refresh).getToolbarActionMap(),
    breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
    // TODO: determine what to show if no props.
    pageTitle: experiment.display_name!,
  };
}
