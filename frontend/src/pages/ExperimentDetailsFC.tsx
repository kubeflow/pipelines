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
import { errorToMessage, logger } from 'src/lib/Utils';
import { useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { Redirect } from 'react-router-dom';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';

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
  const { updateBanner, updateDialog, updateToolbar } = props;
  const [refresh, setRefresh] = useState(true);
  const Refresh = () => setRefresh(refreshed => !refreshed);
  const [toolbarState, setToolbarState] = useState<ToolbarProps>();
  const [activeRecurringRunsCount, setActiveRecurringRunsCount] = useState(0);
  const [recurringRunsManagerOpen, setRecurringRunsManagerOpen] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [runListToolbarProps, setRunListToolbarProps] = useState<ToolbarProps>({
    actions: getRunInitialToolBarButtons(props, Refresh, selectedIds).getToolbarActionMap(),
    breadcrumbs: [],
    pageTitle: 'Runs',
    topLevelToolbar: false,
  });
  const [runStorageState, setRunStorageState] = useState(V2beta1RunStorageState.AVAILABLE);
  const [runlistRefreshCount, setRunlistRefreshCount] = useState(0);

  const experimentId = props.match.params[RouteParams.experimentId];

  const { data: experiment } = useQuery<V2beta1Experiment | undefined, Error>(
    ['experiment', experimentId],
    async () => {
      try {
        return await Apis.experimentServiceApiV2.getExperiment(experimentId);
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        updateBanner({
          additionalInfo: errorMessage ? errorMessage : undefined,
          message:
            `Error: failed to retrieve experiment: ${experimentId}.` +
            (errorMessage ? ' Click Details for more information.' : ''),
          mode: 'error',
        });
        return;
      }
    },
    { enabled: !!experimentId, staleTime: Infinity },
  );

  const description = experiment?.description || '';

  const { data: recurringRunList, refetch: refetchRecurringRun } = useQuery<
    V2beta1RecurringRun[] | undefined,
    Error
  >(
    ['recurring_run_list'],
    async () => {
      try {
        const listRecurringRunsResponse = await Apis.recurringRunServiceApi.listRecurringRuns(
          undefined,
          100,
          '',
          undefined,
          undefined,
          experimentId,
        );
        return listRecurringRunsResponse.recurringRuns ?? [];
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        updateBanner({
          additionalInfo: errorMessage ? errorMessage : undefined,
          message:
            `Error: failed to retrieve recurring runs for experiment: ${experimentId}.` +
            (errorMessage ? ' Click Details for more information.' : ''),
          mode: 'warning',
        });
        return;
      }
    },
    { enabled: !!experimentId, staleTime: 0, cacheTime: 0 },
  );

  useEffect(() => {
    if (experiment) {
      setToolbarState(getInitialToolbarState(experiment, props, Refresh));
    }
  }, [experiment]);

  useEffect(() => {
    if (toolbarState) {
      toolbarState.pageTitle = experiment?.display_name || experimentId;
      updateToolbar(toolbarState);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [toolbarState]);

  useEffect(() => {
    if (recurringRunList) {
      setActiveRecurringRunsCount(
        recurringRunList.filter(rr => rr.status === V2beta1RecurringRunStatus.ENABLED).length,
      );
    }
  }, [recurringRunList]);

  useEffect(() => {
    refetchRecurringRun();
  }, [refresh, refetchRecurringRun]);

  const showPageError = async (message: string, error: Error | undefined) => {
    const errorMessage = await errorToMessage(error);
    updateBanner({
      additionalInfo: errorMessage ? errorMessage : undefined,
      message: message + (errorMessage ? ' Click Details for more information.' : ''),
    });
  };

  const selectionChanged = (selectedIds: string[]) => {
    const toolbarButtons = getRunInitialToolBarButtons(props, Refresh, selectedIds);
    // If user selects to show Active runs list, shows `Archive` button for selected runs.
    // If user selects to show Archive runs list, shows `Restore` button for selected runs.
    if (runStorageState === V2beta1RunStorageState.AVAILABLE) {
      toolbarButtons.archive(
        'run',
        () => selectedIds,
        false,
        ids => selectionChanged(ids),
      );
    } else {
      toolbarButtons.restore(
        'run',
        () => selectedIds,
        false,
        ids => selectionChanged(ids),
      );
    }
    const toolbarActions = toolbarButtons.getToolbarActionMap();
    toolbarActions[ButtonKeys.COMPARE].disabled =
      selectedIds.length <= 1 || selectedIds.length > 10;
    toolbarActions[ButtonKeys.CLONE_RUN].disabled = selectedIds.length !== 1;
    if (toolbarActions[ButtonKeys.ARCHIVE]) {
      toolbarActions[ButtonKeys.ARCHIVE].disabled = !selectedIds.length;
    }
    if (toolbarActions[ButtonKeys.RESTORE]) {
      toolbarActions[ButtonKeys.RESTORE].disabled = !selectedIds.length;
    }
    setRunListToolbarProps({
      actions: toolbarActions,
      breadcrumbs: runListToolbarProps.breadcrumbs,
      pageTitle: runListToolbarProps.pageTitle,
      topLevelToolbar: runListToolbarProps.topLevelToolbar,
    });
    setSelectedIds(selectedIds);
  };

  const onRunTabSwitch = (tab: RunListsGroupTab) => {
    let updatedRunStorageState = V2beta1RunStorageState.AVAILABLE;
    if (tab === RunListsGroupTab.ARCHIVE) {
      updatedRunStorageState = V2beta1RunStorageState.ARCHIVED;
    }
    let updatedRunlistRefreshCount = runlistRefreshCount + 1;
    setRunStorageState(updatedRunStorageState);
    setRunlistRefreshCount(updatedRunlistRefreshCount);
    setSelectedIds([]);
    return;
  };

  const recurringRunsManagerClosed = () => {
    setRecurringRunsManagerOpen(false);
    // Reload the details to get any updated recurring runs
    Refresh();
  };

  return (
    <div className={classes(commonCss.page, padding(20, 'lrt'))}>
      {experiment && (
        <div className={commonCss.page}>
          <div className={css.cardRow}>
            <Paper
              id='recurringRunsCard'
              className={classes(
                css.card,
                css.recurringRunsCard,
                !!activeRecurringRunsCount && css.cardActive,
              )}
              elevation={0}
            >
              <div>
                <div className={css.cardTitle}>
                  <span>Recurring run configs</span>
                  <Button
                    className={css.cardBtn}
                    id='manageExperimentRecurringRunsBtn'
                    disableRipple={true}
                    onClick={() => setRecurringRunsManagerOpen(true)}
                  >
                    Manage
                  </Button>
                </div>
                <div
                  className={classes(
                    css.cardContent,
                    !!activeRecurringRunsCount && css.recurringRunsActive,
                  )}
                >
                  {activeRecurringRunsCount + ' active'}
                </div>
              </div>
            </Paper>
            <Paper
              id='experimentDescriptionCard'
              className={classes(css.card, css.runStatsCard)}
              elevation={0}
            >
              <div className={css.cardTitle}>
                <span>Experiment description</span>
                <Button
                  id='expandExperimentDescriptionBtn'
                  onClick={() =>
                    updateDialog({
                      content: description,
                      title: 'Experiment description',
                    })
                  }
                  className={classes(css.popOutIcon, 'popOutButton')}
                >
                  <Tooltip title='Read more'>
                    <PopOutIcon style={{ fontSize: 18 }} />
                  </Tooltip>
                </Button>
              </div>
              {description
                .split('\n')
                .slice(0, 2)
                .map((line, i) => (
                  <div
                    key={i}
                    style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                  >
                    {line}
                  </div>
                ))}
              {description.split('\n').length > 2 ? '...' : ''}
            </Paper>
          </div>
          <Toolbar {...runListToolbarProps} />
          <RunListsRouter
            storageState={runStorageState}
            onError={showPageError}
            hideExperimentColumn={true}
            experimentIdMask={experiment.experiment_id}
            refreshCount={runlistRefreshCount}
            selectedIds={selectedIds}
            onSelectionChange={selectionChanged}
            onTabSwitch={onRunTabSwitch}
            {...props}
          />

          <Dialog
            open={recurringRunsManagerOpen}
            classes={{ paper: css.recurringRunsDialog }}
            onClose={recurringRunsManagerClosed}
          >
            <DialogContent>
              <RecurringRunsManager
                {...props}
                experimentId={props.match.params[RouteParams.experimentId]}
              />
            </DialogContent>
            <DialogActions>
              <Button
                id='closeExperimentRecurringRunManagerBtn'
                onClick={recurringRunsManagerClosed}
                color='secondary'
              >
                Close
              </Button>
            </DialogActions>
          </Dialog>
        </div>
      )}
    </div>
  );
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

function getRunInitialToolBarButtons(
  props: PageProps,
  refresh: () => void,
  selectedIds: string[],
): Buttons {
  const buttons = new Buttons(props, refresh);
  buttons
    .newRun(() => props.match.params[RouteParams.experimentId])
    .newRecurringRun(props.match.params[RouteParams.experimentId])
    .compareRuns(() => selectedIds)
    .cloneRun(() => selectedIds, false);
  return buttons;
}
