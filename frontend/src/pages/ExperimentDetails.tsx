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
import RunList from '../pages/RunList';
import Toolbar, { ToolbarProps } from '../components/Toolbar';
import Tooltip from '@material-ui/core/Tooltip';
import { ApiExperiment, ExperimentStorageState } from '../apis/experiment';
import { Apis } from '../lib/Apis';
import { Page, PageProps } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { RunStorageState } from '../apis/run';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, padding } from '../Css';
import { logger } from '../lib/Utils';
import { useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { Redirect } from 'react-router-dom';
import { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

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
  selectedTab: number;
  runListToolbarProps: ToolbarProps;
}

export class ExperimentDetails extends Page<{ t: TFunction }, ExperimentDetailsState> {
  private _runlistRef = React.createRef<RunList>();

  constructor(props: any) {
    super(props);

    const buttons = new Buttons(this.props, this.refresh.bind(this));
    const { t } = this.props;
    this.state = {
      activeRecurringRunsCount: 0,
      experiment: null,
      recurringRunsManagerOpen: false,
      runListToolbarProps: {
        actions: buttons
          .newRun(() => this.props.match.params[RouteParams.experimentId])
          .newRecurringRun(this.props.match.params[RouteParams.experimentId])
          .compareRuns(() => this.state.selectedIds)
          .cloneRun(() => this.state.selectedIds, false)
          .archive(
            'run',
            () => this.state.selectedIds,
            false,
            ids => this._selectionChanged(ids),
          )
          .getToolbarActionMap(),
        breadcrumbs: [],
        pageTitle: t('runs'),
        topLevelToolbar: false,
        t,
      },
      // TODO: remove
      selectedIds: [],
      selectedTab: 0,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    const { t } = this.props;
    return {
      actions: buttons.refresh(this.refresh.bind(this)).getToolbarActionMap(),
      breadcrumbs: [{ displayName: t('common:experiments'), href: RoutePage.EXPERIMENTS }],
      // TODO: determine what to show if no props.
      pageTitle: this.props ? this.props.match.params[RouteParams.experimentId] : '',
      t,
    };
  }

  public render(): JSX.Element {
    const { activeRecurringRunsCount, experiment } = this.state;
    const description = experiment ? experiment.description || '' : '';
    const { t } = this.props;

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
                  <div className={css.cardTitle}>{t('recurringRunConfigs')}</div>
                  <div
                    className={classes(
                      css.cardContent,
                      !!activeRecurringRunsCount && css.recurringRunsActive,
                    )}
                  >
                    {activeRecurringRunsCount + ` ${t('experiments:active')}`}
                  </div>
                  <Button
                    className={css.cardBtn}
                    id='manageExperimentRecurringRunsBtn'
                    disableRipple={true}
                    onClick={() => this.setState({ recurringRunsManagerOpen: true })}
                  >
                    {t('common:manage')}
                  </Button>
                </div>
              </Paper>
              <Paper
                id='experimentDescriptionCard'
                className={classes(css.card, css.runStatsCard)}
                elevation={0}
              >
                <div className={css.cardTitle}>
                  <span>{t('experimentDescription')}</span>
                  <Button
                    id='expandExperimentDescriptionBtn'
                    onClick={() =>
                      this.props.updateDialog({
                        content: description,
                        title: t('experimentDescription'),
                      })
                    }
                    className={classes(css.popOutIcon, 'popOutButton')}
                  >
                    <Tooltip title={t('common:readMore')}>
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
            <RunList
              onError={this.showPageError.bind(this)}
              hideExperimentColumn={true}
              experimentIdMask={experiment.id}
              ref={this._runlistRef}
              selectedIds={this.state.selectedIds}
              storageState={RunStorageState.AVAILABLE}
              onSelectionChange={this._selectionChanged.bind(this)}
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
                  {t('common:close')}
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
    if (this._runlistRef.current) {
      await this._runlistRef.current.refresh();
    }
    return;
  }

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();

    const experimentId = this.props.match.params[RouteParams.experimentId];
    const { t } = this.props;

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
      const actions = buttons.getToolbarActionMap();
      this.props.updateToolbar({
        actions,
        breadcrumbs: [{ displayName: t('common:experiments'), href: RoutePage.EXPERIMENTS }],
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
          `${t('errorRetrieveRecurrRunsExperiment')}: ${experimentId}.`,
          err,
        );
        logger.error(`Error fetching recurring runs for experiment: ${experimentId}`, err);
      }

      this.setStateSafe({ activeRecurringRunsCount, experiment });
    } catch (err) {
      await this.showPageError(`${t('errorRetrieveExperiment')}: ${experimentId}.`, err);
      logger.error(`Error loading experiment: ${experimentId}`, err);
    }
  }

  private _selectionChanged(selectedIds: string[]): void {
    const { t } = this.props;
    const toolbarActions = this.state.runListToolbarProps.actions;
    toolbarActions[ButtonKeys.COMPARE].disabled =
      selectedIds.length <= 1 || selectedIds.length > 10;
    toolbarActions[ButtonKeys.CLONE_RUN].disabled = selectedIds.length !== 1;
    toolbarActions[ButtonKeys.ARCHIVE].disabled = !selectedIds.length;
    this.setState({
      runListToolbarProps: {
        t,
        actions: toolbarActions,
        breadcrumbs: this.state.runListToolbarProps.breadcrumbs,
        pageTitle: this.state.runListToolbarProps.pageTitle,
        topLevelToolbar: this.state.runListToolbarProps.topLevelToolbar,
      },
      selectedIds,
    });
  }

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
  const { t } = useTranslation(['experiments', 'common']);
  if (namespaceChanged) {
    return <Redirect to={RoutePage.EXPERIMENTS} />;
  }

  return <ExperimentDetails {...props} t={t} />;
};

export default EnhancedExperimentDetails;
