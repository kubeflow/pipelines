/*
 * Copyright 2018 Google LLC
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
import Button from '@material-ui/core/Button';
import Buttons, { ButtonKeys } from '../lib/Buttons';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import Paper from '@material-ui/core/Paper';
import PopOutIcon from '@material-ui/icons/Launch';
import RecurringRunsManager from './RecurringRunsManager';
import RunListsRouter, { RunListsGroupTab } from './RunListsRouter';
import Toolbar, { ToolbarProps } from '../components/Toolbar';
import Tooltip from '@material-ui/core/Tooltip';
import { ApiExperiment, ExperimentStorageState } from '../apis/experiment';
import { Apis } from '../lib/Apis';
import { Page, PageProps } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, padding } from '../Css';
import { logger } from '../lib/Utils';
import { useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { Redirect } from 'react-router-dom';
import { RunStorageState } from 'src/apis/run';

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
    width: 158,
  },
  recurringRunsDialog: {
    minWidth: 600,
  },
  runStatsCard: {
    width: 270,
  },
});

interface ExperimentDetailsState {
  activeRecurringRunsCount: number;
  experiment: ApiExperiment | null;
  recurringRunsManagerOpen: boolean;
  selectedIds: string[];
  runStorageState: RunStorageState;
  runListToolbarProps: ToolbarProps;
  runlistRefreshCount: number;
}

export class ExperimentDetails extends Page<{}, ExperimentDetailsState> {
  constructor(props: any) {
    super(props);

    this.state = {
      activeRecurringRunsCount: 0,
      experiment: null,
      recurringRunsManagerOpen: false,
      runListToolbarProps: {
        actions: this._getRunInitialToolBarButtons().getToolbarActionMap(),
        breadcrumbs: [],
        pageTitle: 'Runs',
        topLevelToolbar: false,
      },
      // TODO: remove
      selectedIds: [],
      runStorageState: RunStorageState.AVAILABLE,
      runlistRefreshCount: 0,
    };
  }

  private _getRunInitialToolBarButtons(): Buttons {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    buttons
      .newRun(() => this.props.match.params[RouteParams.experimentId])
      .newRecurringRun(this.props.match.params[RouteParams.experimentId])
      .compareRuns(() => this.state.selectedIds)
      .cloneRun(() => this.state.selectedIds, false);
    return buttons;
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons.refresh(this.refresh.bind(this)).getToolbarActionMap(),
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      // TODO: determine what to show if no props.
      pageTitle: this.props ? this.props.match.params[RouteParams.experimentId] : '',
    };
  }

  public render(): JSX.Element {
    const { activeRecurringRunsCount, experiment } = this.state;
    const description = experiment ? experiment.description || '' : '';

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
                  <div className={css.cardTitle}>Recurring run configs</div>
                  <div
                    className={classes(
                      css.cardContent,
                      !!activeRecurringRunsCount && css.recurringRunsActive,
                    )}
                  >
                    {activeRecurringRunsCount + ' active'}
                  </div>
                  <Button
                    className={css.cardBtn}
                    id='manageExperimentRecurringRunsBtn'
                    disableRipple={true}
                    onClick={() => this.setState({ recurringRunsManagerOpen: true })}
                  >
                    Manage
                  </Button>
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
                      this.props.updateDialog({
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
            <Toolbar {...this.state.runListToolbarProps} />
            <RunListsRouter
              storageState={this.state.runStorageState}
              onError={this.showPageError.bind(this)}
              hideExperimentColumn={true}
              experimentIdMask={experiment.id}
              refreshCount={this.state.runlistRefreshCount}
              selectedIds={this.state.selectedIds}
              onSelectionChange={this._selectionChanged}
              onTabSwitch={this._onRunTabSwitch}
              {...this.props}
            />

            <Dialog
              open={this.state.recurringRunsManagerOpen}
              classes={{ paper: css.recurringRunsDialog }}
              onClose={this._recurringRunsManagerClosed.bind(this)}
            >
              <DialogContent>
                <RecurringRunsManager
                  {...this.props}
                  experimentId={this.props.match.params[RouteParams.experimentId]}
                />
              </DialogContent>
              <DialogActions>
                <Button
                  id='closeExperimentRecurringRunManagerBtn'
                  onClick={this._recurringRunsManagerClosed.bind(this)}
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

  public async refresh(): Promise<void> {
    await this.load();
    return;
  }

  public async componentDidMount(): Promise<void> {
    return this.load(true);
  }

  public async load(isFirstTimeLoad: boolean = false): Promise<void> {
    this.clearBanner();

    const experimentId = this.props.match.params[RouteParams.experimentId];

    try {
      const experiment = await Apis.experimentServiceApi.getExperiment(experimentId);
      const pageTitle = experiment.name || this.props.match.params[RouteParams.experimentId];

      // Update the Archive/Restore button based on the storage state of this experiment.
      const buttons = new Buttons(
        this.props,
        this.refresh.bind(this),
        this.getInitialToolbarState().actions,
      );
      const idGetter = () => (experiment.id ? [experiment.id] : []);
      experiment.storage_state === ExperimentStorageState.ARCHIVED
        ? buttons.restore('experiment', idGetter, true, () => this.refresh())
        : buttons.archive('experiment', idGetter, true, () => this.refresh());
      // If experiment is archived, shows archived runs list by default.
      // If experiment is active, shows active runs list by default.
      let runStorageState = this.state.runStorageState;
      // Determine the default Active/Archive run list tab based on experiment status.
      // After component is mounted, it is up to user to decide the run storage state they
      // want to view.
      if (isFirstTimeLoad) {
        runStorageState =
          experiment.storage_state === ExperimentStorageState.ARCHIVED
            ? RunStorageState.ARCHIVED
            : RunStorageState.AVAILABLE;
      }

      const actions = buttons.getToolbarActionMap();
      this.props.updateToolbar({
        actions,
        breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
        pageTitle,
        pageTitleTooltip: pageTitle,
      });

      let activeRecurringRunsCount = -1;

      // Fetch this experiment's jobs
      try {
        // TODO: get ALL jobs in the experiment
        const recurringRuns = await Apis.jobServiceApi.listJobs(
          undefined,
          100,
          '',
          'EXPERIMENT',
          experimentId,
        );
        activeRecurringRunsCount = (recurringRuns.jobs || []).filter(j => j.enabled === true)
          .length;
      } catch (err) {
        await this.showPageError(
          `Error: failed to retrieve recurring runs for experiment: ${experimentId}.`,
          err,
        );
        logger.error(`Error fetching recurring runs for experiment: ${experimentId}`, err);
      }

      let runlistRefreshCount = this.state.runlistRefreshCount + 1;
      this.setStateSafe({
        activeRecurringRunsCount,
        experiment,
        runStorageState,
        runlistRefreshCount,
      });
      this._selectionChanged([]);
    } catch (err) {
      await this.showPageError(`Error: failed to retrieve experiment: ${experimentId}.`, err);
      logger.error(`Error loading experiment: ${experimentId}`, err);
    }
  }

  /**
   * Users can choose to show runs list in different run storage states.
   *
   * @param tab selected by user for run storage state
   */
  _onRunTabSwitch = (tab: RunListsGroupTab) => {
    let runStorageState = RunStorageState.AVAILABLE;
    if (tab === RunListsGroupTab.ARCHIVE) {
      runStorageState = RunStorageState.ARCHIVED;
    }
    let runlistRefreshCount = this.state.runlistRefreshCount + 1;
    this.setStateSafe(
      {
        runStorageState,
        runlistRefreshCount,
      },
      () => {
        this._selectionChanged([]);
      },
    );

    return;
  };

  _selectionChanged = (selectedIds: string[]) => {
    const toolbarButtons = this._getRunInitialToolBarButtons();
    // If user selects to show Active runs list, shows `Archive` button for selected runs.
    // If user selects to show Archive runs list, shows `Restore` button for selected runs.
    if (this.state.runStorageState === RunStorageState.AVAILABLE) {
      toolbarButtons.archive(
        'run',
        () => this.state.selectedIds,
        false,
        ids => this._selectionChanged(ids),
      );
    } else {
      toolbarButtons.restore(
        'run',
        () => this.state.selectedIds,
        false,
        ids => this._selectionChanged(ids),
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
    this.setState({
      runListToolbarProps: {
        actions: toolbarActions,
        breadcrumbs: this.state.runListToolbarProps.breadcrumbs,
        pageTitle: this.state.runListToolbarProps.pageTitle,
        topLevelToolbar: this.state.runListToolbarProps.topLevelToolbar,
      },
      selectedIds,
    });
  };

  private _recurringRunsManagerClosed(): void {
    this.setState({ recurringRunsManagerOpen: false });
    // Reload the details to get any updated recurring runs
    this.refresh();
  }
}

const EnhancedExperimentDetails: React.FC<PageProps> = props => {
  // When namespace changes, this experiment no longer belongs to new namespace.
  // So we redirect to experiment list page instead.
  const namespaceChanged = useNamespaceChangeEvent();
  if (namespaceChanged) {
    return <Redirect to={RoutePage.EXPERIMENTS} />;
  }

  return <ExperimentDetails {...props} />;
};

export default EnhancedExperimentDetails;
