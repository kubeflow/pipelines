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
import BusyButton from '../atoms/BusyButton';
import Button from '@material-ui/core/Button';
import Input from '../atoms/Input';
import { ApiExperiment, ApiResourceType, ApiRelationship } from '../apis/experiment';
import { Apis } from '../lib/Apis';
import { Page, PageProps } from './Page';
import { RoutePage, QUERY_PARAMS } from '../components/Router';
import { TextFieldProps } from '@material-ui/core/TextField';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser } from '../lib/URLParser';
import { classes, stylesheet } from 'typestyle';
import { commonCss, padding, fontsize } from '../Css';
import { logger, errorToMessage } from '../lib/Utils';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import i18n from 'i18next'

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
      pageTitle: i18n.t('NewExperiment.newExperiment'),
    };
  }

  public render(): JSX.Element {
    const { description, experimentName, isbeingCreated, validationError } = this.state;

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <div className={classes(commonCss.scrollContainer, padding(20, 'lr'))}>
          <div className={commonCss.header}>{i18n.t('NewExperiment.experimentDetails')}</div>
          {/* TODO: this description needs work. */}
          <div className={css.explanation}>
            {i18n.t('NewExperiment.thinkOfAnExperiment')}
          </div>

          <Input
            id='experimentName'
            label={i18n.t('NewExperiment.experimentName')}
            inputRef={this._experimentNameRef}
            required={true}
            onChange={this.handleChange('experimentName')}
            value={experimentName}
            autoFocus={true}
            variant='outlined'
          />
          <Input
            id='experimentDescription'
            label={i18n.t('NewExperiment.description')}
            multiline={true}
            onChange={this.handleChange('description')}
            value={description}
            variant='outlined'
          />

          <div className={commonCss.flex}>
            <BusyButton
              id='createExperimentBtn'
              disabled={!!validationError}
              busy={isbeingCreated}
              className={commonCss.buttonAction}
              title={i18n.t('NewExperiment.next')}
              onClick={this._create.bind(this)}
            />
            <Button
              id='cancelNewExperimentBtn'
              onClick={() => this.props.history.push(RoutePage.EXPERIMENTS)}
            >
              {i18n.t('NewExperiment.cancel')}
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
    const newExperiment: ApiExperiment = {
      description: this.state.description,
      name: this.state.experimentName,
      resource_references: this.props.namespace
        ? [
            {
              key: {
                id: this.props.namespace,
                type: ApiResourceType.NAMESPACE,
              },
              relationship: ApiRelationship.OWNER,
            },
          ]
        : undefined,
    };

    this.setState({ isbeingCreated: true }, async () => {
      try {
        const response = await Apis.experimentServiceApi.createExperiment(newExperiment);
        let searchString = '';
        if (this.state.pipelineId) {
          searchString = new URLParser(this.props).build({
            [QUERY_PARAMS.experimentId]: response.id || '',
            [QUERY_PARAMS.pipelineId]: this.state.pipelineId,
            [QUERY_PARAMS.firstRunInExperiment]: '1',
          });
        } else {
          searchString = new URLParser(this.props).build({
            [QUERY_PARAMS.experimentId]: response.id || '',
            [QUERY_PARAMS.firstRunInExperiment]: '1',
          });
        }
        this.props.history.push(RoutePage.NEW_RUN + searchString);
        this.props.updateSnackbar({
          autoHideDuration: 10000,
          message: `${i18n.t('NewExperiment.successfullyCreatedNewExperiment')} ${newExperiment.name}`,
          open: true,
        });
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        await this.showErrorDialog(i18n.t('NewExperiment.experimentCreationFailed'), errorMessage);
        logger.error(i18n.t('NewExperiment.errorCreatingExperiment'), err);
        this.setState({ isbeingCreated: false });
      }
    });
  }

  private _validate(): void {
    // Validate state
    const { experimentName } = this.state;
    try {
      if (!experimentName) {
        throw new Error(i18n.t('NewExperiment.experimentNameIsRequired'));
      }
      this.setState({ validationError: '' });
    } catch (err) {
      this.setState({ validationError: err.message });
    }
  }
}

const EnhancedNewExperiment: React.FC<PageProps> = props => {
  const namespace = React.useContext(NamespaceContext);
  return <NewExperiment {...props} namespace={namespace} />;
};

export default EnhancedNewExperiment;
