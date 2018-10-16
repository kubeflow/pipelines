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
import DialogTitle from '@material-ui/core/DialogTitle';
import Dropzone from 'react-dropzone';
import Input from '../atoms/Input';
import { TextFieldProps } from '@material-ui/core/TextField';
import { classes, stylesheet } from 'typestyle';
import { commonCss, padding } from '../Css';

const css = stylesheet({
  dropOverlay: {
    backgroundColor: '#eee',
    border: '2px dashed #aaa',
    borderRadius: 3,
    bottom: 0,
    left: 0,
    margin: 3,
    maxWidth: 450,
    padding: '2.5em 0',
    position: 'absolute',
    right: 0,
    textAlign: 'center',
    top: 0,
    zIndex: 1,
  },
  uploadPipelineName: {
    border: '1px solid #ddd',
    display: 'inline-block',
    height: 30,
    lineHeight: '30px',
    minWidth: 180,
    paddingLeft: 5,
  },
});

interface UploadPipelineDialogProps {
  open: boolean;
  onClose: (name: string, file: File | null) => Promise<boolean>;
}

interface UploadPipelineDialogState {
  busy: boolean;
  dropzoneActive: boolean;
  fileToUpload: File | null;
  uploadPipelineName: string;
}

class UploadPipelineDialog extends React.Component<UploadPipelineDialogProps, UploadPipelineDialogState> {
  private _dropzoneRef = React.createRef<Dropzone & HTMLDivElement>();

  constructor(props: any) {
    super(props);

    this.state = {
      busy: false,
      dropzoneActive: false,
      fileToUpload: null,
      uploadPipelineName: '',
    };
  }

  public render() {
    return (
      <Dialog id='uploadDialog' onClose={() => this._uploadDialogClosed(false)} open={this.props.open}>
        <DialogTitle>Upload and name your pipeline</DialogTitle>
        <Dropzone id='dropZone' disableClick={true} className={padding(20, 'lr')}
          onDrop={this._onDrop.bind(this)} onDragEnter={this._onDropzoneDragEnter.bind(this)}
          onDragLeave={this._onDropzoneDragLeave.bind(this)} style={{ position: 'relative' }}
          ref={this._dropzoneRef}>

          {this.state.dropzoneActive && <div className={css.dropOverlay}>Drop files..</div>}

          <div className={padding(10, 'b')}>Click choose file or just drop it here</div>
          <div className={commonCss.flex}>
            <Button id='chooseFileBtn' onClick={() => this._dropzoneRef.current!.open()}
              className={classes(commonCss.actionButton, commonCss.primaryButton, commonCss.noShrink)}>
              Choose file
              </Button>
            <span className={classes(css.uploadPipelineName, commonCss.ellipsis)}>
              {this.state.fileToUpload && this.state.fileToUpload.name}
            </span>
          </div>

          <Input label='Pipeline name' instance={this} field='uploadPipelineName' />
        </Dropzone>

        <DialogActions>
          <BusyButton id='confirmUploadBtn' onClick={() => this._uploadDialogClosed.bind(this)(true)}
            className={commonCss.actionButton} title='Upload' busy={this.state.busy} />
          <Button id='cancelUploadBtn' onClick={() => this._uploadDialogClosed.bind(this)(false)}
            className={commonCss.actionButton}>Cancel</Button>
        </DialogActions>
      </Dialog>
    );
  }

  public handleChange = (name: string) => (event: any) => {
    this.setState({
      [name]: (event.target as TextFieldProps).value,
    } as any);
  }

  private _onDropzoneDragEnter() {
    this.setState({ dropzoneActive: true });
  }

  private _onDropzoneDragLeave() {
    this.setState({ dropzoneActive: false });
  }

  private _onDrop(files: File[]) {
    this.setState({
      dropzoneActive: false,
      fileToUpload: files[0],
      uploadPipelineName: files[0].name,
    });
  }

  private _uploadDialogClosed(confirmed: boolean) {
    const file = confirmed ? this.state.fileToUpload : null;
    this.setState({ busy: true }, async () => {
      const success = await this.props.onClose(this.state.uploadPipelineName, file);
      if (success) {
        this.setState({
          busy: false,
          dropzoneActive: false,
          fileToUpload: null,
          uploadPipelineName: '',
        });
      } else {
        this.setState({
          busy: false,
        });
      }
    });
  }
}

export default UploadPipelineDialog;
