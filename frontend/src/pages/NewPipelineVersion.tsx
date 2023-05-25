/*
 * Copyright 2019 The Kubeflow Authors
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

import Button from '@material-ui/core/Button';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import InputAdornment from '@material-ui/core/InputAdornment';
import Radio from '@material-ui/core/Radio';
import { TextFieldProps } from '@material-ui/core/TextField';
import * as React from 'react';
import Dropzone from 'react-dropzone';
import { DocumentationCompilePipeline } from 'src/components/UploadPipelineDialog';
import { classes, stylesheet } from 'typestyle';
import BusyButton from 'src/atoms/BusyButton';
import Input from 'src/atoms/Input';
import { CustomRendererProps } from 'src/components/CustomTable';
import { Description } from 'src/components/Description';
import { QUERY_PARAMS, RoutePage, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { color, commonCss, padding, zIndex } from 'src/Css';
import { Apis, PipelineSortKeys, BuildInfo } from 'src/lib/Apis';
import { URLParser } from 'src/lib/URLParser';
import { errorToMessage, logger } from 'src/lib/Utils';
import { Page, PageProps } from './Page';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import PrivateSharedSelector from 'src/components/PrivateSharedSelector';
import { BuildInfoContext } from 'src/lib/BuildInfo';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import PipelinesDialogV2 from 'src/components/PipelinesDialogV2';

interface NewPipelineVersionState {
  validationError: string;
  isbeingCreated: boolean;
  errorMessage: string;

  pipelineDescription: string;
  pipelineId?: string;
  pipelineName?: string;
  pipelineVersionName: string;
  pipelineVersionDescription: string;
  pipeline?: V2beta1Pipeline;

  codeSourceUrl: string;

  // Package can be local file or url
  importMethod: ImportMethod;
  fileName: string;
  file: File | null;
  packageUrl: string;
  dropzoneActive: boolean;

  // Create a new pipeline or not
  newPipeline: boolean;

  // Select existing pipeline
  pipelineSelectorOpen: boolean;
  unconfirmedSelectedPipeline?: V2beta1Pipeline;

  isPrivate: boolean;
}

interface NewPipelineVersionProps extends PageProps {
  buildInfo?: BuildInfo;
  namespace?: string;
}

export enum ImportMethod {
  LOCAL = 'local',
  URL = 'url',
}

const css = stylesheet({
  dropOverlay: {
    backgroundColor: color.lightGrey,
    border: '2px dashed #aaa',
    bottom: 0,
    left: 0,
    padding: '2.5em 0',
    position: 'absolute',
    right: 0,
    textAlign: 'center',
    top: 0,
    zIndex: zIndex.DROP_ZONE_OVERLAY,
  },
  errorMessage: {
    color: 'red',
  },
  nonEditableInput: {
    color: color.secondaryText,
  },
  selectorDialog: {
    // If screen is small, use calc(100% - 120px). If screen is big, use 1200px.
    maxWidth: 1200, // override default maxWidth to expand this dialog further
    minWidth: 680,
    width: 'calc(100% - 120px)',
  },
});

const descriptionCustomRenderer: React.FC<CustomRendererProps<string>> = props => {
  return <Description description={props.value || ''} forceInline={true} />;
};

export class NewPipelineVersion extends Page<NewPipelineVersionProps, NewPipelineVersionState> {
  private _dropzoneRef = React.createRef<Dropzone & HTMLDivElement>();
  private _pipelineVersionNameRef = React.createRef<HTMLInputElement>();
  private _pipelineNameRef = React.createRef<HTMLInputElement>();
  private _pipelineDescriptionRef = React.createRef<HTMLInputElement>();

  private pipelineSelectorColumns = [
    { label: 'Pipeline name', flex: 1, sortKey: PipelineSortKeys.NAME },
    { label: 'Description', flex: 2, customRenderer: descriptionCustomRenderer },
    { label: 'Uploaded on', flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
  ];

  constructor(props: NewPipelineVersionProps) {
    super(props);

    const urlParser = new URLParser(props);
    const pipelineId = urlParser.get(QUERY_PARAMS.pipelineId);

    let isPrivate = false;
    if (props.buildInfo?.apiServerMultiUser) {
      isPrivate = true;
    }

    this.state = {
      codeSourceUrl: '',
      dropzoneActive: false,
      errorMessage: '',
      file: null,
      fileName: '',
      importMethod: ImportMethod.URL,
      isbeingCreated: false,
      newPipeline: pipelineId ? false : true,
      packageUrl: '',
      pipelineDescription: '',
      pipelineId: '',
      pipelineName: '',
      pipelineSelectorOpen: false,
      pipelineVersionName: '',
      pipelineVersionDescription: '',
      validationError: '',
      isPrivate,
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Pipeline Versions', href: RoutePage.NEW_PIPELINE_VERSION }],
      pageTitle: 'New Pipeline',
    };
  }

  public render(): JSX.Element {
    const {
      packageUrl,
      pipelineName,
      pipelineVersionName,
      pipelineVersionDescription,
      isbeingCreated,
      validationError,
      pipelineSelectorOpen,
      codeSourceUrl,
      importMethod,
      newPipeline,
      pipelineDescription,
      fileName,
      dropzoneActive,
    } = this.state;

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <div className={classes(commonCss.scrollContainer, padding(20, 'lr'))}>
          {/* Two subpages: one for creating version under existing pipeline and one for creating version under new pipeline */}
          <div className={classes(padding(10, 't'))}>Upload pipeline or pipeline version.</div>
          <div className={classes(commonCss.flex, padding(10, 'b'))}>
            <FormControlLabel
              id='createNewPipelineBtn'
              label='Create a new pipeline'
              checked={newPipeline === true}
              control={<Radio color='primary' />}
              onChange={() =>
                this.setState({
                  codeSourceUrl: '',
                  newPipeline: true,
                  pipelineDescription: '',
                  pipelineName: '',
                  pipelineVersionName: '',
                })
              }
            />
            <FormControlLabel
              id='createPipelineVersionUnderExistingPipelineBtn'
              label='Create a new pipeline version under an existing pipeline'
              checked={newPipeline === false}
              control={<Radio color='primary' />}
              onChange={() =>
                this.setState({
                  codeSourceUrl: '',
                  newPipeline: false,
                  pipelineDescription: '',
                  pipelineVersionDescription: '',
                  pipelineName: '',
                  pipelineVersionName: '',
                })
              }
            />
          </div>

          {newPipeline === true && this.props.buildInfo?.apiServerMultiUser && (
            <PrivateSharedSelector
              onChange={val => {
                this.setState({
                  isPrivate: val,
                });
              }}
            ></PrivateSharedSelector>
          )}

          {/* Pipeline name and help text for uploading new pipeline */}
          {newPipeline === true && (
            <>
              <div>Upload pipeline with the specified package.</div>
              <Input
                id='newPipelineName'
                value={pipelineName}
                required={true}
                label='Pipeline Name'
                variant='outlined'
                inputRef={this._pipelineNameRef}
                onChange={this.handleChange('pipelineName')}
                autoFocus={true}
              />
              <Input
                id='pipelineDescription'
                value={pipelineDescription}
                required={false}
                label='Pipeline Description'
                variant='outlined'
                inputRef={this._pipelineDescriptionRef}
                onChange={this.handleChange('pipelineDescription')}
                autoFocus={true}
              />
              {/* Choose a local file for package or specify a url for package */}
            </>
          )}

          {/* Pipeline selector and help text for uploading new pipeline version */}
          {newPipeline === false && (
            <>
              <div>Upload pipeline version with the specified package.</div>
              {/* Select pipeline */}
              <Input
                value={pipelineName}
                required={true}
                label='Pipeline'
                disabled={true}
                variant='outlined'
                inputRef={this._pipelineNameRef}
                onChange={this.handleChange('pipelineName')}
                autoFocus={true}
                InputProps={{
                  classes: { disabled: css.nonEditableInput },
                  endAdornment: (
                    <InputAdornment position='end'>
                      <Button
                        color='secondary'
                        id='choosePipelineBtn'
                        onClick={() => this.setStateSafe({ pipelineSelectorOpen: true })}
                        style={{ padding: '3px 5px', margin: 0 }}
                      >
                        Choose
                      </Button>
                    </InputAdornment>
                  ),
                  readOnly: true,
                }}
              />

              <PipelinesDialogV2
                {...this.props}
                open={pipelineSelectorOpen}
                selectorDialog={css.selectorDialog}
                onClose={(confirmed, selectedPipeline?: V2beta1Pipeline) => {
                  this.setStateSafe({ unconfirmedSelectedPipeline: selectedPipeline }, () => {
                    this._pipelineSelectorClosed(confirmed);
                  });
                }}
                namespace={this.props.namespace}
                pipelineSelectorColumns={this.pipelineSelectorColumns}
              ></PipelinesDialogV2>

              {/* Set pipeline version name */}
              <Input
                id='pipelineVersionName'
                label='Pipeline Version name'
                inputRef={this._pipelineVersionNameRef}
                required={true}
                onChange={this.handleChange('pipelineVersionName')}
                value={pipelineVersionName}
                autoFocus={true}
                variant='outlined'
              />
              <Input
                id='pipelineVersionDescription'
                value={pipelineVersionDescription}
                required={false}
                label='Pipeline Version Description'
                variant='outlined'
                onChange={this.handleChange('pipelineVersionDescription')}
                autoFocus={true}
              />
            </>
          )}

          {/* Different package explanation based on import method*/}
          {this.state.importMethod === ImportMethod.LOCAL && (
            <>
              <div className={padding(10, 'b')}>
                Choose a pipeline package file from your computer, and give the pipeline a unique
                name.
                <br />
                You can also drag and drop the file here.
              </div>
              <DocumentationCompilePipeline />
            </>
          )}
          {this.state.importMethod === ImportMethod.URL && (
            <>
              <div className={padding(10, 'b')}>URL must be publicly accessible.</div>
              <DocumentationCompilePipeline />
            </>
          )}

          {/* Different package input field based on import method*/}
          <div className={classes(commonCss.flex, padding(10, 'b'))}>
            <FormControlLabel
              id='localPackageBtn'
              label='Upload a file'
              checked={importMethod === ImportMethod.LOCAL}
              control={<Radio color='primary' />}
              onChange={() => this.setState({ importMethod: ImportMethod.LOCAL })}
            />
            <Dropzone
              id='dropZone'
              disableClick={true}
              onDrop={this._onDrop.bind(this)}
              onDragEnter={this._onDropzoneDragEnter.bind(this)}
              onDragLeave={this._onDropzoneDragLeave.bind(this)}
              style={{ position: 'relative' }}
              ref={this._dropzoneRef}
              inputProps={{ tabIndex: -1 }}
              disabled={importMethod === ImportMethod.URL}
            >
              {dropzoneActive && <div className={css.dropOverlay}>Drop files..</div>}
              <Input
                data-testid='uploadFileInput'
                onChange={this.handleChange('fileName')}
                value={fileName}
                required={true}
                label='File'
                variant='outlined'
                disabled={importMethod === ImportMethod.URL}
                // Find a better to align this input box with others
                InputProps={{
                  endAdornment: (
                    <InputAdornment position='end'>
                      <Button
                        color='secondary'
                        onClick={() => this._dropzoneRef.current!.open()}
                        style={{ padding: '3px 5px', margin: 0, whiteSpace: 'nowrap' }}
                        disabled={importMethod === ImportMethod.URL}
                      >
                        Choose file
                      </Button>
                    </InputAdornment>
                  ),
                  readOnly: true,
                  style: {
                    maxWidth: 2000,
                    width: 455,
                  },
                }}
              />
            </Dropzone>
          </div>
          <div className={classes(commonCss.flex, padding(10, 'b'))}>
            <FormControlLabel
              id='remotePackageBtn'
              label='Import by url'
              checked={importMethod === ImportMethod.URL}
              control={<Radio color='primary' />}
              onChange={() => this.setState({ importMethod: ImportMethod.URL })}
            />
            <Input
              id='pipelinePackageUrl'
              label='Package Url'
              multiline={true}
              onChange={this.handleChange('packageUrl')}
              value={packageUrl}
              variant='outlined'
              disabled={importMethod === ImportMethod.LOCAL}
              // Find a better to align this input box with others
              style={{
                maxWidth: 2000,
                width: 465,
              }}
            />
          </div>

          {/* Fill pipeline version code source url */}
          <Input
            id='pipelineVersionCodeSource'
            label='Code Source'
            multiline={true}
            onChange={this.handleChange('codeSourceUrl')}
            required={false}
            value={codeSourceUrl}
            variant='outlined'
          />

          {/* Create pipeline or pipeline version */}
          <div className={commonCss.flex}>
            <BusyButton
              id='createNewPipelineOrVersionBtn'
              disabled={!!validationError}
              busy={isbeingCreated}
              className={commonCss.buttonAction}
              title={'Create'}
              onClick={this._create.bind(this)}
            />
            <Button
              id='cancelNewPipelineOrVersionBtn'
              onClick={() => this.props.history.push(RoutePage.PIPELINES)}
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
      const pipelineResponse = await Apis.pipelineServiceApiV2.getPipeline(pipelineId);
      this.setState({
        pipelineId,
        pipelineName: pipelineResponse.display_name,
        pipeline: pipelineResponse,
      });
      // Suggest a version name based on pipeline name
      const currDate = new Date();
      this.setState({
        pipelineVersionName:
          pipelineResponse.display_name + '_version_at_' + currDate.toISOString(),
      });
    }

    this._validate();
  }

  public handleChange = (name: string) => (event: any) => {
    const value = (event.target as TextFieldProps).value;
    this.setState({ [name]: value } as any, this._validate.bind(this));

    // When pipeline name is changed, we have some special logic
    if (name === 'pipelineName') {
      // Suggest a version name based on pipeline name
      const currDate = new Date();
      this.setState(
        { pipelineVersionName: value + '_version_at_' + currDate.toISOString() },
        this._validate.bind(this),
      );
    }
  };

  protected async _pipelineSelectorClosed(confirmed: boolean): Promise<void> {
    let { pipeline } = this.state;
    const currDate = new Date();
    if (confirmed && this.state.unconfirmedSelectedPipeline) {
      pipeline = this.state.unconfirmedSelectedPipeline;
    }

    this.setStateSafe(
      {
        pipeline,
        pipelineId: (pipeline && pipeline.pipeline_id) || '',
        pipelineName: (pipeline && pipeline.display_name) || '',
        pipelineSelectorOpen: false,
        // Suggest a version name based on pipeline name
        pipelineVersionName:
          (pipeline && pipeline.display_name + '_version_at_' + currDate.toISOString()) || '',
      },
      () => this._validate(),
    );
  }

  // To call _onDrop from test, so make a protected method
  protected _onDropForTest(files: File[]): void {
    this._onDrop(files);
  }

  private async _create(): Promise<void> {
    this.setState({ isbeingCreated: true }, async () => {
      try {
        let namespace: undefined | string;
        if (this.props.buildInfo?.apiServerMultiUser) {
          if (this.state.isPrivate) {
            namespace = this.props.namespace;
          }
        }
        // 3 use case for now:
        // (1) new pipeline (and a default version) from local file
        // (2) new pipeline (and a default version) from url
        // (3) new pipeline version (under an existing pipeline) from url
        let pipelineVersionResponse: V2beta1PipelineVersion;
        if (this.state.newPipeline && this.state.importMethod === ImportMethod.LOCAL) {
          const pipelineResponse = await Apis.uploadPipelineV2(
            this.state.pipelineName!,
            this.state.pipelineDescription,
            this.state.file!,
            namespace,
          );
          const listVersionsResponse = await Apis.pipelineServiceApiV2.listPipelineVersions(
            pipelineResponse.pipeline_id!,
            undefined,
            1, // Only need the latest one
            'created_at desc',
          );
          if (listVersionsResponse.pipeline_versions) {
            pipelineVersionResponse = listVersionsResponse.pipeline_versions[0];
          } else {
            throw new Error('Pipeline is empty');
          }
        } else if (this.state.newPipeline && this.state.importMethod === ImportMethod.URL) {
          const newPipeline: V2beta1Pipeline = {
            description: this.state.pipelineDescription,
            display_name: this.state.pipelineName,
            namespace,
          };
          const createPipelineResponse = await Apis.pipelineServiceApiV2.createPipeline(
            newPipeline,
          );
          this.setState({ pipelineId: createPipelineResponse.pipeline_id });
          pipelineVersionResponse = await this._createPipelineVersion();
        } else {
          pipelineVersionResponse = await this._createPipelineVersion();
        }

        // If success, go to pipeline details page of the new version
        this.props.history.push(
          RoutePage.PIPELINE_DETAILS.replace(
            `:${RouteParams.pipelineId}`,
            pipelineVersionResponse.pipeline_id! /* pipeline id of this version */,
          ).replace(
            `:${RouteParams.pipelineVersionId}`,
            pipelineVersionResponse.pipeline_version_id!,
          ),
        );
        this.props.updateSnackbar({
          autoHideDuration: 10000,
          message: `Successfully created new pipeline version: ${pipelineVersionResponse.display_name}`,
          open: true,
        });
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        await this.showErrorDialog('Pipeline version creation failed', errorMessage);
        logger.error('Error creating pipeline version:', err);
        this.setState({ isbeingCreated: false });
      }
    });
  }

  private async _createPipelineVersion(): Promise<V2beta1PipelineVersion> {
    if (this.state.importMethod === ImportMethod.LOCAL) {
      if (!this.state.file) {
        throw new Error('File should be selected');
      }
      return Apis.uploadPipelineVersionV2(
        this.state.pipelineVersionName,
        this.state.pipelineId!,
        this.state.file,
        this.state.pipelineVersionDescription,
      );
    } else {
      // this.state.importMethod === ImportMethod.URL
      let newPipeline: V2beta1PipelineVersion = {
        pipeline_id: this.state.pipelineId,
        display_name: this.state.pipelineVersionName,
        description: this.state.pipelineVersionDescription,
        package_url: { pipeline_url: this.state.packageUrl },
      };
      return Apis.pipelineServiceApiV2.createPipelineVersion(this.state.pipelineId!, newPipeline);
    }
  }

  private _validate(): void {
    // Validate state
    // 3 valid use case for now:
    // (1) new pipeline (and a default version) from local file
    // (2) new pipeline (and a default version) from url
    // (3) new pipeline version (under an existing pipeline) from url
    const {
      fileName,
      pipeline,
      pipelineVersionName,
      packageUrl,
      newPipeline,
      pipelineName,
    } = this.state;
    try {
      if (newPipeline) {
        if (!packageUrl && !fileName) {
          throw new Error('Must specify either package url  or file in .yaml, .zip, or .tar.gz');
        }
        if (!pipelineName) {
          throw new Error('Pipeline name is required');
        }
      } else {
        if (!pipeline) {
          throw new Error('Pipeline is required');
        }
        if (!pipelineVersionName) {
          throw new Error('Pipeline version name is required');
        }
        if (pipelineVersionName && pipelineVersionName.length > 100) {
          throw new Error('Pipeline version name must contain no more than 100 characters');
        }
        if (!packageUrl && !fileName) {
          throw new Error('Please specify either package url or file in .yaml, .zip, or .tar.gz');
        }
      }
      this.setState({ validationError: '' });
    } catch (err) {
      this.setState({ validationError: err.message });
    }
  }

  private _onDropzoneDragEnter(): void {
    this.setState({ dropzoneActive: true });
  }

  private _onDropzoneDragLeave(): void {
    this.setState({ dropzoneActive: false });
  }

  private _onDrop(files: File[]): void {
    this.setStateSafe(
      {
        dropzoneActive: false,
        file: files[0],
        fileName: files[0].name,
        pipelineName: this.state.pipelineName || files[0].name.split('.')[0],
      },
      () => {
        this._validate();
      },
    );
  }
}

const EnhancedNewPipelineVersion: React.FC<PageProps> = props => {
  const buildInfo = React.useContext(BuildInfoContext);
  const namespace = React.useContext(NamespaceContext);

  return <NewPipelineVersion {...props} buildInfo={buildInfo} namespace={namespace} />;
};

export default EnhancedNewPipelineVersion;
