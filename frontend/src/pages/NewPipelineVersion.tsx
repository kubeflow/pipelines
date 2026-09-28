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

import * as React from 'react';
import { Button } from 'src/components/ui/button';
import { TextField } from 'src/components/ui/text-field';
import 'src/components/modernization/PipelineForms.css';
import DropzoneArea, { DropzoneAreaHandle } from 'src/atoms/DropzoneArea';
import {
  DocumentationCompilePipeline,
  PIPELINE_PACKAGE_ACCEPT,
  PIPELINE_PACKAGE_REJECT_MESSAGE,
  pipelinePackageValidator,
} from 'src/components/UploadPipelineDialog';
import { CustomRendererProps } from 'src/components/CustomTable';
import { Description } from 'src/components/Description';
import { QUERY_PARAMS, RoutePage, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/lib/PageChromeTypes';
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
  pipelineDisplayName?: string;
  pipelineVersionName: string;
  pipelineVersionDisplayName: string;
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

const getK8sNameRegex = () => /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/;

const descriptionCustomRenderer: React.FC<CustomRendererProps<string>> = (props) => {
  return <Description description={props.value || ''} forceInline={true} />;
};

export class NewPipelineVersion extends Page<NewPipelineVersionProps, NewPipelineVersionState> {
  private _dropzoneRef = React.createRef<DropzoneAreaHandle>();
  private _pipelineVersionNameRef = React.createRef<HTMLInputElement>();
  private _pipelineVersionDisplayNameRef = React.createRef<HTMLInputElement>();
  private _pipelineNameRef = React.createRef<HTMLInputElement>();
  private _pipelineDisplayNameRef = React.createRef<HTMLInputElement>();
  private _pipelineDescriptionRef = React.createRef<HTMLInputElement>();

  private pipelineSelectorColumns = [
    {
      label: 'Pipeline name',
      flex: 1,
      sortKey: PipelineSortKeys.DISPLAY_NAME,
      customRenderer: ({
        value,
      }: CustomRendererProps<{ display_name?: string; name?: string }>) => (
        <span title={value?.name}>{value?.display_name || value?.name}</span>
      ),
    },
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
      pipelineDisplayName: '',
      pipelineSelectorOpen: false,
      pipelineVersionName: '',
      pipelineVersionDisplayName: '',
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

  public render(): React.JSX.Element {
    const {
      packageUrl,
      pipelineName,
      pipelineDisplayName,
      pipelineVersionName,
      pipelineVersionDisplayName,
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
    const kubernetesStore = this.props.buildInfo?.pipelineStore === 'kubernetes';
    return (
      <div className='kfp-pipeline-form-page'>
        <form
          className='kfp-pipeline-form'
          onSubmit={(event) => {
            event.preventDefault();
            if (!validationError && !isbeingCreated) void this._create();
          }}
        >
          <section className='kfp-pipeline-form-card'>
            <h2>Pipeline</h2>
            <fieldset className='kfp-pipeline-choice'>
              <legend>Upload pipeline or pipeline version.</legend>
              <label>
                <input
                  id='createNewPipelineBtn'
                  name='pipeline-kind'
                  type='radio'
                  checked={newPipeline}
                  onChange={() =>
                    this.setState(
                      {
                        codeSourceUrl: '',
                        newPipeline: true,
                        pipelineDescription: '',
                        pipelineName: '',
                        pipelineDisplayName: '',
                        pipelineVersionName: '',
                        pipelineVersionDisplayName: '',
                      },
                      this._validate.bind(this),
                    )
                  }
                />
                Create a new pipeline
              </label>
              <label>
                <input
                  id='createPipelineVersionUnderExistingPipelineBtn'
                  name='pipeline-kind'
                  type='radio'
                  checked={!newPipeline}
                  onChange={() =>
                    this.setState(
                      {
                        codeSourceUrl: '',
                        newPipeline: false,
                        pipelineDescription: '',
                        pipelineVersionDescription: '',
                        pipelineName: '',
                        pipelineDisplayName: '',
                        pipelineVersionName: '',
                        pipelineVersionDisplayName: '',
                      },
                      this._validate.bind(this),
                    )
                  }
                />
                Create a new pipeline version under an existing pipeline
              </label>
            </fieldset>
            {newPipeline && this.props.buildInfo?.apiServerMultiUser && (
              <PrivateSharedSelector
                value={this.state.isPrivate}
                onChange={(isPrivate) => this.setState({ isPrivate })}
              />
            )}
            {newPipeline ? (
              <>
                <TextField
                  id='newPipelineName'
                  value={pipelineName}
                  required
                  label={'Pipeline Name' + (kubernetesStore ? ' (Kubernetes object name)' : '')}
                  ref={this._pipelineNameRef}
                  onChange={this.handleChange('pipelineName')}
                  autoFocus
                />
                {kubernetesStore && (
                  <TextField
                    id='newPipelineDisplayName'
                    value={pipelineDisplayName}
                    label='Pipeline Display Name'
                    ref={this._pipelineDisplayNameRef}
                    onChange={this.handleChange('pipelineDisplayName')}
                  />
                )}
                <TextField
                  id='pipelineDescription'
                  value={pipelineDescription}
                  label='Pipeline Description'
                  ref={this._pipelineDescriptionRef}
                  onChange={this.handleChange('pipelineDescription')}
                />
              </>
            ) : (
              <>
                <TextField
                  value={pipelineDisplayName || pipelineName}
                  required
                  label='Pipeline'
                  readOnly
                  ref={this._pipelineNameRef}
                  trailingContent={
                    <Button
                      variant='secondary'
                      id='choosePipelineBtn'
                      onClick={() => this.setStateSafe({ pipelineSelectorOpen: true })}
                    >
                      Choose
                    </Button>
                  }
                />
                <PipelinesDialogV2
                  {...this.props}
                  open={pipelineSelectorOpen}
                  selectorDialog=''
                  namespace={this.props.namespace}
                  pipelineSelectorColumns={this.pipelineSelectorColumns}
                  onClose={(confirmed, selectedPipeline?: V2beta1Pipeline) =>
                    this.setStateSafe({ unconfirmedSelectedPipeline: selectedPipeline }, () =>
                      this._pipelineSelectorClosed(confirmed),
                    )
                  }
                />
                <TextField
                  id='pipelineVersionName'
                  label={
                    'Pipeline Version Name' + (kubernetesStore ? ' (Kubernetes object name)' : '')
                  }
                  ref={this._pipelineVersionNameRef}
                  required
                  onChange={this.handleChange('pipelineVersionName')}
                  value={pipelineVersionName}
                  autoFocus
                />
                {kubernetesStore && (
                  <TextField
                    id='pipelineVersionDisplayName'
                    label='Pipeline Version Display Name'
                    ref={this._pipelineVersionDisplayNameRef}
                    onChange={this.handleChange('pipelineVersionDisplayName')}
                    value={pipelineVersionDisplayName}
                  />
                )}
                <TextField
                  id='pipelineVersionDescription'
                  value={pipelineVersionDescription}
                  label='Pipeline Version Description'
                  onChange={this.handleChange('pipelineVersionDescription')}
                />
              </>
            )}
          </section>
          <section className='kfp-pipeline-form-card'>
            <h2>Package</h2>
            <fieldset className='kfp-pipeline-choice'>
              <legend>Pipeline package source</legend>
              <label>
                <input
                  id='localPackageBtn'
                  name='package-source'
                  type='radio'
                  checked={importMethod === ImportMethod.LOCAL}
                  onChange={() =>
                    this.setState({ importMethod: ImportMethod.LOCAL }, this._validate.bind(this))
                  }
                />
                Upload a file
              </label>
              <label>
                <input
                  id='remotePackageBtn'
                  name='package-source'
                  type='radio'
                  checked={importMethod === ImportMethod.URL}
                  onChange={() =>
                    this.setState({ importMethod: ImportMethod.URL }, this._validate.bind(this))
                  }
                />
                Import by url
              </label>
            </fieldset>
            <DocumentationCompilePipeline />
            <div className='kfp-pipeline-dropzone'>
              <DropzoneArea
                id='dropZone'
                aria-label='Pipeline package drop zone'
                onDrop={this._onDrop.bind(this)}
                onDropRejected={this._onDropRejected.bind(this)}
                onDragEnter={this._onDropzoneDragEnter.bind(this)}
                onDragLeave={this._onDropzoneDragLeave.bind(this)}
                accept={PIPELINE_PACKAGE_ACCEPT}
                validator={pipelinePackageValidator}
                disabled={importMethod === ImportMethod.URL}
                ref={this._dropzoneRef}
                inputProps={{ tabIndex: -1 }}
              >
                {dropzoneActive && <div className='kfp-pipeline-drop-overlay'>Drop files…</div>}
                <TextField
                  data-testid='uploadFileInput'
                  value={fileName}
                  required={importMethod === ImportMethod.LOCAL}
                  label='File'
                  readOnly
                  disabled={importMethod === ImportMethod.URL}
                  trailingContent={
                    <Button
                      variant='secondary'
                      onClick={() => this._dropzoneRef.current?.open()}
                      disabled={importMethod === ImportMethod.URL}
                    >
                      Choose file
                    </Button>
                  }
                />
                <p className='kfp-pipeline-form-hint'>You can also drag and drop the file here.</p>
              </DropzoneArea>
            </div>
            <TextField
              id='pipelinePackageUrl'
              label='Package Url'
              multiline
              onChange={this.handleChange('packageUrl')}
              value={packageUrl}
              disabled={importMethod === ImportMethod.LOCAL}
              required={importMethod === ImportMethod.URL}
              hint='URL must be publicly accessible.'
            />
          </section>
          <section className='kfp-pipeline-form-card'>
            <h2>Source</h2>
            <TextField
              id='pipelineVersionCodeSource'
              label='Code Source'
              multiline
              onChange={this.handleChange('codeSourceUrl')}
              value={codeSourceUrl}
            />
          </section>
          <div className='kfp-pipeline-form-actions'>
            <Button
              id='createNewPipelineOrVersionBtn'
              type='submit'
              disabled={!!validationError || isbeingCreated}
              aria-busy={isbeingCreated}
            >
              Create
            </Button>
            <Button
              variant='secondary'
              id='cancelNewPipelineOrVersionBtn'
              onClick={() => this.props.navigate(RoutePage.PIPELINES)}
            >
              Cancel
            </Button>
            {validationError && (
              <p role='status' className='kfp-pipeline-form-error'>
                {validationError}
              </p>
            )}
          </div>
        </form>
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return;
  }

  public async componentDidMount(): Promise<void> {
    this._isMounted = true;
    const urlParser = new URLParser(this.props);
    const pipelineId = urlParser.get(QUERY_PARAMS.pipelineId);
    if (pipelineId) {
      const pipelineResponse = await Apis.pipelineServiceApiV2.getPipeline(pipelineId);
      const currDate = new Date();
      this.setStateSafe(
        {
          pipelineId,
          pipelineName: pipelineResponse.display_name,
          pipeline: pipelineResponse,
          pipelineVersionName:
            pipelineResponse.display_name +
            '-version-at-' +
            currDate.toISOString().toLowerCase().replace(/:/g, '-'),
        },
        () => {
          this._validate();
        },
      );
    } else {
      this._validate();
    }
  }

  public handleChange =
    (name: string) => (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      const value = event.target.value;
      this.setState({ [name]: value } as any, this._validate.bind(this));

      // When pipeline name is changed, we have some special logic
      if (name === 'pipelineName') {
        // Suggest a version name based on pipeline name
        const currDate = new Date();
        this.setState(
          {
            pipelineVersionName:
              value + '-version-at-' + currDate.toISOString().toLowerCase().replace(/:/g, '-'),
          },
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
        pipelineName: (pipeline && pipeline.name) || '',
        pipelineDisplayName: (pipeline && pipeline.display_name) || '',
        pipelineSelectorOpen: false,
        // Suggest a version name based on pipeline name
        pipelineVersionName:
          (pipeline &&
            pipeline.name +
              '-version-at-' +
              currDate.toISOString().toLowerCase().replace(/:/g, '-')) ||
          '',
      },
      () => this._validate(),
    );
  }

  protected _onDropForTest(files: File[]): void {
    if (files.length) {
      this._onDrop(files);
    }
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
            this.state.pipelineDisplayName!,
            this.state.pipelineDescription,
            this.state.file!,
            namespace,
            this.state.codeSourceUrl || undefined,
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
            name: this.state.pipelineName,
            namespace,
          };
          const createPipelineResponse =
            await Apis.pipelineServiceApiV2.createPipeline(newPipeline);
          this.setState({ pipelineId: createPipelineResponse.pipeline_id });
          pipelineVersionResponse = await this._createPipelineVersion(
            createPipelineResponse.pipeline_id!,
          );
        } else {
          pipelineVersionResponse = await this._createPipelineVersion(this.state.pipelineId!);
        }

        // If success, go to pipeline details page of the new version
        this.props.navigate(
          RoutePage.PIPELINE_DETAILS.replace(
            `:${RouteParams.pipelineId}`,
            encodeURIComponent(
              pipelineVersionResponse.pipeline_id!,
            ) /* pipeline id of this version */,
          ).replace(
            `:${RouteParams.pipelineVersionId}`,
            encodeURIComponent(pipelineVersionResponse.pipeline_version_id!),
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

  private async _createPipelineVersion(pipelineId: string): Promise<V2beta1PipelineVersion> {
    if (this.state.importMethod === ImportMethod.LOCAL) {
      if (!this.state.file) {
        throw new Error('File should be selected');
      }
      return Apis.uploadPipelineVersionV2(
        this.state.pipelineVersionName,
        this.state.pipelineVersionDisplayName,
        pipelineId,
        this.state.file,
        this.state.pipelineVersionDescription,
        this.state.codeSourceUrl || undefined,
      );
    } else {
      // this.state.importMethod === ImportMethod.URL
      let newPipeline: V2beta1PipelineVersion = {
        pipeline_id: pipelineId,
        display_name: this.state.pipelineVersionDisplayName,
        name: this.state.pipelineVersionName,
        description: this.state.pipelineVersionDescription,
        package_url: { pipeline_url: this.state.packageUrl },
        code_source_url: this.state.codeSourceUrl || undefined,
      };
      return Apis.pipelineServiceApiV2.createPipelineVersion(pipelineId, newPipeline);
    }
  }

  private _validate(): void {
    // Validate state
    // 3 valid use case for now:
    // (1) new pipeline (and a default version) from local file
    // (2) new pipeline (and a default version) from url
    // (3) new pipeline version (under an existing pipeline) from url
    const { fileName, pipeline, pipelineVersionName, packageUrl, newPipeline, pipelineName } =
      this.state;
    try {
      if (newPipeline) {
        if (!packageUrl && !fileName) {
          throw new Error('Must specify either package url  or file in .yaml, .zip, or .tar.gz');
        }
        if (!pipelineName) {
          throw new Error('Pipeline name is required');
        }
        if (this.props.buildInfo?.pipelineStore === 'kubernetes') {
          if (!getK8sNameRegex().test(pipelineName)) {
            throw new Error(
              'Pipeline name must match Kubernetes naming pattern: lowercase letters, numbers, hyphens, and dots',
            );
          }
        }
      } else {
        if (!pipeline) {
          throw new Error('Pipeline is required');
        }
        if (!pipelineVersionName) {
          throw new Error('Pipeline version name name is required');
        }
        if (pipelineVersionName && pipelineVersionName.length > 100) {
          throw new Error('Pipeline version name must contain no more than 100 characters');
        }
        if (this.props.buildInfo?.pipelineStore === 'kubernetes') {
          if (!getK8sNameRegex().test(pipelineVersionName)) {
            throw new Error(
              'Pipeline version name must match Kubernetes naming pattern: lowercase letters, numbers, hyphens, and dots',
            );
          }
        }
        if (!packageUrl && !fileName) {
          throw new Error('Please specify either package url or file in .yaml, .zip, or .tar.gz');
        }
      }
      this.setState({ validationError: '' });
    } catch (err) {
      this.setState({
        validationError:
          err instanceof Error ? err.message : 'Validation failed for pipeline version.',
      });
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

  private _onDropRejected(): void {
    this.setStateSafe({ dropzoneActive: false, file: null, fileName: '' }, () => {
      this._validate();
    });
    this.props.updateSnackbar({
      autoHideDuration: 5000,
      message: PIPELINE_PACKAGE_REJECT_MESSAGE,
      open: true,
    });
  }
}

const EnhancedNewPipelineVersion: React.FC<PageProps> = (props) => {
  const buildInfo = React.useContext(BuildInfoContext);
  const namespace = React.useContext(NamespaceContext);

  return <NewPipelineVersion {...props} buildInfo={buildInfo} namespace={namespace} />;
};

export default EnhancedNewPipelineVersion;
