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
import BusyButton from 'src/atoms/BusyButton';
import Button from '@material-ui/core/Button';
import Input from 'src/atoms/Input';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';
import { Apis } from 'src/lib/Apis';
import { Page, PageProps } from 'src/pages/Page';
import { RoutePage, QUERY_PARAMS } from 'src/components/Router';
import { TextFieldProps } from '@material-ui/core/TextField';
import { ToolbarProps } from 'src/components/Toolbar';
import { URLParser } from 'src/lib/URLParser';
import { classes, stylesheet } from 'typestyle';
import { commonCss, padding, fontsize } from 'src/Css';
import { logger, errorToMessage } from 'src/lib/Utils';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { getLatestVersion } from 'src/pages/NewRunV2';
import { NewExperimentFC } from 'src/pages/functional_components/NewExperimentFC';
import { FeatureKey, isFeatureEnabled } from 'src/features';

interface NewExperimentState {
  description: string;
  validationError: string;
  isbeingCreated: boolean;
  experimentName: string;
  pipelineId?: string;
}

const css = stylesheet({
  errorMessage: {
    color: 'red',
  },
  // TODO: move to Css.tsx and probably rename.
  explanation: {
    fontSize: fontsize.small,
  },
});

export class NewExperiment extends Page<{ namespace?: string }, NewExperimentState> {
  private _experimentNameRef = React.createRef<HTMLInputElement>();

  constructor(props: any) {
    super(props);

    this.state = {
      description: '',
      experimentName: '',
      isbeingCreated: false,
      validationError: '',
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'New experiment',
    };
  }

  public render(): JSX.Element {
    const { description, experimentName, isbeingCreated, validationError } = this.state;

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <div className={classes(commonCss.scrollContainer, padding(20, 'lr'))}>
          <div className={commonCss.header}>Experiment details</div>
          {/* TODO: this description needs work. */}
          <div className={css.explanation}>
            Think of an Experiment as a space that contains the history of all pipelines and their
            associated runs
          </div>

          <Input
            id='experimentName'
            label='Experiment name'
            inputRef={this._experimentNameRef}
            required={true}
            onChange={this.handleChange('experimentName')}
            value={experimentName}
            autoFocus={true}
            variant='outlined'
          />
          <Input
            id='experimentDescription'
            label='Description'
            multiline={true}
            onChange={this.handleChange('description')}
            required={false}
            value={description}
            variant='outlined'
          />

          <div className={commonCss.flex}>
            <BusyButton
              id='createExperimentBtn'
              disabled={!!validationError}
              busy={isbeingCreated}
              className={commonCss.buttonAction}
              title={'Next'}
              onClick={this._create.bind(this)}
            />
            <Button
              id='cancelNewExperimentBtn'
              onClick={() => this.props.history.push(RoutePage.EXPERIMENTS)}
            >
              Cancel
            </Button>
            <div className={css.errorMessage}>{validationError}</div>
          </div>
        </div>
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return;
  }

  public async componentDidMount(): Promise<void> {
    const urlParser = new URLParser(this.props);
    const pipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
    if (pipelineId) {
      this.setState({ pipelineId });
    }

    this._validate();
  }

  public handleChange = (name: string) => (event: any) => {
    const value = (event.target as TextFieldProps).value;
    this.setState({ [name]: value } as any, this._validate.bind(this));
  };

  private _create(): void {
    const newExperiment: V2beta1Experiment = {
      description: this.state.description,
      display_name: this.state.experimentName,
      namespace: this.props.namespace,
    };

    this.setState({ isbeingCreated: true }, async () => {
      try {
        const response = await Apis.experimentServiceApiV2.createExperiment(newExperiment);
        let searchString = '';
        if (this.state.pipelineId) {
          const latestVersion = await getLatestVersion(this.state.pipelineId);
          searchString = new URLParser(this.props).build({
            [QUERY_PARAMS.experimentId]: response.experiment_id || '',
            [QUERY_PARAMS.pipelineId]: this.state.pipelineId,
            [QUERY_PARAMS.pipelineVersionId]: latestVersion?.pipeline_version_id || '',
            [QUERY_PARAMS.firstRunInExperiment]: '1',
          });
        } else {
          searchString = new URLParser(this.props).build({
            [QUERY_PARAMS.experimentId]: response.experiment_id || '',
            [QUERY_PARAMS.firstRunInExperiment]: '1',
          });
        }
        this.props.history.push(RoutePage.NEW_RUN + searchString);
        this.props.updateSnackbar({
          autoHideDuration: 10000,
          message: `Successfully created new Experiment: ${newExperiment.display_name}`,
          open: true,
        });
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        await this.showErrorDialog('Experiment creation failed', errorMessage);
        logger.error('Error creating experiment:', err);
        this.setState({ isbeingCreated: false });
      }
    });
  }

  private _validate(): void {
    // Validate state
    const { experimentName } = this.state;
    try {
      if (!experimentName) {
        throw new Error('Experiment name is required');
      }
      this.setState({ validationError: '' });
    } catch (err) {
      this.setState({ validationError: err.message });
    }
  }
}

const EnhancedNewExperiment: React.FC<PageProps> = props => {
  const namespace = React.useContext(NamespaceContext);
  return isFeatureEnabled(FeatureKey.FUNCTIONAL_COMPONENT) ? (
    <NewExperimentFC {...props} namespace={namespace} />
  ) : (
    <NewExperiment {...props} namespace={namespace} />
  );
};

export default EnhancedNewExperiment;
