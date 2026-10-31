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

import 'src/pages/DetailLayouts.css';
import { RecurringRunOverview } from 'src/components/runs/RecurringRunOverview';
import type * as React from 'react';
import Buttons, { ButtonKeys } from 'src/lib/Buttons';
import 'src/components/runs/ExperimentWorkflows.css';
import { V2Experiment } from 'src/apisv2/experiment';
import { V2RecurringRun, V2RecurringRunStatus } from 'src/apisv2/recurringrun';
import { Apis } from 'src/lib/Apis';
import { Page } from './Page';
import { RoutePage, RouteParams } from 'src/components/Router';
import { Breadcrumb, ToolbarProps } from 'src/lib/PageChromeTypes';
import { errorToMessage } from 'src/lib/Utils';

interface RecurringRunConfigState {
  run: V2RecurringRun | null;
}

class RecurringRunDetailsView extends Page<{}, RecurringRunConfigState> {
  constructor(props: any) {
    super(props);

    this.state = {
      run: null,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons
        .cloneRecurringRun(() => (this.state.run ? [this.state.run.recurring_run_id!] : []), true)
        .refresh(this.refresh.bind(this))
        .enableRecurringRun(() => (this.state.run ? this.state.run.recurring_run_id! : ''))
        .disableRecurringRun(() => (this.state.run ? this.state.run.recurring_run_id! : ''))
        .delete(
          () => (this.state.run ? [this.state.run!.recurring_run_id!] : []),
          'recurring run config',
          this._deleteCallback.bind(this),
          true /* useCurrentResource */,
        )
        .getToolbarActionMap(),
      breadcrumbs: [],
      pageTitle: '',
    };
  }

  public render(): React.JSX.Element {
    return this.state.run ? <RecurringRunOverview run={this.state.run} /> : <div />;
  }

  public componentDidMount(): Promise<void> {
    this._isMounted = true;
    return this.load();
  }

  public async refresh(): Promise<void> {
    await this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();
    const recurringRunId = this.props.params[RouteParams.recurringRunId] ?? '';

    let run: V2RecurringRun;
    try {
      run = await Apis.recurringRunServiceApi.getRecurringRun(recurringRunId);
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      await this.showPageError(
        `Error: failed to retrieve recurring run: ${recurringRunId}.`,
        new Error(errorMessage),
      );
      return;
    }

    const relatedExperimentId = run.experiment_id;
    let experiment: V2Experiment | undefined;
    if (relatedExperimentId) {
      try {
        experiment = await Apis.experimentServiceApiV2.getExperiment(relatedExperimentId);
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        await this.showPageError(
          `Error: failed to retrieve this recurring run's experiment.`,
          new Error(errorMessage),
          'warning',
        );
      }
    }
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
    const pageTitle = run ? run.display_name! : recurringRunId;

    const toolbarActions = this.props.toolbarProps.actions;
    toolbarActions[ButtonKeys.ENABLE_RECURRING_RUN].disabled =
      run.status === V2RecurringRunStatus.ENABLED;
    toolbarActions[ButtonKeys.DISABLE_RECURRING_RUN].disabled =
      run.status !== V2RecurringRunStatus.ENABLED;

    if (!this._isMounted) {
      return;
    }

    this.props.updateToolbar({ actions: toolbarActions, breadcrumbs, pageTitle });

    this.setStateSafe({ run });
  }

  private _deleteCallback(_: string[], success: boolean): void {
    if (success) {
      const breadcrumbs = this.props.toolbarProps.breadcrumbs;
      const previousPage = breadcrumbs.length
        ? breadcrumbs[breadcrumbs.length - 1].href
        : RoutePage.EXPERIMENTS;
      this.props.navigate(previousPage);
    }
  }
}

export default RecurringRunDetailsView;
