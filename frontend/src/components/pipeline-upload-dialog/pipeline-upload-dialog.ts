// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import 'paper-button/paper-button.html';
import 'paper-input/paper-input.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';
import { apiPipeline } from '../../api/pipeline';
import { DialogResult, PopupDialog } from '../popup-dialog/popup-dialog';

import './pipeline-upload-dialog.html';

export interface PipelineUploadDialogResult {
  buttonPressed: DialogResult;
  pipeline: apiPipeline|null;
}

@customElement('pipeline-upload-dialog')
export class PipelineUploadDialog extends Polymer.Element {

  @property({ type: Boolean })
  protected _busy = false;

  @property({ type: String })
  protected _fileName = '';

  @property({ type: String })
  protected _pipelineName = '';

  private _pipelineFile: File|null = null;

  public get popupDialog(): PopupDialog {
    return this.$.dialog as PopupDialog;
  }

  constructor() {
    super();

    document.body.appendChild(this);

    this.addEventListener(
        'iron-overlay-closed',
        () => document.body.removeChild(this));
  }

  public async open(): Promise<PipelineUploadDialogResult> {
    const innerDialogResult = await this.popupDialog.open();
    let pipeline: apiPipeline|null = null;
    // BUTTON1 is 'Upload'
    if (innerDialogResult === DialogResult.BUTTON1) {
      this._busy = true;
      try {
        if (!this._pipelineFile) {
          throw new Error('Somehow upload was clicked without selecting a file.');
        }
        pipeline = await Apis.uploadPipeline(this._pipelineName, this._pipelineFile);
      } catch (err) {
        Utils.showDialog('There was an error uploading the pipeline.', err);
      } finally {
        (this.$.altFileUpload as HTMLInputElement).value = '';
        this._busy = false;
      }
    }
    return {
      buttonPressed: innerDialogResult,
      pipeline,
    };
  }

  @observe('_fileName', '_pipelineName')
  protected _updateUploadButtonState(): void {
    this.popupDialog.button1Element.disabled = !this._fileName || !this._pipelineName;
  }

  protected async _altUpload(): Promise<void> {
    (this.$.altFileUpload as HTMLInputElement).click();
  }

  protected _upload(): void {
    const files = (this.$.altFileUpload as HTMLInputElement).files;

    if (!files || !files.length) {
      return;
    }

    this._pipelineFile = files[0];
    this._fileName = this._pipelineFile.name;
    if (!this._pipelineName) {
      this._pipelineName = this._fileName;
    }
  }
}
