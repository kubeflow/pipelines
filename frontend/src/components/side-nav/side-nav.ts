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

import 'iron-icon/iron-icon.html';
import 'iron-iconset-svg/iron-iconset-svg.html';
import 'paper-button/paper-button.html';
import 'paper-icon-button/paper-icon-button.html';
import 'polymer/polymer.html';
import '../../assets/pipelines-icons.html';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';

import './side-nav.html';

@customElement('side-nav')
export class SideNav extends Polymer.Element {

  @property({ type: String })
  protected page = '';

  @property({ type: Boolean })
  protected _navCollapsed = true;

  @property({ type: String })
  protected _chevronIcon = 'chevron-right';

  public get jobsButton(): PaperButtonElement {
    return this.$.jobsBtn as PaperButtonElement;
  }

  public get pipelinesButton(): PaperButtonElement {
    return this.$.pipelinesBtn as PaperButtonElement;
  }

  public ready(): void {
    super.ready();
  }

  @observe('page')
  protected _updateActiveNavButton(): void {
    this.pipelinesButton.active = this.page.startsWith('pipeline');
    this.jobsButton.active = !this.pipelinesButton.active;
  }

  protected _toggleNavExpand(): void {
    this._chevronIcon = this._navCollapsed ? 'chevron-left' : 'chevron-right';
    this._navCollapsed = !this._navCollapsed;
  }
}
