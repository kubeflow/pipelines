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

/// <reference path="../../bower_components/polymer/types/polymer.d.ts" />

import { customElement, property } from 'polymer-decorators/src/decorators';

@customElement('page-element')
export class PageElement extends Polymer.Element {

  @property({type: String})
  protected _pageError = '';

  @property({type: String})
  protected _pageErrorDetails = '';

  public load(path?: string, queryParams?: {}, data?: {}): void {
    throw new Error('No implementation in abstract class');
  }

  public showPageError(error: string, details?: string): void {
    this._pageError = error;
    if (details) {
      this._pageErrorDetails = details;
    }
  }
}
