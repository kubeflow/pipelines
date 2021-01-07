/*
 * Copyright 2019 Google LLC
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
import Buttons from '../lib/Buttons';
import Dropzone from 'react-dropzone';
import Input from '../atoms/Input';
import { Page } from './Page';
import { RoutePage, QUERY_PARAMS, RouteParams } from '../components/Router';
import { TextFieldProps } from '@material-ui/core/TextField';
import { ToolbarProps } from '../components/Toolbar';
import { URLParser } from '../lib/URLParser';
import { classes, stylesheet } from 'typestyle';
import { commonCss, padding, color, fontsize, zIndex } from '../Css';
import { logger, errorToMessage } from '../lib/Utils';
import ResourceSelector from './ResourceSelector';
import InputAdornment from '@material-ui/core/InputAdornment';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import { ApiResourceType } from '../apis/run';
import { Apis, PipelineSortKeys } from '../lib/Apis';
import { ApiPipeline, ApiPipelineVersion } from '../apis/pipeline';
import { CustomRendererProps } from '../components/CustomTable';
import { Description } from '../components/Description';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Radio from '@material-ui/core/Radio';
import { ExternalLink } from '../atoms/ExternalLink';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

interface NewPipelineVersionState {
  validationError: string;
  isbeingCreated: boolean;
  errorMessage: string;

  pipelineDescription: string;
  pipelineId?: string;
  pipelineName?: string;
  pipelineVersionName: string;
  pipeline?: ApiPipeline;

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
  unconfirmedSelectedPipeline?: ApiPipeline;
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
  explanation: {
    fontSize: fontsize.small,
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

class NewPipelineVersion extends Page<{ t: TFunction }, NewPipelineVersionState> {
  private _dropzoneRef = React.createRef<Dropzone & HTMLDivElement>();
  private _pipelineVersionNameRef = React.createRef<HTMLInputElement>();
  private _pipelineNameRef = React.createRef<HTMLInputElement>();
  private _pipelineDescriptionRef = React.createRef<HTMLInputElement>();

  private pipelineSelectorColumns = [
    { label: this.props.t('common:pipelineName'), flex: 1, sortKey: PipelineSortKeys.NAME },
    {
      label: this.props.t('common:description'),
      flex: 2,
      customRenderer: descriptionCustomRenderer,
    },
    { label: this.props.t('common:uploadedOn'), flex: 1, sortKey: PipelineSortKeys.CREATED_AT },
  ];

  constructor(props: any) {
    super(props);

    const urlParser = new URLParser(props);
    const pipelineId = urlParser.get(QUERY_PARAMS.pipelineId);

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
      validationError: '',
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const { t } = this.props;
    return {
      actions: {},
      breadcrumbs: [
        { displayName: t('pipelines:pipelineVersions'), href: RoutePage.NEW_PIPELINE_VERSION },
      ],
      pageTitle: t('pipelines:uploadPipelineTitle'),
      t,
    };
  }

  public render(): JSX.Element {
    const { t } = this.props;
    const {
      packageUrl,
      pipelineName,
      pipelineVersionName,
      isbeingCreated,
      validationError,
      pipelineSelectorOpen,
      unconfirmedSelectedPipeline,
      codeSourceUrl,
      importMethod,
      newPipeline,
      pipelineDescription,
      fileName,
      dropzoneActive,
    } = this.state;

    const buttons = new Buttons(this.props, this.refresh.bind(this));

    const compilePipelineDocs = (
      <div className={padding(10, 'b')}>
        {t('expectedFileFormat')}{' '}
        <ExternalLink href='https://www.kubeflow.org/docs/pipelines/sdk/build-component/#compile-the-pipeline'>
          {t('compilePipelineDoc')}
        </ExternalLink>
        .
      </div>
    );

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <div className={classes(commonCss.scrollContainer, padding(20, 'lr'))}>
          {/* Two subpages: one for creating version under existing pipeline and one for creating version under new pipeline */}
          <div className={classes(commonCss.flex, padding(10, 'b'))}>
            <FormControlLabel
              id='createNewPipelineBtn'
              label={t('createPipeline')}
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
              label={t('createPipelineVersion')}
              checked={newPipeline === false}
              control={<Radio color='primary' />}
              onChange={() =>
                this.setState({
                  codeSourceUrl: '',
                  newPipeline: false,
                  pipelineDescription: '',
                  pipelineName: '',
                  pipelineVersionName: '',
                })
              }
            />
          </div>

          {/* Pipeline name and help text for uploading new pipeline */}
          {newPipeline === true && (
            <>
              <div className={css.explanation}>{t('common:uploadPipeline')}</div>
              <Input
                id='newPipelineName'
                value={pipelineName}
                required={true}
                label={t('pipelineNameCaps')}
                variant='outlined'
                inputRef={this._pipelineNameRef}
                onChange={this.handleChange('pipelineName')}
                autoFocus={true}
              />

              <Input
                id='pipelineDescription'
                value={pipelineDescription}
                required={true}
                label={t('pipelineDescription')}
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
              <div className={css.explanation}>{t('uploadPipelineVersion')}</div>
              {/* Select pipeline */}
              <Input
                value={pipelineName}
                required={true}
                label={t('common:pipeline')}
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
                        {t('common:choose')}
                      </Button>
                    </InputAdornment>
                  ),
                  readOnly: true,
                }}
              />
              <Dialog
                open={pipelineSelectorOpen}
                classes={{ paper: css.selectorDialog }}
                onClose={() => this._pipelineSelectorClosed(false)}
                PaperProps={{ id: 'pipelineSelectorDialog' }}
              >
                <DialogContent>
                  <ResourceSelector
                    {...this.props}
                    title={t('common:choosePipeline')}
                    filterLabel={t('common:filterPipelines')}
                    listApi={async (...args) => {
                      const response = await Apis.pipelineServiceApi.listPipelines(...args);
                      return {
                        nextPageToken: response.next_page_token || '',
                        resources: response.pipelines || [],
                      };
                    }}
                    columns={this.pipelineSelectorColumns}
                    emptyMessage={t('common:noPipelinesFoundTryAgain')}
                    initialSortColumn={PipelineSortKeys.CREATED_AT}
                    selectionChanged={(selectedPipeline: ApiPipeline) =>
                      this.setStateSafe({ unconfirmedSelectedPipeline: selectedPipeline })
                    }
                    toolbarActionMap={buttons
                      .upload(() => this.setStateSafe({ pipelineSelectorOpen: false }))
                      .getToolbarActionMap()}
                  />
                </DialogContent>
                <DialogActions>
                  <Button
                    id='cancelPipelineSelectionBtn'
                    onClick={() => this._pipelineSelectorClosed(false)}
                    color='secondary'
                  >
                    {t('common:cancel')}
                  </Button>
                  <Button
                    id='usePipelineBtn'
                    onClick={() => this._pipelineSelectorClosed(true)}
                    color='secondary'
                    disabled={!unconfirmedSelectedPipeline}
                  >
                    {t('common:usePipeline')}
                  </Button>
                </DialogActions>
              </Dialog>

              {/* Set pipeline version name */}
              <Input
                id='pipelineVersionName'
                label={t('pipelineVersionName')}
                inputRef={this._pipelineVersionNameRef}
                required={true}
                onChange={this.handleChange('pipelineVersionName')}
                value={pipelineVersionName}
                autoFocus={true}
                variant='outlined'
              />
            </>
          )}

          {/* Different package explanation based on import method*/}
          {this.state.importMethod === ImportMethod.LOCAL && (
            <>
              <div className={padding(10, 'b')}>
                {t('uploadPackagePipelineFileInstructions')}
                <br />
                {t('dragAndDropFile')}
              </div>
              {compilePipelineDocs}
            </>
          )}
          {this.state.importMethod === ImportMethod.URL && (
            <>
              <div className={padding(10, 'b')}>{t('urlPublic')}</div>
              {compilePipelineDocs}
            </>
          )}

          {/* Different package input field based on import method*/}
          <div className={classes(commonCss.flex, padding(10, 'b'))}>
            <FormControlLabel
              id='localPackageBtn'
              label={t('common:uploadFile')}
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
              {dropzoneActive && <div className={css.dropOverlay}>{t('common:dropFiles')}</div>}
              <Input
                onChange={this.handleChange('fileName')}
                value={fileName}
                required={true}
                label={t('common:file')}
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
                        {t('common:chooseFile')}
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
              label={t('importByUrl')}
              checked={importMethod === ImportMethod.URL}
              control={<Radio color='primary' />}
              onChange={() => this.setState({ importMethod: ImportMethod.URL })}
            />
            <Input
              id='pipelinePackageUrl'
              label={t('packageUrl')}
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
            label={t('codeSource')}
            multiline={true}
            onChange={this.handleChange('codeSourceUrl')}
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
              title={t('common:create')}
              onClick={this._create.bind(this)}
            />
            <Button
              id='cancelNewPipelineOrVersionBtn'
              onClick={() => this.props.history.push(RoutePage.PIPELINES)}
            >
              {t('common:cancel')}
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
      const apiPipeline = await Apis.pipelineServiceApi.getPipeline(pipelineId);
      this.setState({ pipelineId, pipelineName: apiPipeline.name, pipeline: apiPipeline });
      // Suggest a version name based on pipeline name
      const currDate = new Date();
      this.setState({
        pipelineVersionName: apiPipeline.name + '_version_at_' + currDate.toISOString(),
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
        pipelineId: (pipeline && pipeline.id) || '',
        pipelineName: (pipeline && pipeline.name) || '',
        pipelineSelectorOpen: false,
        // Suggest a version name based on pipeline name
        pipelineVersionName:
          (pipeline && pipeline.name + '_version_at_' + currDate.toISOString()) || '',
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
      const { t } = this.props;
      try {
        // 3 use case for now:
        // (1) new pipeline (and a default version) from local file
        // (2) new pipeline (and a default version) from url
        // (3) new pipeline version (under an existing pipeline) from url
        const response =
          this.state.newPipeline && this.state.importMethod === ImportMethod.LOCAL
            ? (
                await Apis.uploadPipeline(
                  this.state.pipelineName!,
                  this.state.pipelineDescription,
                  this.state.file!,
                )
              ).default_version!
            : this.state.newPipeline && this.state.importMethod === ImportMethod.URL
            ? (
                await Apis.pipelineServiceApi.createPipeline({
                  description: this.state.pipelineDescription,
                  name: this.state.pipelineName!,
                  url: { pipeline_url: this.state.packageUrl },
                })
              ).default_version!
            : await this._createPipelineVersion();

        // If success, go to pipeline details page of the new version
        this.props.history.push(
          RoutePage.PIPELINE_DETAILS.replace(
            `:${RouteParams.pipelineId}`,
            response.resource_references![0].key!.id! /* pipeline id of this version */,
          ).replace(`:${RouteParams.pipelineVersionId}`, response.id!),
        );
        this.props.updateSnackbar({
          autoHideDuration: 10000,
          message: `${t('newPipelineVersionSuccess')}: ${response.name}`,
          open: true,
        });
      } catch (err) {
        const errorMessage = await errorToMessage(err);
        await this.showErrorDialog(t('newPipelineVersionFailed'), errorMessage);
        logger.error(t('newPipelineVersionError'), err);
        this.setState({ isbeingCreated: false });
      }
    });
  }

  private async _createPipelineVersion(): Promise<ApiPipelineVersion> {
    const { t } = this.props;
    const getPipelineId = async () => {
      if (this.state.pipelineId) {
        // Get existing pipeline's id.
        return this.state.pipelineId;
      } else {
        // Get the new pipeline's id.
        // The new pipeline version is going to be put under this new pipeline
        // instead of an eixsting pipeline. So create this new pipeline first.
        const newPipeline: ApiPipeline = {
          description: this.state.pipelineDescription,
          name: this.state.pipelineName,
          url: { pipeline_url: this.state.packageUrl },
        };
        const response = await Apis.pipelineServiceApi.createPipeline(newPipeline);
        return response.id!;
      }
    };

    if (this.state.importMethod === ImportMethod.LOCAL) {
      if (!this.state.file) {
        throw new Error(t('fileSelectedError'));
      }
      return Apis.uploadPipelineVersion(
        this.state.pipelineVersionName,
        await getPipelineId(),
        this.state.file,
      );
    } else {
      // this.state.importMethod === ImportMethod.URL
      return Apis.pipelineServiceApi.createPipelineVersion({
        code_source_url: this.state.codeSourceUrl,
        name: this.state.pipelineVersionName,
        package_url: { pipeline_url: this.state.packageUrl },
        resource_references: [
          { key: { id: await getPipelineId(), type: ApiResourceType.PIPELINE }, relationship: 1 },
        ],
      });
    }
  }

  private _validate(): void {
    // Validate state
    // 3 valid use case for now:
    // (1) new pipeline (and a default version) from local file
    // (2) new pipeline (and a default version) from url
    // (3) new pipeline version (under an existing pipeline) from url
    const { fileName, pipeline, pipelineVersionName, packageUrl, newPipeline } = this.state;
    const { t } = this.props;
    try {
      if (newPipeline) {
        if (!packageUrl && !fileName) {
          throw new Error(t('mustSpecifyUrlFileError'));
        }
      } else {
        if (!pipeline) {
          throw new Error(t('pipelineRequired'));
        }
        if (!pipelineVersionName) {
          throw new Error(t('versionNameRequired'));
        }
        if (!packageUrl && !fileName) {
          throw new Error(t('pleaeSpecifyUrlFileError'));
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

export default withTranslation(['pipelines', 'common'])(NewPipelineVersion);
