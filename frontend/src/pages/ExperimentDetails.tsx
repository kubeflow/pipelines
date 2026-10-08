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
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import { ExternalLink } from 'lucide-react';
import RecurringRunsManager from './RecurringRunsManager';
import ExperimentRunTabs, { RunListsGroupTab } from './ExperimentRunTabs';
import type { ToolbarProps } from 'src/lib/PageChromeTypes';
import { Toolbar } from 'src/components/shell/PageChrome';
import { V2beta1Experiment, V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { Apis } from 'src/lib/Apis';
import { Page, PageProps } from './Page';
import { RoutePage, RouteParams } from 'src/components/Router';
import { errorToMessage, logger } from 'src/lib/Utils';
import { useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { Navigate } from 'react-router';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';

import { Button } from 'src/components/ui/button';
import { ModalDialog } from 'src/components/ui/dialog';
import 'src/components/runs/ExperimentWorkflows.css';

interface ExperimentDetailsState {
  activeRecurringRunsCount: number;
  experiment: V2beta1Experiment | null;
  recurringRunsManagerOpen: boolean;
  selectedIds: string[];
  runStorageState: V2beta1RunStorageState;
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
      runStorageState: V2beta1RunStorageState.AVAILABLE,
      runlistRefreshCount: 0,
    };
  }

  private _getRunInitialToolBarButtons(): Buttons {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    buttons
      .newRun(() => this.props.params[RouteParams.experimentId] ?? '')
      .newRecurringRun(this.props.params[RouteParams.experimentId] ?? '')
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
      pageTitle: this.props ? (this.props.params[RouteParams.experimentId] ?? '') : '',
    };
  }

  public render(): React.JSX.Element {
    const { activeRecurringRunsCount, experiment } = this.state;
    const description = experiment ? experiment.description || '' : '';

    return (
      <div className='kfp-workflow-page'>
        {experiment && (
          <div className='kfp-workflow-content'>
            <div className='kfp-workflow-cards'>
              <section
                id='recurringRunsCard'
                className='kfp-workflow-card'
                aria-label='Recurring run configs'
              >
                <div className='kfp-workflow-card-heading'>
                  <h2>Recurring run configs</h2>
                  <Button
                    variant='ghost'
                    size='sm'
                    id='manageExperimentRecurringRunsBtn'
                    onClick={() => this.setState({ recurringRunsManagerOpen: true })}
                  >
                    Manage
                  </Button>
                </div>
                <div
                  className='kfp-workflow-card-count'
                  data-active={activeRecurringRunsCount > 0 ? '' : undefined}
                >
                  {activeRecurringRunsCount < 0
                    ? 'Unavailable'
                    : `${activeRecurringRunsCount} active`}
                </div>
              </section>
              <section
                id='experimentDescriptionCard'
                className='kfp-workflow-card'
                aria-label='Experiment description'
              >
                <div className='kfp-workflow-card-heading'>
                  <h2>Experiment description</h2>
                  <Button
                    variant='ghost'
                    size='icon'
                    id='expandExperimentDescriptionBtn'
                    aria-label='Read more'
                    title='Read more'
                    onClick={() =>
                      this.props.updateDialog({
                        content: description,
                        title: 'Experiment description',
                      })
                    }
                  >
                    <ExternalLink aria-hidden='true' />
                  </Button>
                </div>
                {description
                  .split('\n')
                  .slice(0, 2)
                  .map((line, i) => (
                    <div key={i} className='kfp-workflow-description-line'>
                      {line}
                    </div>
                  ))}
                {description.split('\n').length > 2 ? '...' : ''}
              </section>
            </div>
            <Toolbar {...this.state.runListToolbarProps} />
            <ExperimentRunTabs
              storageState={this.state.runStorageState}
              onError={this.showPageError.bind(this)}
              hideExperimentColumn={true}
              experimentIdMask={experiment.experiment_id}
              refreshCount={this.state.runlistRefreshCount}
              selectedIds={this.state.selectedIds}
              onSelectionChange={this._selectionChanged}
              onTabSwitch={this._onRunTabSwitch}
              {...this.props}
            />
            {this.state.recurringRunsManagerOpen && (
              <ModalDialog
                open
                title='Recurring run configs'
                size='lg'
                onClose={this._recurringRunsManagerClosed.bind(this)}
                actions={
                  <Button
                    variant='secondary'
                    id='closeExperimentRecurringRunManagerBtn'
                    onClick={this._recurringRunsManagerClosed.bind(this)}
                  >
                    Close
                  </Button>
                }
              >
                <RecurringRunsManager
                  {...this.props}
                  experimentId={this.props.params[RouteParams.experimentId] ?? ''}
                />
              </ModalDialog>
            )}
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
    this._isMounted = true;
    return this.load(true);
  }

  public async load(isFirstTimeLoad: boolean = false): Promise<void> {
    this.clearBanner();

    const experimentId = this.props.params[RouteParams.experimentId] ?? '';

    try {
      const experiment = await Apis.experimentServiceApiV2.getExperiment(experimentId);
      const pageTitle =
        experiment.display_name || (this.props.params[RouteParams.experimentId] ?? '');

      // Update the Archive/Restore button based on the storage state of this experiment.
      const buttons = new Buttons(
        this.props,
        this.refresh.bind(this),
        this.getInitialToolbarState().actions,
      );
      const idGetter = () => (experiment.experiment_id ? [experiment.experiment_id] : []);
      experiment.storage_state === V2beta1ExperimentStorageState.ARCHIVED
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
          experiment.storage_state === V2beta1ExperimentStorageState.ARCHIVED
            ? V2beta1RunStorageState.ARCHIVED
            : V2beta1RunStorageState.AVAILABLE;
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
        const recurringRuns = await Apis.recurringRunServiceApi.listRecurringRuns(
          undefined,
          100,
          '',
          undefined,
          undefined,
          experimentId,
        );
        activeRecurringRunsCount = (recurringRuns.recurringRuns || []).filter(
          (rr) => rr.status === V2beta1RecurringRunStatus.ENABLED,
        ).length;
      } catch (err) {
        const error = err instanceof Error ? err : new Error(await errorToMessage(err));
        await this.showPageError(
          `Error: failed to retrieve recurring runs for experiment: ${experimentId}.`,
          error,
        );
        logger.error(`Error fetching recurring runs for experiment: ${experimentId}`, err);
      }

      let runlistRefreshCount = this.state.runlistRefreshCount + 1;
      this.setStateSafe(
        {
          activeRecurringRunsCount,
          experiment,
          runStorageState,
          runlistRefreshCount,
        },
        () => {
          this._selectionChanged([]);
        },
      );
    } catch (err) {
      const error = err instanceof Error ? err : new Error(await errorToMessage(err));
      await this.showPageError(`Error: failed to retrieve experiment: ${experimentId}.`, error);
      logger.error(`Error loading experiment: ${experimentId}`, err);
    }
  }

  /**
   * Users can choose to show runs list in different run storage states.
   *
   * @param tab selected by user for run storage state
   */
  _onRunTabSwitch = (tab: RunListsGroupTab) => {
    let runStorageState: V2beta1RunStorageState = V2beta1RunStorageState.AVAILABLE;
    if (tab === RunListsGroupTab.ARCHIVE) {
      runStorageState = V2beta1RunStorageState.ARCHIVED;
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
    if (this.state.runStorageState === V2beta1RunStorageState.AVAILABLE) {
      toolbarButtons.archive(
        'run',
        () => this.state.selectedIds,
        false,
        (ids) => this._selectionChanged(ids),
      );
    } else {
      toolbarButtons.restore(
        'run',
        () => this.state.selectedIds,
        false,
        (ids) => this._selectionChanged(ids),
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
    this.setStateSafe({
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
    this.setStateSafe({ recurringRunsManagerOpen: false });
    // Reload the details to get any updated recurring runs
    if (this._isMounted) {
      this.refresh();
    }
  }
}

const ExperimentDetailsWithContext: React.FC<PageProps> = (props) => {
  // When namespace changes, this experiment no longer belongs to new namespace.
  // So we redirect to experiment list page instead.
  const namespaceChanged = useNamespaceChangeEvent();
  if (namespaceChanged) {
    return <Navigate replace to={RoutePage.EXPERIMENTS} />;
  }

  return <ExperimentDetails {...props} />;
};

export default ExperimentDetailsWithContext;
