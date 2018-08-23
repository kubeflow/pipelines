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

import { customElement, property } from 'polymer-decorators/src/decorators';
import * as Utils from '../../lib/utils';

import 'iron-icon/iron-icon.html';
import 'paper-button/paper-button.html';
import 'polymer/polymer.html';

import './page-error.html';

@customElement('page-error')
export class PageError extends Polymer.Element {
  @property({ type: String })
  public error = '';

  @property({ type: String })
  public details = '';

  @property({ type: Boolean })
  public showButton = true;

  protected _refresh(): void {
    location.reload();
  }

  protected _showDetails(): void {
    Utils.showDialog(this.error, this.details, 'Close');
  }
}
