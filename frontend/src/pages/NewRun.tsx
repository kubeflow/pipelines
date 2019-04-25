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
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Input from '../atoms/Input';
import InputAdornment from '@material-ui/core/InputAdornment';
import Radio from '@material-ui/core/Radio';
import RunUtils from '../lib/RunUtils';
import ResourceSelector from './ResourceSelector';
import TextField, { TextFieldProps } from '@material-ui/core/TextField';
import Trigger from '../components/Trigger';
import WorkflowParser from '../lib/WorkflowParser';
import { ApiExperiment } from '../apis/experiment';
import { ApiPipeline } from '../apis/pipeline';
import { ApiRun, ApiResourceReference, ApiRelationship, ApiResourceType, ApiRunDetail } from '../apis/run';
import { ApiTrigger, ApiJob } from '../apis/job';
import { Apis, PipelineSortKeys, ExperimentSortKeys } from '../lib/Apis';
import { Link } from 'react-router-dom';
import { Page } from './Page';
import { RoutePage, RouteParams, QUERY_PARAMS } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser } from '../lib/URLParser';
import { Workflow } from '../../../frontend/third_party/argo-ui/argo_template';
import { classes, stylesheet } from 'typestyle';
import { commonCss, padding, color } from '../Css';
import { logger, errorToMessage } from '../lib/Utils';

interface NewRunState {
  description: string;
  errorMessage: string;
  experiment?: ApiExperiment;
  experimentName: string;
  experimentSelectorOpen: boolean;
  isBeingStarted: boolean;
  isFirstRunInExperiment: boolean;
  isRecurringRun: boolean;
  maxConcurrentRuns?: string;
  pipeline?: ApiPipeline;
  // This represents a pipeline from a run that is being cloned, or if a user is creating a run from
  // a pipeline that was not uploaded to the system (as in the case of runs created from notebooks).
  // By storing this here instead of in the 'pipeline' field, we won't lose it if the user selects a
  // different pipeline.
  pipelineFromRun?: ApiPipeline;
  // TODO: this is only here to properly display the name in the text field.
  // There is definitely a way to do this that doesn't necessitate this being in state.
  // Note: this cannot be undefined/optional or the label animation for the input field will not
  // work properly.
  pipelineName: string;
  pipelineSelectorOpen: boolean;
  runName: string;
  trigger?: ApiTrigger;
  unconfirmedSelectedExperiment?: ApiExperiment;
  unconfirmedSelectedPipeline?: ApiPipeline;
  usePipelineFromRun: boolean;
  usePipelineFromRunLabel: string;
}

const css = stylesheet({
  nonEditableInput: {
    color: color.secondaryText,
  },
  selectorDialog: {
    minWidth: 680,
  },
});

class NewRun extends Page<{}, NewRunState> {

  private pipelineSelectorColumns = [
    { label: 'Pipeline name', flex: 1, sortKey: PipelineSortKeys.NAME },
    { label: 'Description', flex: 1.5 },
    { label: 'Uploaded on', flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
  ];

  private experimentSelectorColumns = [
    { label: 'Experiment name', flex: 1, sortKey: ExperimentSortKeys.NAME },
    { label: 'Description', flex: 1.5 },
    { label: 'Created at', flex: 1, sortKey: ExperimentSortKeys.CREATED_AT },
  ];

  constructor(props: any) {
    super(props);

    this.state = {
      description: '',
      errorMessage: '',
      experimentName: '',
      experimentSelectorOpen: false,
      isBeingStarted: false,
      isFirstRunInExperiment: false,
      isRecurringRun: false,
      pipelineName: '',
      pipelineSelectorOpen: false,
      runName: '',
      usePipelineFromRun: false,
      usePipelineFromRunLabel: 'Use pipeline from cloned run',
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: [],
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'Start a new run',
    };
  }

  public render(): JSX.Element {
    const {
      pipelineFromRun,
      description,
      errorMessage,
      experimentName,
      experimentSelectorOpen,
      isRecurringRun,
      isFirstRunInExperiment,
      pipeline,
      pipelineName,
      pipelineSelectorOpen,
      runName,
      unconfirmedSelectedExperiment,
      unconfirmedSelectedPipeline,
      usePipelineFromRun,
      usePipelineFromRunLabel,
    } = this.state;

    const originalRunId = new URLParser(this.props).get(QUERY_PARAMS.cloneFromRun);
    const pipelineDetailsUrl = originalRunId ?
      RoutePage.PIPELINE_DETAILS.replace(':' + RouteParams.pipelineId + '?', '') +
      new URLParser(this.props).build({ [QUERY_PARAMS.fromRunId]: originalRunId }) : '';

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <div className={commonCss.scrollContainer}>

          <div className={commonCss.header}>Run details</div>

          {!!pipelineFromRun && (<React.Fragment>
            <FormControlLabel label={usePipelineFromRunLabel} control={<Radio color='primary' />}
              onChange={() => this.setStateSafe({ pipeline: pipelineFromRun, usePipelineFromRun: true })}
              checked={usePipelineFromRun} />
            {!!originalRunId && <Link to={pipelineDetailsUrl}>[View pipeline]</Link>}
            <FormControlLabel label='Select a pipeline from list' control={<Radio color='primary' />}
              onChange={() => this.setStateSafe({ pipeline: undefined, usePipelineFromRun: false })}
              checked={!usePipelineFromRun} />
          </React.Fragment>)}
          {!usePipelineFromRun && (
            <Input value={pipelineName} required={true} label='Pipeline' disabled={true}
              variant='outlined'
              InputProps={{
                classes: { disabled: css.nonEditableInput },
                endAdornment: (
                  <InputAdornment position='end'>
                    <Button color='secondary' id='choosePipelineBtn'
                      onClick={() => this.setStateSafe({ pipelineSelectorOpen: true })}
                      style={{ padding: '3px 5px', margin: 0 }}>
                      Choose
                    </Button>
                  </InputAdornment>
                ),
                readOnly: true,
              }} />
          )}

          <Dialog open={pipelineSelectorOpen}
            classes={{ paper: css.selectorDialog }}
            onClose={() => this._pipelineSelectorClosed(false)}
            PaperProps={{ id: 'pipelineSelectorDialog' }}>
            <DialogContent>
              <ResourceSelector {...this.props}
                title='Choose a pipeline'
                filterLabel='Filter pipelines'
                listApi={async (...args) => {
                  const response = await Apis.pipelineServiceApi.listPipelines(...args);
                  return { resources: response.pipelines || [], nextPageToken: response.next_page_token || '' };
                }}
                columns={this.pipelineSelectorColumns}
                emptyMessage='No pipelines found. Upload a pipeline and then try again.'
                initialSortColumn={PipelineSortKeys.CREATED_AT}
                selectionChanged={(selectedPipeline: ApiPipeline) =>
                  this.setStateSafe({ unconfirmedSelectedPipeline: selectedPipeline })} />
            </DialogContent>
            <DialogActions>
              <Button id='cancelPipelineSelectionBtn' onClick={() => this._pipelineSelectorClosed(false)} color='secondary'>
                Cancel
              </Button>
              <Button id='usePipelineBtn' onClick={() => this._pipelineSelectorClosed(true)}
                color='secondary' disabled={!unconfirmedSelectedPipeline}>
                Use this pipeline
              </Button>
            </DialogActions>
          </Dialog>

          <Dialog open={experimentSelectorOpen}
            classes={{ paper: css.selectorDialog }}
            onClose={() => this._experimentSelectorClosed(false)}
            PaperProps={{ id: 'experimentSelectorDialog' }}>
            <DialogContent>
              <ResourceSelector {...this.props}
                title='Choose an experiment'
                filterLabel='Filter experiments'
                listApi={async (...args) => {
                  const response = await Apis.experimentServiceApi.listExperiment(...args);
                  return { resources: response.experiments || [], nextPageToken: response.next_page_token || '' };
                }}
                columns={this.experimentSelectorColumns}
                emptyMessage='No experiments found. Create an experiment and then try again.'
                initialSortColumn={ExperimentSortKeys.CREATED_AT}
                selectionChanged={(selectedExperiment: ApiExperiment) =>
                  this.setStateSafe({ unconfirmedSelectedExperiment: selectedExperiment })} />
            </DialogContent>
            <DialogActions>
              <Button id='cancelExperimentSelectionBtn' onClick={() => this._experimentSelectorClosed(false)} color='secondary'>
                Cancel
              </Button>
              <Button id='useExperimentBtn' onClick={() => this._experimentSelectorClosed(true)}
                color='secondary' disabled={!unconfirmedSelectedExperiment}>
                Use this experiment
              </Button>
            </DialogActions>
          </Dialog>

          <Input label='Run name' required={true} onChange={this.handleChange('runName')}
            autoFocus={true} value={runName} variant='outlined' />
          <Input label='Description (optional)' multiline={true}
            onChange={this.handleChange('description')} value={description} variant='outlined' />

          <div>This run will be associated with the following experiment</div>
          <Input value={experimentName} required={true} label='Experiment' disabled={true}
            variant='outlined'
            InputProps={{
              classes: { disabled: css.nonEditableInput },
              endAdornment: (
                <InputAdornment position='end'>
                  <Button color='secondary' id='chooseExperimentBtn'
                    onClick={() => this.setStateSafe({ experimentSelectorOpen: true })}
                    style={{ padding: '3px 5px', margin: 0 }}>
                    Choose
                  </Button>
                </InputAdornment>
              ),
              readOnly: true,
            }} />

          {isRecurringRun && (
            <React.Fragment>
              <div className={commonCss.header}>Run trigger</div>

              <Trigger onChange={(trigger, maxConcurrentRuns) => this.setStateSafe({
                maxConcurrentRuns,
                trigger,
              }, this._validate.bind(this))} />
            </React.Fragment>
          )}

          <div className={commonCss.header}>Run parameters</div>
          <div>{this._runParametersMessage(pipeline)}</div>

          {pipeline && Array.isArray(pipeline.parameters) && !!pipeline.parameters.length && (
            <div>
              {pipeline.parameters.map((param, i) =>
                <TextField id={`newRunPipelineParam${i}`} key={i} variant='outlined'
                  label={param.name} value={param.value || ''}
                  onChange={(ev) => this._handleParamChange(i, ev.target.value || '')}
                  style={{ height: 40, maxWidth: 600 }} className={commonCss.textField} />)}
            </div>
          )}

          <div className={classes(commonCss.flex, padding(20, 'tb'))}>
            <BusyButton id='startNewRunBtn' disabled={!!errorMessage}
              busy={this.state.isBeingStarted}
              className={commonCss.buttonAction} title='Start'
              onClick={this._start.bind(this)} />
            <Button id='exitNewRunPageBtn' onClick={() => {
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
    // If we are not cloning, we may have an embedded pipeline from a run from a Notebook
    const embeddedPipelineRunId = urlParser.get(QUERY_PARAMS.fromRunId);
    if (originalRunId) {
      try {
        const originalRun = await Apis.runServiceApi.getRun(originalRunId);
        await this._prepareFormFromClone(originalRun);
        // If the querystring did not contain an experiment ID, try to get one from the run.
        if (!experimentId) {
          experimentId = RunUtils.getFirstExperimentReferenceId(originalRun.run);
        }
      } catch (err) {
        await this.showPageError(`Error: failed to retrieve original run: ${originalRunId}.`, err);
        logger.error(`Failed to retrieve original run: ${originalRunId}`, err);
      }
    } else if(embeddedPipelineRunId) {
      this._prepareFormFromEmbeddedPipeline(embeddedPipelineRunId);
    } else {
      // Get pipeline id from querystring if any
      const possiblePipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
      if (possiblePipelineId) {
        try {
          const pipeline = await Apis.pipelineServiceApi.getPipeline(possiblePipelineId);
          this.setStateSafe({ pipeline, pipelineName: (pipeline && pipeline.name) || '' });
        } catch (err) {
          urlParser.clear(QUERY_PARAMS.pipelineId);
          await this.showPageError(
            `Error: failed to retrieve pipeline: ${possiblePipelineId}.`, err);
          logger.error(`Failed to retrieve pipeline: ${possiblePipelineId}`, err);
        }
      }
    }

    let experiment: ApiExperiment | undefined;
    let experimentName = '';
    const breadcrumbs = [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }];
    if (experimentId) {
      try {
        experiment = await Apis.experimentServiceApi.getExperiment(experimentId);
        experimentName = experiment.name || '';
        breadcrumbs.push({
          displayName: experimentName!,
          href: RoutePage.EXPERIMENT_DETAILS.replace(':' + RouteParams.experimentId, experimentId),
        });
      } catch (err) {
        await this.showPageError(
          `Error: failed to retrieve associated experiment: ${experimentId}.`, err);
        logger.error(`Failed to retrieve associated experiment: ${experimentId}`, err);
      }
    }

    const isRecurringRun = urlParser.get(QUERY_PARAMS.isRecurring) === '1';
    const pageTitle = isRecurringRun ? 'Start a recurring run' : 'Start a new run';
    this.props.updateToolbar({ actions: this.props.toolbarProps.actions, breadcrumbs, pageTitle });

    this.setStateSafe({
      experiment,
      experimentName,
      isFirstRunInExperiment: urlParser.get(QUERY_PARAMS.firstRunInExperiment) === '1',
      isRecurringRun,
    });

    this._validate();
  }

  public handleChange = (name: string) => (event: any) => {
    const value = (event.target as TextFieldProps).value;
    this.setStateSafe({ [name]: value, } as any, () => { this._validate(); });
  }

  protected async _experimentSelectorClosed(confirmed: boolean): Promise<void> {
    let { experiment } = this.state;
    if (confirmed && this.state.unconfirmedSelectedExperiment) {
      experiment = this.state.unconfirmedSelectedExperiment;
    }

    this.setStateSafe({
      experiment,
      experimentName: (experiment && experiment.name) || '',
      experimentSelectorOpen: false
    });
  }

  protected async _pipelineSelectorClosed(confirmed: boolean): Promise<void> {
    let { pipeline } = this.state;
    if (confirmed && this.state.unconfirmedSelectedPipeline) {
      pipeline = this.state.unconfirmedSelectedPipeline;
    }

    this.setStateSafe({
      pipeline,
      pipelineName: (pipeline && pipeline.name) || '',
      pipelineSelectorOpen: false
    }, () => this._validate());
  }

  private async _prepareFormFromEmbeddedPipeline(embeddedPipelineRunId: string): Promise<void> {
    let embeddedPipelineSpec: string | null;

    try {
      const runWithEmbeddedPipeline = await Apis.runServiceApi.getRun(embeddedPipelineRunId);
      embeddedPipelineSpec = RunUtils.getPipelineSpec(runWithEmbeddedPipeline.run);
    } catch (err) {
      await this.showPageError(
        `Error: failed to retrieve the specified run: ${embeddedPipelineRunId}.`, err);
      logger.error(`Failed to retrieve the specified run: ${embeddedPipelineRunId}`, err);
      return;
    }

    if (!embeddedPipelineSpec) {
      await this.showPageError(
        `Error: somehow the run provided in the query params: ${embeddedPipelineRunId} had no embedded pipeline.`);
      return;
    }

    try {
      const pipeline = JSON.parse(embeddedPipelineSpec);
      this.setStateSafe({
        pipeline,
        pipelineFromRun: pipeline,
        pipelineName: (pipeline && pipeline.name) || '',
        usePipelineFromRun: true,
        usePipelineFromRunLabel: 'Use pipeline from previous step',
      });
    } catch (err) {
      await this.showPageError(
        `Error: failed to parse the embedded pipeline's spec: ${embeddedPipelineSpec}.`, err);
      logger.error(`Failed to parse the embedded pipeline's spec from run: ${embeddedPipelineRunId}`, err);
      return;
    }

    this._validate();
  }

  private async _prepareFormFromClone(originalRun: ApiRunDetail): Promise<void> {
    if (!originalRun.run) {
      logger.error('Could not get cloned run details');
      return;
    }

    let pipeline: ApiPipeline;
    let workflow: Workflow;
    let pipelineFromRun: ApiPipeline;
    let usePipelineFromRun = false;
    let usePipelineFromRunLabel = '';

    // This corresponds to a run using a pipeline that has been uploaded
    const referencePipelineId = RunUtils.getPipelineId(originalRun.run);
    // This corresponds to a run where the pipeline has not been uploaded, such as runs started from
    // the CLI or notebooks
    const embeddedPipelineSpec = RunUtils.getPipelineSpec(originalRun.run);
    if (referencePipelineId) {
      try {
        pipeline = await Apis.pipelineServiceApi.getPipeline(referencePipelineId);
      } catch (err) {
        await this.showPageError(
          'Error: failed to find a pipeline corresponding to that of the original run:'
          + ` ${originalRun.run.id}.`, err);
        return;
      }
    } else if (embeddedPipelineSpec) {
      try {
        pipeline = JSON.parse(embeddedPipelineSpec);
        pipelineFromRun = pipeline;
      } catch (err) {
        await this.showPageError('Error: failed to read the clone run\'s pipeline definition.', err);
        return;
      }
      usePipelineFromRun = true;
      usePipelineFromRunLabel = 'Use pipeline from cloned run';
    } else {
      await this.showPageError('Could not find the cloned run\'s pipeline definition.');
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
      await this.showPageError('Error: failed to parse the original run\'s runtime.', err);
      logger.error(originalRun.pipeline_runtime!.workflow_manifest);
      return;
    }

    // Set pipeline parameter values from run's workflow
    pipeline.parameters = WorkflowParser.getParameters(workflow);

    this.setStateSafe({
      pipeline,
      pipelineFromRun: pipelineFromRun!,
      pipelineName: (pipeline && pipeline.name) || '',
      runName: this._getCloneName(originalRun.run.name!),
      usePipelineFromRun,
      usePipelineFromRunLabel,
    });

    this._validate();
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

  private _start(): void {
    const { pipelineFromRun, pipeline, usePipelineFromRun } = this.state;
    // TODO: This cannot currently be reached because _validate() is called everywhere and blocks
    // the button from being clicked without first having a pipeline.
    if (!pipeline) {
      this.showErrorDialog('Run creation failed', 'Cannot start run without pipeline');
      logger.error('Cannot start run without pipeline');
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
    let newRun: ApiRun | ApiJob = {
      description: this.state.description,
      name: this.state.runName,
      pipeline_spec: {
        parameters: pipeline.parameters,
        pipeline_id: usePipelineFromRun ? undefined : pipeline.id,
        workflow_manifest: usePipelineFromRun ? JSON.stringify(pipelineFromRun) : undefined,
      },
      resource_references: references,
    };
    if (isRecurringRun) {
      newRun = Object.assign(newRun, {
        enabled: true,
        max_concurrency: this.state.maxConcurrentRuns || '1',
        trigger: this.state.trigger,
      });
    }

    this.setStateSafe({ isBeingStarted: true }, async () => {
      // TODO: there was previously a bug here where the await wasn't being applied to the API
      // calls, so a run creation could fail, and the success path would still be taken. We need
      // tests for this and other similar situations.
      try {
        isRecurringRun
          ? await Apis.jobServiceApi.createJob(newRun)
          : await Apis.runServiceApi.createRun(newRun);
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        this.showErrorDialog('Run creation failed', errorMessage);
        logger.error('Error creating Run:', err);
        return;
      } finally {
        this.setStateSafe({ isBeingStarted: false });
      }

      if (this.state.experiment) {
        this.props.history.push(
          RoutePage.EXPERIMENT_DETAILS.replace(
            ':' + RouteParams.experimentId, this.state.experiment.id!));
      } else {
        this.props.history.push(RoutePage.RUNS);
      }
      this.props.updateSnackbar({
        message: `Successfully started new Run: ${newRun.name}`,
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
    this.setStateSafe({ pipeline });
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
        const validMaxConcurrentRuns = (input: string) =>
          !isNaN(Number.parseInt(input, 10)) && +input > 0;

        if (maxConcurrentRuns !== undefined && !validMaxConcurrentRuns(maxConcurrentRuns)) {
          throw new Error('For triggered runs, maximum concurrent runs must be a positive number');
        }
      }

      this.setStateSafe({ errorMessage: '' });
    } catch (err) {
      this.setStateSafe({ errorMessage: err.message });
    }
  }
}

export default NewRun;
