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
import BusyButton from '../atoms/BusyButton';
import Button from '@material-ui/core/Button';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import Input from '../atoms/Input';
import InputAdornment from '@material-ui/core/InputAdornment';
import PipelineSelector from './PipelineSelector';
import RunUtils from '../lib/RunUtils';
import TextField, { TextFieldProps } from '@material-ui/core/TextField';
import Trigger from '../components/Trigger';
import WorkflowParser from '../lib/WorkflowParser';
import { ApiExperiment } from '../apis/experiment';
import { ApiPipeline } from '../apis/pipeline';
import { ApiRun, ApiResourceReference, ApiRelationship, ApiResourceType, ApiRunDetail } from '../apis/run';
import { ApiTrigger, ApiJob } from '../apis/job';
import { Apis } from '../lib/Apis';
import { Page } from './Page';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { URLParser, QUERY_PARAMS } from '../lib/URLParser';
import { Workflow } from '../../../frontend/third_party/argo-ui/argo_template';
import { classes, stylesheet } from 'typestyle';
import { commonCss, padding } from '../Css';
import { logger, errorToMessage } from '../lib/Utils';

interface NewRunState {
  description: string;
  errorMessage: string;
  experiment?: ApiExperiment;
  experimentName?: string;
  isBeingCreated: boolean;
  isFirstRunInExperiment: boolean;
  isRecurringRun: boolean;
  maxConcurrentRuns?: string;
  pipeline?: ApiPipeline;
  // TODO: this is only here to properly display the name in the text field.
  // There is definitely a way to do this that doesn't necessitate this being in state.
  pipelineName: string;
  pipelineSelectorOpen: boolean;
  runName: string;
  trigger?: ApiTrigger;
  unconfirmedDialogPipelineId: string;
}

const css = stylesheet({
  pipelineSelectorDialog: {
    minWidth: 680,
  },
});

class NewRun extends Page<{}, NewRunState> {

  constructor(props: any) {
    super(props);

    this.state = {
      description: '',
      errorMessage: '',
      isBeingCreated: false,
      isFirstRunInExperiment: false,
      isRecurringRun: false,
      pipelineName: '',
      pipelineSelectorOpen: false,
      runName: '',
      unconfirmedDialogPipelineId: '',
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [],
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        { displayName: 'Start a new run', href: '' }
      ]
    };
  }

  public render(): JSX.Element {
    const {
      errorMessage,
      experimentName,
      isRecurringRun,
      isFirstRunInExperiment,
      pipeline,
      pipelineSelectorOpen,
      unconfirmedDialogPipelineId,
    } = this.state;

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <div className={commonCss.scrollContainer}>

          <div className={commonCss.header}>Run details</div>

          <Input field='pipelineName' instance={this} required={true} label='Pipeline' disabled={true}
            InputProps={{
              endAdornment: (
                <InputAdornment position='end'>
                  <Button color='secondary' id='choosePipelineBtn'
                    onClick={() => this.setState({ pipelineSelectorOpen: true })}
                    style={{ padding: '3px 5px', margin: 0 }}>
                    Choose
                </Button>
                </InputAdornment>
              ),
              readOnly: true,
            }} />

          <Dialog open={pipelineSelectorOpen} classes={{ paper: css.pipelineSelectorDialog }}
            onClose={() => this._pipelineSelectorClosed(false)} PaperProps={{ id: 'pipelineSelectorDialog' }}>
            <DialogContent>
              <PipelineSelector {...this.props} pipelineSelectionChanged={this._pipelineSelectionChanged.bind(this)} />
            </DialogContent>
            <DialogActions>
              <Button onClick={() => this._pipelineSelectorClosed(false)} color='secondary'>
                Cancel
              </Button>
              <Button id='usePipelineBtn' onClick={() => this._pipelineSelectorClosed(true)}
                color='secondary' disabled={!unconfirmedDialogPipelineId}>
                Use this pipeline
              </Button>
            </DialogActions>
          </Dialog>

          <Input label='Run name' required={true} instance={this} field='runName' autoFocus={true} />
          <Input label='Description (optional)' multiline={true} instance={this}
            field='description' height='auto' />

          {experimentName && (
            <div>
              <div>This run will be associated with the following experiment</div>
              <Input label='Experiment' instance={this} disabled={true} field='experimentName' />
            </div>
          )}

          {isRecurringRun && (
            <React.Fragment>
              <div className={commonCss.header}>Run trigger</div>

              <Trigger onChange={(trigger, maxConcurrentRuns) => this.setState({
                maxConcurrentRuns,
                trigger,
              }, this._validate.bind(this))} />
            </React.Fragment>
          )}

          <div className={commonCss.header}>Run parameters</div>
          <div>{this._runParametersMessage(pipeline)}</div>

          {pipeline && pipeline.parameters && !!pipeline.parameters.length && (
            <div>
              {pipeline && (pipeline.parameters || []).map((param, i) =>
                <TextField key={i} variant='outlined' label={param.name} value={param.value || ''}
                  onChange={(ev) => this._handleParamChange(i, ev.target.value || '')}
                  style={{ height: 40, maxWidth: 600 }} className={commonCss.textField} />)}
            </div>
          )}

          <div className={classes(commonCss.flex, padding(20, 'tb'))}>
            <BusyButton id='createBtn' disabled={!!errorMessage} busy={this.state.isBeingCreated}
              className={commonCss.buttonAction} title='Create'
              onClick={this._create.bind(this)} />
            <Button onClick={() => {
              this.props.history.push(
                !!this.state.experiment
                  ? RoutePage.EXPERIMENT_DETAILS.replace(
                    ':' + RouteParams.experimentId, this.state.experiment.id!)
                  : RoutePage.RUNS);
            }}>
              {isFirstRunInExperiment ? 'Skip this step' : 'Cancel'}
            </Button>
            <div style={{ color: 'red' }}>{errorMessage}</div>
          </div>
        </div>
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();
    const urlParser = new URLParser(this.props);
    let experimentId: string | null = urlParser.get(QUERY_PARAMS.experimentId);

    // Get clone run id from querystring if any
    const originalRunId = urlParser.get(QUERY_PARAMS.cloneFromRun);
    if (originalRunId) {
      try {
        const originalRun = await Apis.runServiceApi.getRun(originalRunId);
        await this._prepareFormFromClone(originalRun);
        // If the querystring did not contain an experiment ID, try to get one from the run.
        if (!experimentId) {
          experimentId = RunUtils.getFirstExperimentReferenceId(originalRun.run);
        }
      } catch (err) {
        await this.showPageError(`Error: failed to get original run: ${originalRunId}.`, err);
      }
    } else {
      // Get pipeline id from querystring if any
      const possiblePipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
      if (possiblePipelineId) {
        try {
          const pipeline = await Apis.pipelineServiceApi.getPipeline(possiblePipelineId);
          this.setState({ pipeline, pipelineName: (pipeline && pipeline.name) || '' });
        } catch (err) {
          urlParser.clear(QUERY_PARAMS.pipelineId);
          await this.showPageError(
            'Error: failed to find a pipeline corresponding to that of the original run:'
            + ` ${originalRunId}.`, err);
        }
      }
    }

    let experiment: ApiExperiment | undefined;
    let experimentName: string | undefined;
    const breadcrumbs = [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }];
    if (experimentId) {
      try {
        experiment = await Apis.experimentServiceApi.getExperiment(experimentId);
        experimentName = experiment.name;
        breadcrumbs.push({
          displayName: experimentName!,
          href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experimentId),
        });
      } catch (err) {
        await this.showPageError(`Error: failed to get associated experiment: ${experimentId}.`, err);
        logger.error(`Failed to get associated experiment ${experimentId}`, err);
      }
    }

    const isRecurringRun = urlParser.get(QUERY_PARAMS.isRecurring) === '1';
    breadcrumbs.push({
      displayName: isRecurringRun ? 'Start a recurring run' : 'Start a new run',
      href: '',
    });
    this.props.updateToolbar({ actions: this.props.toolbarProps.actions, breadcrumbs });

    this.setState({
      experiment,
      experimentName,
      isFirstRunInExperiment: urlParser.get(QUERY_PARAMS.firstRunInExperiment) === '1',
      isRecurringRun,
    });

    this._validate();
  }

  public handleChange = (name: string) => (event: any) => {
    const value = (event.target as TextFieldProps).value;
    this.setState({
      [name]: value,
    } as any, () => {
      // Set querystring if pipeline id has changed
      if (name === 'pipelineId') {
        const urlParser = new URLParser(this.props);
        urlParser.set(QUERY_PARAMS.pipelineId, (value || '').toString());

        // Clear other query params so as not to confuse the user
        urlParser.clear(QUERY_PARAMS.cloneFromRun);
      }

      this._validate();
    });
  }

  private async _prepareFormFromClone(originalRun: ApiRunDetail): Promise<void> {
    const associatedPipelineId = RunUtils.getPipelineId(originalRun.run);
    if (originalRun.run && associatedPipelineId) {
      let pipeline: ApiPipeline;
      let workflow: Workflow;

      try {
        pipeline = await Apis.pipelineServiceApi.getPipeline(associatedPipelineId);
      } catch (err) {
        await this.showPageError(
          'Error: failed to find a pipeline corresponding to that of the original run:'
          + ` ${originalRun.run.id}.`, err);
        return;
      }

      if (originalRun.pipeline_runtime!.workflow_manifest === undefined) {
        await this.showPageError(`Error: run ${originalRun.run.id} had no workflow manifest`);
        logger.error(originalRun.pipeline_runtime!.workflow_manifest);
        return;
      }
      try {
        workflow = JSON.parse(originalRun.pipeline_runtime!.workflow_manifest!) as Workflow;
      } catch (err) {
        await this.showPageError('Error: failed to parse the original run\'s runtime', err);
        logger.error(originalRun.pipeline_runtime!.workflow_manifest);
        return;
      }

      pipeline.parameters = WorkflowParser.getParameters(workflow);

      this.setState({
        pipeline,
        pipelineName: (pipeline && pipeline.name) || '',
        runName: this._getCloneName(originalRun.run.name!)
      });
      return;
    }
  }

  private async _pipelineSelectorClosed(confirmed: boolean): Promise<void> {
    let { pipeline } = this.state;
    if (confirmed && this.state.unconfirmedDialogPipelineId) {
      const pipelineId = this.state.unconfirmedDialogPipelineId;
      try {
        pipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);
      } catch (err) {
        await this.showPageError(`Error: failed to retrieve pipeline with ID: ${pipelineId}`, err);
        logger.error(`Error: failed to retrieve pipeline with ID: ${pipelineId}`, err);
        return;
      }
    }
    this.setState({
      pipeline,
      pipelineName: (pipeline && pipeline.name) || '',
      pipelineSelectorOpen: false
    });

    // Now that we may have a pipeline, update the validation.
    this._validate();
  }

  /* This function is passed as a callback to the PipelineSelector dialog. */
  private _pipelineSelectionChanged(selectedId: string): void {
    this.setState({ unconfirmedDialogPipelineId: selectedId });
  }

  private _runParametersMessage(selectedPipeline: ApiPipeline | undefined): string {
    if (selectedPipeline) {
      if (selectedPipeline.parameters && selectedPipeline.parameters.length) {
        return 'Specify parameters required by the pipeline';
      } else {
        return 'This pipeline has no parameters';
      }
    }
    return 'Parameters will appear after you select a pipeline';
  }

  private _create(): void {
    const { pipeline } = this.state;
    if (!pipeline) {
      this.showErrorDialog('Run creation failed', 'Cannot create run without pipeline');
      logger.error('Cannot create run without pipeline');
      return;
    }
    const references: ApiResourceReference[] = [];
    if (this.state.experiment) {
      references.push({
        key: {
          id: this.state.experiment.id,
          type: ApiResourceType.EXPERIMENT,
        },
        relationship: ApiRelationship.OWNER,
      });
    }

    const isRecurringRun = this.state.isRecurringRun;
    let newRun: ApiRun | ApiJob;
    if (isRecurringRun) {
      newRun = {
        description: this.state.description,
        enabled: true,
        max_concurrency: this.state.maxConcurrentRuns || '1',
        name: this.state.runName,
        pipeline_spec: { parameters: pipeline.parameters, pipeline_id: pipeline.id },
        resource_references: references,
        trigger: this.state.trigger,
      };
    } else {
      newRun = {
        description: this.state.description,
        name: this.state.runName,
        pipeline_spec: { parameters: pipeline.parameters, pipeline_id: pipeline.id },
        resource_references: references,
      };
    }

    this.setState({ isBeingCreated: true }, async () => {
      try {
        await isRecurringRun
          ? Apis.jobServiceApi.createJob(newRun)
          : Apis.runServiceApi.createRun(newRun);
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        this.showErrorDialog('Run creation failed', errorMessage);
        logger.error('Error creating Run:', err);
        this.setState({ isBeingCreated: false });
        return;
      }
      if (this.state.experiment) {
        this.props.history.push(
          RoutePage.EXPERIMENT_DETAILS.replace(
            ':' + RouteParams.experimentId, this.state.experiment.id!));
      } else {
        this.props.history.push(RoutePage.RUNS);
      }
      this.props.updateSnackbar({
        message: `Successfully created new Run: ${newRun.name}`,
        open: true,
      });
    });
  }

  private _handleParamChange(index: number, value: string): void {
    const { pipeline } = this.state;
    if (!pipeline || !pipeline.parameters) {
      return;
    }
    pipeline.parameters[index].value = value;
    // TODO: is this doing anything?
    this.setState({ pipeline });
  }

  private _getCloneName(oldName: string): string {
    const numberRegex = /Clone(?: \(([0-9]*)\))? of (.*)/;
    const match = oldName.match(numberRegex);
    if (!match) { // No match, add Clone prefix
      return 'Clone of ' + oldName;
    } else {
      const cloneNumber = match[1] ? +match[1] : 1;
      return `Clone (${cloneNumber + 1}) of ${match[2]}`;
    }
  }

  private _validate(): void {
    // Validate state
    const { pipeline, maxConcurrentRuns, runName, trigger } = this.state;
    try {
      if (!pipeline) {
        throw new Error('A pipeline must be selected');
      }
      if (!runName) {
        throw new Error('Run name is required');
      }

      const hasTrigger = trigger && (!!trigger.cron_schedule || !!trigger.periodic_schedule);
      if (hasTrigger) {
        const startDate = !!trigger!.cron_schedule ?
          trigger!.cron_schedule!.start_time : trigger!.periodic_schedule!.start_time;
        const endDate = !!trigger!.cron_schedule ?
          trigger!.cron_schedule!.end_time : trigger!.periodic_schedule!.end_time;
        if (startDate && endDate && startDate > endDate) {
          throw new Error('End date/time cannot be earlier than start date/time');
        }
      }
      const validMaxConcurrentRuns = (input: string) =>
        !isNaN(Number.parseInt(input, 10)) && +input > 0;

      if (hasTrigger && maxConcurrentRuns !== undefined &&
        !validMaxConcurrentRuns(maxConcurrentRuns)) {
        throw new Error('For triggered runs, maximum concurrent runs must be a positive number');
      }

      this.setState({ errorMessage: '' });
    } catch (err) {
      this.setState({ errorMessage: err.message });
    }
  }
}

export default NewRun;
